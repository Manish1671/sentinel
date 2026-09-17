import { apiFetch, withQuery } from "@/lib/api/client";
import type { FetchInit, ItemResponse, ListResponse, RecommendationDetail } from "@/lib/api/types";

export function listRecommendations(incidentId: string, status?: string, init: FetchInit = {}) {
  return apiFetch<ListResponse<RecommendationDetail>>(
    withQuery(`/api/v1/incidents/${encodeURIComponent(incidentId)}/recommendations`, { status }),
    { signal: init.signal },
  );
}

export function getRecommendation(id: string, init: FetchInit = {}) {
  return apiFetch<ItemResponse<RecommendationDetail>>(`/api/v1/recommendations/${id}`, { signal: init.signal });
}
