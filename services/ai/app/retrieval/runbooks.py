from __future__ import annotations

from typing import Any

import psycopg

from app.tools.common import no_evidence, ok


def search_runbooks(conn: psycopg.Connection, args: dict[str, Any], ctx: dict[str, Any]) -> dict[str, Any]:
    query = (args.get("query") or "").strip()
    limit = int(args.get("limit") or 5)
    service_id = args.get("service_id") or ctx.get("service_id")
    sql = """
        SELECT id, slug, title, failure_class, body, service_id,
               ts_rank(
                 to_tsvector('english', coalesce(title,'') || ' ' || coalesce(body,'') || ' ' || coalesce(failure_class,'')),
                 plainto_tsquery('english', %s)
               ) AS score
        FROM runbooks
        WHERE status = 'published'
    """
    params: list[Any] = [query or "incident"]
    if service_id:
        sql += " AND (service_id = %s OR service_id IS NULL)"
        params.append(service_id)
    sql += " ORDER BY score DESC, updated_at DESC LIMIT %s"
    params.append(limit)
    rows = conn.execute(sql, params).fetchall()
    hits = []
    qwords = {w.lower() for w in query.split() if len(w) > 2}
    for row in rows:
        body = row["body"] or ""
        excerpt = body[:400]
        score = float(row["score"] or 0)
        blob = f"{row['title']} {row['failure_class']} {body}".lower()
        overlap = sum(1 for w in qwords if w in blob)
        score = max(score, overlap / max(len(qwords), 1))
        hits.append(
            {
                "runbook_id": str(row["id"]),
                "slug": row["slug"],
                "title": row["title"],
                "excerpt": excerpt,
                "failure_class": row["failure_class"],
                "score": round(score, 4),
            }
        )
    hits.sort(key=lambda h: h["score"], reverse=True)
    if not hits or all(h["score"] == 0 for h in hits):
        # still return published service runbooks as weak matches
        if hits:
            return ok({"hits": hits[:limit]})
        return no_evidence("no published runbooks matched the query")
    return ok({"hits": hits[:limit]})
