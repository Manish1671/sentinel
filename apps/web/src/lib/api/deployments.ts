import { apiFetch, withQuery } from "@/lib/api/client";
import type { DeploymentSummary, FetchInit, ListQuery, ListResponse } from "@/lib/api/types";

export function listDeployments(
  query: ListQuery & { service_id?: string } = {},
  init: FetchInit = {},
) {
  return apiFetch<ListResponse<DeploymentSummary>>(
    withQuery("/api/v1/deployments", {
      service_id: query.service_id,
      limit: query.limit,
      cursor: query.cursor,
    }),
    { signal: init.signal },
  );
}
