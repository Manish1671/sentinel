import { apiFetch, withQuery } from "@/lib/api/client";
import type { FetchInit, InvestigationDetail, ItemResponse, ListQuery, ListResponse } from "@/lib/api/types";

export function listInvestigations(
  query: ListQuery & { incident_id?: string; status?: string } = {},
  init: FetchInit = {},
) {
  return apiFetch<ListResponse<InvestigationDetail>>(
    withQuery("/api/v1/investigations", {
      incident_id: query.incident_id,
      status: query.status,
      limit: query.limit,
      cursor: query.cursor,
    }),
    { signal: init.signal },
  );
}

export function listIncidentInvestigations(incidentId: string, query: ListQuery = {}, init: FetchInit = {}) {
  return apiFetch<ListResponse<InvestigationDetail>>(
    withQuery(`/api/v1/incidents/${encodeURIComponent(incidentId)}/investigations`, {
      limit: query.limit,
      cursor: query.cursor,
    }),
    { signal: init.signal },
  );
}

export function getInvestigation(id: string, init: FetchInit = {}) {
  return apiFetch<ItemResponse<InvestigationDetail>>(`/api/v1/investigations/${id}`, { signal: init.signal });
}

export function requestInvestigation(
  incidentId: string,
  options: { idempotencyKey: string; token?: string; timeWindow?: { from: string; to: string } },
) {
  return apiFetch<ItemResponse<InvestigationDetail>>(`/api/v1/incidents/${incidentId}/investigations`, {
    method: "POST",
    idempotencyKey: options.idempotencyKey,
    token: options.token,
    body: options.timeWindow ? { time_window: options.timeWindow } : {},
  });
}
