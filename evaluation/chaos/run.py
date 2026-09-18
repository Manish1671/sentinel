"""End-to-end chaos/evaluation runner. Uses live Sentinel HTTP APIs when healthy."""

from __future__ import annotations

import argparse
import json
import os
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any
from uuid import uuid4

from evaluation.assertions import add_assertion, ai_properties, detectors_present, mark_category
from evaluation.chaos.client import HTTPClient, load_scenarios
from evaluation.chaos.inject import PAYMENTS_ID, emit_payments, healthy_cleanup
from evaluation.model import compute_durations, iso, new_record, parse_iso, stage, utcnow
from evaluation.runner import evaluate_result, load_case

ROOT = Path(__file__).resolve().parents[1]
PASSWORD = "sentinel-dev"
VIEWER = "riley.park@sentinel.dev"
RESPONDER = "sam.okonkwo@sentinel.dev"
ADMIN = "maya.chen@sentinel.dev"


class Runner:
    def __init__(self, args: argparse.Namespace) -> None:
        self.args = args
        self.http = HTTPClient(timeout=args.timeout)
        self.api = args.api_url.rstrip("/")
        self.ingestion = args.ingestion_url.rstrip("/")
        self.ai = args.ai_url.rstrip("/")
        self.tokens: dict[str, str] = {}

    def health(self) -> dict[str, bool]:
        checks = {
            "ingestion": self._ok(self.ingestion + "/health"),
            "detection": self._ok(self.args.detection_url.rstrip("/") + "/health"),
            "incident": self._ok(self.args.incident_url.rstrip("/") + "/health"),
            "ai": self._ok(self.ai + "/health"),
            "remediation": self._ok(self.args.remediation_url.rstrip("/") + "/health"),
            "api": self._ok(self.api + "/health"),
        }
        return checks

    def _ok(self, url: str) -> bool:
        try:
            code, data = self.http.request("GET", url)
            return code == 200 and isinstance(data, dict) and data.get("status") == "ok"
        except Exception:
            return False

    def login(self, email: str) -> str:
        if email in self.tokens:
            return self.tokens[email]
        code, data = self.http.request(
            "POST",
            self.api + "/api/v1/auth/login",
            {"email": email, "password": PASSWORD},
        )
        if code != 200 or not isinstance(data, dict):
            raise RuntimeError(f"login {email} failed: {code} {data}")
        token = data["data"]["token"]
        self.tokens[email] = token
        return token

    def auth(self, email: str) -> dict[str, str]:
        return {"Authorization": "Bearer " + self.login(email)}

    def list_incidents(self) -> list[dict[str, Any]]:
        code, data = self.http.request(
            "GET",
            self.api + "/api/v1/incidents?service_id=" + PAYMENTS_ID + "&limit=50",
            headers=self.auth(RESPONDER),
        )
        if code != 200 or not isinstance(data, dict):
            raise RuntimeError(f"list incidents: {code} {data}")
        return list(data.get("data") or [])

    def incident_alerts(self, incident_id: str) -> list[dict[str, Any]]:
        code, data = self.http.request(
            "GET",
            self.api + f"/api/v1/incidents/{incident_id}/alerts",
            headers=self.auth(RESPONDER),
        )
        if code != 200 or not isinstance(data, dict):
            return []
        return list(data.get("data") or [])

    def find_payments_incident(self) -> dict[str, Any] | None:
        items = self.list_incidents()
        active = [i for i in items if i.get("status") not in ("resolved", "closed")]
        pool = active or items
        return pool[0] if pool else None

    def wait_for_detector(self, detector: str, started: datetime, timeout: float) -> tuple[dict | None, dict | None]:
        deadline = time.time() + timeout
        last_inc = None
        while time.time() < deadline:
            last_inc = self.find_payments_incident()
            if last_inc:
                for alert in self.incident_alerts(last_inc["id"]):
                    if alert.get("detector_id") != detector:
                        continue
                    started_at = parse_iso(alert.get("started_at"))
                    if started_at and started_at >= started.replace(tzinfo=timezone.utc) - timedelta(seconds=5):
                        return last_inc, alert
                    # open alert for this rule is sufficient if fingerprint already existed
                    if alert.get("status") in ("open", "acknowledged"):
                        return last_inc, alert
            time.sleep(1.5)
        return last_inc, None

    def request_investigation(self, incident_id: str) -> dict[str, Any]:
        code, data = self.http.request(
            "POST",
            self.ai + f"/api/v1/incidents/{incident_id}/investigations",
            {},
        )
        if code not in (200, 201) or not isinstance(data, dict):
            raise RuntimeError(f"request investigation: {code} {data}")
        return data.get("data") or data

    def wait_investigation(self, investigation_id: str, timeout: float) -> dict[str, Any] | None:
        deadline = time.time() + timeout
        while time.time() < deadline:
            code, data = self.http.request(
                "GET",
                self.api + f"/api/v1/investigations/{investigation_id}",
                headers=self.auth(RESPONDER),
            )
            if code == 200 and isinstance(data, dict):
                row = data.get("data") or {}
                if row.get("status") in ("completed", "failed"):
                    return row
            time.sleep(1.5)
        return None

    def run_scenario(self, spec: dict[str, Any]) -> dict[str, Any]:
        record = new_record(spec["id"], spec.get("expected", {}).get("service_slug", "payments-api"))
        record["expected"] = spec.get("expected") or {}
        record["started_at"] = iso(utcnow())
        record["timestamps"]["scenario_started_at"] = record["started_at"]
        try:
            required = spec.get("requires_fault") or []
            if required and not all(os.getenv(k, "").lower() == "true" for k in required):
                record["execution"] = "not_executed"
                record["passed"] = None
                record["failure_reason"] = (
                    "fault flags not enabled on this process: " + ", ".join(required)
                    + ". Covered by in-process tests; live compose was not mutated."
                )
                mark_category(record, "failure_handling_correctness", None, record["failure_reason"])
                return record
            kind = (spec.get("inject") or {}).get("kind")
            if kind in ("telemetry", "telemetry_replay"):
                self._run_telemetry(spec, record)
            elif spec["id"] == "CHAOS-005":
                self._run_unauthorized(spec, record)
            elif spec["id"] == "CHAOS-008":
                self._run_idempotency(spec, record)
            else:
                record["execution"] = "not_executed"
                record["failure_reason"] = "no live handler for this scenario"
            if spec.get("cleanup") and "healthy" in spec.get("cleanup", ""):
                healthy_cleanup(self.http, self.ingestion)
        except Exception as exc:
            record["execution"] = "failed"
            record["passed"] = False
            record["failure_reason"] = str(exc)
        record["completed_at"] = iso(utcnow())
        compute_durations(record)
        if record["passed"] is None and record["execution"] not in ("not_executed",):
            record["passed"] = all(a["passed"] for a in record.get("assertions") or [] if a.get("passed") is not None)
            if record.get("assertions") and record["passed"] is False and not record.get("failure_reason"):
                record["failure_reason"] = "one or more assertions failed"
        return record

    def _run_telemetry(self, spec: dict[str, Any], record: dict[str, Any]) -> None:
        inj = spec["inject"]
        expected = spec["expected"]
        t0 = utcnow()
        event_id = str(uuid4()) if spec["inject"]["kind"] == "telemetry_replay" else None
        first = emit_payments(
            self.http,
            self.ingestion,
            latency_ms=float(inj["latency_ms"]),
            error_rate=float(inj["error_rate"]),
            db_utilization=float(inj["db_utilization"]),
            version=inj.get("version"),
            log_error=bool(inj.get("log_error")),
            event_id=event_id,
        )
        record["timestamps"]["telemetry_emitted_at"] = first["emitted_at"]
        record["execution"] = "live_http"
        if spec["inject"]["kind"] == "telemetry_replay":
            second = emit_payments(
                self.http,
                self.ingestion,
                latency_ms=float(inj["latency_ms"]),
                error_rate=float(inj["error_rate"]),
                db_utilization=float(inj["db_utilization"]),
                version=None,
                log_error=False,
                event_id=event_id,
            )
            replayed = any(e.get("replayed") or e.get("code") == 200 for e in second["events"] if e.get("path") != "log")
            add_assertion(record, "ingest_replayed", replayed, "second post with same event_id")
            mark_category(record, "idempotency_correctness", replayed, "ingestion replay")
        detector = expected["detectors"][0]
        inc, alert = self.wait_for_detector(detector, t0, self.args.wait)
        ok_det, missing = detectors_present([alert] if alert else [], expected["detectors"])
        add_assertion(record, "detection_rule", ok_det, f"missing={missing}")
        mark_category(record, "detection_correctness", ok_det, detector)
        record["detection_result"] = stage("observed" if alert else "missing", ok_det, detector)
        if alert:
            emitted = parse_iso(record["timestamps"].get("telemetry_emitted_at"))
            started_at = parse_iso(alert.get("started_at"))
            if emitted and started_at and started_at >= emitted - timedelta(seconds=30):
                record["timestamps"]["alert_created_at"] = alert.get("started_at")
            else:
                record["actual"]["alert_reused_open"] = True
            record["actual"]["alert"] = alert
        if inc:
            record["incident_result"] = stage("observed", inc.get("service_slug") == "payments-api", inc.get("reference"))
            detected = parse_iso(inc.get("detected_at"))
            emitted = parse_iso(record["timestamps"].get("telemetry_emitted_at"))
            if emitted and detected and detected >= emitted - timedelta(seconds=30):
                record["timestamps"]["incident_created_at"] = inc.get("detected_at")
            record["actual"]["incident"] = {"id": inc.get("id"), "reference": inc.get("reference"), "status": inc.get("status")}
            add_assertion(record, "incident_service", inc.get("service_slug") == "payments-api", inc.get("service_slug") or "")
            mark_category(record, "correlation_correctness", inc.get("service_slug") == "payments-api", inc.get("reference"))
            if spec["inject"]["kind"] == "telemetry_replay" and alert:
                alerts = self.incident_alerts(inc["id"])
                ids = [a["id"] for a in alerts if a.get("detector_id") == detector]
                unique = len(ids) == len(set(ids))
                add_assertion(record, "no_duplicate_alert_rows", unique, f"count={len(ids)}")
        else:
            add_assertion(record, "incident_present", False, "no payments-api incident")
            mark_category(record, "correlation_correctness", False, "missing incident")
        if expected.get("investigate") and inc:
            self._investigate(inc["id"], record)

    def _investigate(self, incident_id: str, record: dict[str, Any]) -> None:
        started = utcnow()
        record["timestamps"]["investigation_started_at"] = iso(started)
        created = self.request_investigation(incident_id)
        inv_id = created.get("id")
        row = self.wait_investigation(inv_id, self.args.wait) if inv_id else None
        if not row:
            add_assertion(record, "investigation_complete", False, "timeout")
            record["investigation_result"] = stage("timeout", False, inv_id)
            mark_category(record, "investigation_completion", False, "timeout")
            return
        record["timestamps"]["investigation_completed_at"] = row.get("completed_at")
        ok = row.get("status") == "completed"
        add_assertion(record, "investigation_complete", ok, row.get("status") or "")
        record["investigation_result"] = stage(row.get("status"), ok, inv_id)
        mark_category(record, "investigation_completion", ok, row.get("status"))
        evidence = row.get("evidence") or []
        result = {
            "root_cause_hypothesis": row.get("root_cause_hypothesis"),
            "reasoning_summary": row.get("reasoning_summary") or "",
            "confidence": row.get("confidence"),
            "recommendations": [],
            "evidence_refs": [
                {"evidence_id": e.get("id"), "source_type": e.get("source_type"), "summary": e.get("summary")}
                for e in evidence
            ],
        }
        code, recs = self.http.request(
            "GET",
            self.api + f"/api/v1/incidents/{incident_id}/recommendations",
            headers=self.auth(RESPONDER),
        )
        rec_list = recs.get("data") if isinstance(recs, dict) else []
        result["recommendations"] = rec_list or []
        props = ai_properties(result, evidence)
        add_assertion(record, "ai_schema", props["passed"], json.dumps({k: props[k] for k in ("schema_ok", "confidence_in_range", "reasoning_separated")}))
        mark_category(record, "evidence_grounding", props["evidence_grounded"], str(props.get("cited_unknown_evidence")))
        mark_category(record, "recommendation_validity", props["has_recommendation"], str(props.get("actions")))
        record["recommendation_result"] = stage("observed", bool(rec_list), str(len(rec_list)))
        if rec_list:
            record["timestamps"]["recommendation_created_at"] = rec_list[0].get("created_at")
        case_path = ROOT / "cases" / "payments_api_1180.json"
        if case_path.is_file() and ok:
            scored = evaluate_result(load_case(case_path), result)
            record["actual"]["keyword_eval"] = scored

    def _run_unauthorized(self, spec: dict[str, Any], record: dict[str, Any]) -> None:
        record["execution"] = "live_http"
        rem_id = spec["expected"]["remediation_id"]
        code, data = self.http.request(
            "POST",
            self.api + f"/api/v1/remediations/{rem_id}/approve",
            {"comment": "chaos unauthorized"},
            {**self.auth(VIEWER), "Idempotency-Key": "chaos-unauth-" + uuid4().hex[:12]},
        )
        ok = code == 403
        add_assertion(record, "viewer_forbidden", ok, f"status={code}")
        mark_category(record, "approval_enforcement", ok, str(code))
        record["approval_result"] = stage("rejected" if ok else "unexpected", ok, str(code))
        recode, after = self.http.request(
            "GET",
            self.api + f"/api/v1/remediations/{rem_id}",
            headers=self.auth(RESPONDER),
        )
        status = (after.get("data") or {}).get("status") if isinstance(after, dict) else None
        add_assertion(record, "remediation_not_executed", recode == 200 and status == "pending_approval", str(status))
        mark_category(record, "remediation_correctness", status == "pending_approval", status)
        record["actual"]["unauthorized"] = {"http_status": code, "remediation_status": status}

    def _run_idempotency(self, spec: dict[str, Any], record: dict[str, Any]) -> None:
        record["execution"] = "live_http"
        key = "chaos-idemp-" + uuid4().hex[:16]
        body = {
            "service_id": "22222222-2222-4222-8222-222222222225",
            "title": "Chaos idempotency probe",
            "severity": "low",
        }
        headers = {**self.auth(RESPONDER), "Idempotency-Key": key}
        c1, d1 = self.http.request("POST", self.api + "/api/v1/incidents", body, headers)
        c2, d2 = self.http.request("POST", self.api + "/api/v1/incidents", body, headers)
        id1 = (d1.get("data") or {}).get("id") if isinstance(d1, dict) else None
        id2 = (d2.get("data") or {}).get("id") if isinstance(d2, dict) else None
        ok = c1 in (200, 201) and c2 == 200 and id1 and id1 == id2
        add_assertion(record, "same_incident_id", bool(ok), f"{c1}/{c2} {id1} {id2}")
        mark_category(record, "idempotency_correctness", bool(ok), key)
        record["actual"]["idempotency"] = {"first": c1, "second": c2, "id": id1}


