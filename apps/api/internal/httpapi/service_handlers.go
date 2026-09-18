package httpapi

import (
	"net/http"

	"github.com/sentinel-dev/sentinel/apps/api/internal/paging"
	"github.com/sentinel-dev/sentinel/apps/api/internal/services"
)

func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	limit, err := paging.ParseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	items, page, err := s.catalog.List(r.Context(), services.ListFilter{
		Environment:  r.URL.Query().Get("environment"),
		HealthStatus: r.URL.Query().Get("health_status"),
		Limit:        limit,
		Cursor:       r.URL.Query().Get("cursor"),
	})
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, map[string]any{
			"id":            item.ID,
			"slug":          item.Slug,
			"name":          item.Name,
			"environment":   item.Environment,
			"health_status":        item.HealthStatus,
			"owner_user_id":        item.OwnerUserID,
			"current_version":      item.CurrentVersion,
			"active_incident_count": item.ActiveIncidentCount,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": data,
		"page": pageDTO(page),
	})
}

func (s *Server) getService(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	item, deps, err := s.catalog.Resolve(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	deployments := make([]map[string]any, 0, len(deps))
	for _, d := range deps {
		deployments = append(deployments, map[string]any{
			"id":           d.ID,
			"version":      d.Version,
			"git_sha":      d.GitSHA,
			"status":       d.Status,
			"started_at":   d.StartedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			"completed_at": formatTimePtr(d.CompletedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":                 item.ID,
			"slug":               item.Slug,
			"name":               item.Name,
			"environment":        item.Environment,
			"description":        item.Description,
			"health_status":         item.HealthStatus,
			"owner_user_id":         item.OwnerUserID,
			"current_version":       item.CurrentVersion,
			"active_incident_count": item.ActiveIncidentCount,
			"recent_deployments":    deployments,
		},
	})
}
