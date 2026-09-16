from __future__ import annotations

from typing import Any

import psycopg

from app.tools.common import no_evidence, ok


def search_previous_incidents(conn: psycopg.Connection, args: dict[str, Any], ctx: dict[str, Any]) -> dict[str, Any]:
    query = (args.get("query") or "").strip()
    limit = int(args.get("limit") or 5)
    service_id = args.get("service_id") or ctx.get("service_id")
    current_id = ctx.get("incident_id")
    sql = """
        SELECT i.id, i.reference, i.title, i.summary, i.severity::text, i.resolved_at,
               ts_rank(
                 to_tsvector('english', coalesce(i.title,'') || ' ' || coalesce(i.summary,'')),
                 plainto_tsquery('english', %s)
               ) AS score
        FROM historical_incidents i
        WHERE 1=1
    """
    params: list[Any] = [query or "incident"]
    if service_id:
        sql += " AND i.service_id = %s"
        params.append(service_id)
    if current_id:
        sql += " AND i.id <> %s"
        params.append(current_id)
    sql += " ORDER BY score DESC, i.detected_at DESC LIMIT %s"
    params.append(limit)
    rows = conn.execute(sql, params).fetchall()
    hits = []
    qwords = {w.lower() for w in query.split() if len(w) > 2}
    for row in rows:
        blob = f"{row['title']} {row.get('summary') or ''}".lower()
        overlap = sum(1 for w in qwords if w in blob)
        score = max(float(row["score"] or 0), overlap / max(len(qwords), 1))
        hits.append(
            {
                "incident_id": str(row["id"]),
                "reference": row["reference"],
                "title": row["title"],
                "severity": row["severity"],
                "resolved_at": row["resolved_at"].isoformat() if row["resolved_at"] else None,
                "score": round(score, 4),
                "supporting_only": True,
            }
        )
    hits.sort(key=lambda h: h["score"], reverse=True)
    if not hits:
        return no_evidence("no historical incidents matched")
    return ok({"hits": hits[:limit]})
