package httpapi

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/apps/api/internal/apierr"
	"github.com/sentinel-dev/sentinel/apps/api/internal/incidents"
	"github.com/sentinel-dev/sentinel/apps/api/internal/paging"
)

func (s *Server) listIncidents(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	limit, err := paging.ParseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	items, page, err := s.incidents.List(r.Context(), incidents.ListFilter{
		Status:    r.URL.Query().Get("status"),
		ServiceID: r.URL.Query().Get("service_id"),
		Severity:  r.URL.Query().Get("severity"),
		Limit:     limit,
		Cursor:    r.URL.Query().Get("cursor"),
	})
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, incidentListDTO(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "page": pageDTO(page)})
}

func (s *Server) getIncident(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	item, err := s.incidents.Resolve(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": incidentDetailDTO(item)})
}

func (s *Server) incidentTimeline(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	item, err := s.incidents.Resolve(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	limit, err := paging.ParseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	items, page, err := s.incidents.Timeline(r.Context(), item.ID, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, ev := range items {
		data = append(data, map[string]any{
			"id":            ev.ID,
			"kind":          ev.Kind,
			"event_type":    ev.Kind,
			"occurred_at":   ev.OccurredAt.UTC().Format(time.RFC3339Nano),
			"timestamp":     ev.OccurredAt.UTC().Format(time.RFC3339Nano),
			"actor_user_id": ev.ActorUserID,
			"source":        ev.Source,
			"summary":       ev.Summary,
			"metadata":      ev.Metadata,
			"payload":       ev.Payload,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "page": pageDTO(page)})
}

func (s *Server) createIncident(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var body struct {
		ServiceID string   `json:"service_id"`
		Title     string   `json:"title"`
		Summary   string   `json:"summary"`
		Severity  string   `json:"severity"`
		AlertIDs  []string `json:"alert_ids"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, s.log, err)
		return
	}
	serviceID, err := uuid.Parse(body.ServiceID)
	if err != nil {
		writeError(w, r, s.log, apierr.Validation([]map[string]string{
			apierr.Field("service_id", "invalid", "service_id must be a UUID"),
		}))
		return
	}
	var alertIDs []uuid.UUID
	for _, raw := range body.AlertIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, r, s.log, apierr.Validation([]map[string]string{
				apierr.Field("alert_ids", "invalid", "alert_ids must be UUIDs"),
			}))
			return
		}
		alertIDs = append(alertIDs, id)
	}
	item, created, err := s.incidents.Create(r.Context(), user, r.Header.Get("Idempotency-Key"), incidents.CreateInput{
		ServiceID: serviceID,
		Title:     body.Title,
		Summary:   body.Summary,
		Severity:  body.Severity,
		AlertIDs:  alertIDs,
	})
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{"data": incidentDetailDTO(item)})
}
