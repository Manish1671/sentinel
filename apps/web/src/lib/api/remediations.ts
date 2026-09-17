import { apiFetch, withQuery } from "@/lib/api/client";
import type { FetchInit, ItemResponse, ListQuery, ListResponse, RemediationDetail } from "@/lib/api/types";

export function listRemediations(
  query: ListQuery & { incident_id?: string; status?: string } = {},
  init: FetchInit = {},
) {
  return apiFetch<ListResponse<RemediationDetail>>(
    withQuery("/api/v1/remediations", {
      incident_id: query.incident_id,
      status: query.status,
      limit: query.limit,
      cursor: query.cursor,
    }),
    { signal: init.signal },
  );
}

export function listIncidentRemediations(incidentId: string, init: FetchInit = {}) {
  return apiFetch<ListResponse<RemediationDetail>>(
    `/api/v1/incidents/${encodeURIComponent(incidentId)}/remediations`,
    { signal: init.signal },
  );
}

export function getRemediation(id: string, init: FetchInit = {}) {
  return apiFetch<ItemResponse<RemediationDetail>>(`/api/v1/remediations/${id}`, { signal: init.signal });
}

export function approveRemediation(
  id: string,
  options: { idempotencyKey: string; comment?: string; signal?: AbortSignal },
) {
  return apiFetch<ItemResponse<RemediationDetail>>(`/api/v1/remediations/${id}/approve`, {
    method: "POST",
    idempotencyKey: options.idempotencyKey,
    signal: options.signal,
    body: { comment: options.comment ?? "" },
  });
}

export function rejectRemediation(
  id: string,
  options: { idempotencyKey: string; comment?: string; signal?: AbortSignal },
) {
  return apiFetch<ItemResponse<RemediationDetail>>(`/api/v1/remediations/${id}/reject`, {
    method: "POST",
    idempotencyKey: options.idempotencyKey,
    signal: options.signal,
    body: { comment: options.comment ?? "" },
  });
}
