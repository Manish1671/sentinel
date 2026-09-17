import { apiFetch, withQuery } from "@/lib/api/client";
import type {
  FetchInit,
  ItemResponse,
  ListQuery,
  ListResponse,
  ServiceDetail,
  ServiceSummary,
} from "@/lib/api/types";

export function listServices(
  query: ListQuery & { environment?: string; health_status?: string } = {},
  init: FetchInit = {},
) {
  return apiFetch<ListResponse<ServiceSummary>>(
    withQuery("/api/v1/services", {
      environment: query.environment,
      health_status: query.health_status,
      limit: query.limit,
      cursor: query.cursor,
    }),
    { signal: init.signal },
  );
}

export function getService(id: string, init: FetchInit = {}) {
  return apiFetch<ItemResponse<ServiceDetail>>(`/api/v1/services/${id}`, { signal: init.signal });
}