def write_outputs(out_dir: Path, records: list[dict[str, Any]]) -> Path:
    out_dir.mkdir(parents=True, exist_ok=True)
    run_id = records[0]["run_id"] if records else "empty"
    bundle = {
        "run_id": run_id,
        "completed_at": iso(utcnow()),
        "scenarios": records,
        "passed": all(r.get("passed") is True for r in records if r.get("execution") not in ("not_executed",)),
        "executed": [r["scenario_id"] for r in records if r.get("execution") not in ("not_executed", "not_started")],
        "not_executed": [r["scenario_id"] for r in records if r.get("execution") == "not_executed"],
    }
    json_path = out_dir / f"{run_id}.json"
    md_path = out_dir / f"{run_id}.md"
    json_path.write_text(json.dumps(bundle, indent=2), encoding="utf-8")
    lines = [
        f"# Evaluation run `{run_id}`",
        "",
        f"Executed: {', '.join(bundle['executed']) or '(none)'}",
        f"Not executed: {', '.join(bundle['not_executed']) or '(none)'}",
        "",
    ]
    for r in records:
        lines.append(f"## {r['scenario_id']}")
        lines.append(f"- execution: {r.get('execution')}")
        lines.append(f"- pass: {r.get('passed')}")
        lines.append(f"- expected: `{json.dumps(r.get('expected'))}`")
        lines.append(f"- actual: `{json.dumps(r.get('actual'))}`")
        lines.append(f"- durations_ms: `{json.dumps(r.get('durations_ms'))}`")
        if r.get("failure_reason"):
            lines.append(f"- failure: {r['failure_reason']}")
        lines.append("")
    md_path.write_text("\n".join(lines), encoding="utf-8")
    return json_path


