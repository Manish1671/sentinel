import { apiFetch, withQuery } from "@/lib/api/client";
import type {
  AlertDetail,
  FetchInit,
  IncidentDetail,
  IncidentSummary,
  IncidentTimelineEvent,
  ItemResponse,
  ListQuery,
  ListResponse,
} from "@/lib/api/types";

export function listIncidents(
  query: ListQuery & { status?: string; service_id?: string; severity?: string } = {},
  init: FetchInit = {},
) {
  return apiFetch<ListResponse<IncidentSummary>>(
    withQuery("/api/v1/incidents", {
      status: query.status,
      service_id: query.service_id,
      severity: query.severity,
      limit: query.limit,
      cursor: query.cursor,
    }),
    { signal: init.signal },
  );
}

export function getIncident(id: string, init: FetchInit = {}) {
  return apiFetch<ItemResponse<IncidentDetail>>(`/api/v1/incidents/${encodeURIComponent(id)}`, {
    signal: init.signal,
  });
}

export function getIncidentTimeline(id: string, query: ListQuery = {}, init: FetchInit = {}) {
  return apiFetch<ListResponse<IncidentTimelineEvent>>(
    withQuery(`/api/v1/incidents/${encodeURIComponent(id)}/timeline`, {
      limit: query.limit,
      cursor: query.cursor,
    }),
    { signal: init.signal },
  );
}

export function getIncidentAlerts(id: string, init: FetchInit = {}) {
  return apiFetch<ListResponse<AlertDetail>>(`/api/v1/incidents/${encodeURIComponent(id)}/alerts`, {
    signal: init.signal,
  });
}