def main() -> int:
    parser = argparse.ArgumentParser(description="Run Sentinel chaos/evaluation scenarios")
    parser.add_argument("--api-url", default=os.getenv("API_URL", "http://localhost:8080"))
    parser.add_argument("--ingestion-url", default=os.getenv("INGESTION_URL", "http://localhost:8090"))
    parser.add_argument("--detection-url", default=os.getenv("DETECTION_URL", "http://localhost:8091"))
    parser.add_argument("--incident-url", default=os.getenv("INCIDENT_URL", "http://localhost:8092"))
    parser.add_argument("--ai-url", default=os.getenv("AI_SERVICE_URL", "http://localhost:8000"))
    parser.add_argument("--remediation-url", default=os.getenv("REMEDIATION_URL", "http://localhost:8093"))
    parser.add_argument("--scenario", action="append", dest="scenarios")
    parser.add_argument("--wait", type=float, default=45.0)
    parser.add_argument("--timeout", type=float, default=15.0)
    parser.add_argument("--out", default=str(ROOT / "results"))
    args = parser.parse_args()
    runner = Runner(args)
    health = runner.health()
    specs = load_scenarios(ROOT)
    if args.scenarios:
        wanted = set(args.scenarios)
        specs = [s for s in specs if s["id"] in wanted]
    records = []
    if not all(health.values()):
        print("health:", json.dumps(health))
        for spec in specs:
            rec = new_record(spec["id"])
            rec["execution"] = "not_executed"
            rec["failure_reason"] = "stack unhealthy: " + json.dumps(health)
            rec["expected"] = spec.get("expected")
            records.append(rec)
        path = write_outputs(Path(args.out), records)
        print("wrote", path)
        return 2
    print("health ok:", json.dumps(health))
    for spec in specs:
        print("running", spec["id"], spec["name"])
        rec = runner.run_scenario(spec)
        print(" ", rec["id"] if False else rec["scenario_id"], "passed=", rec.get("passed"), "exec=", rec.get("execution"))
        records.append(rec)
    path = write_outputs(Path(args.out), records)
    print("wrote", path)
    live = [r for r in records if r.get("execution") not in ("not_executed",)]
    if live and any(r.get("passed") is False for r in live):
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
