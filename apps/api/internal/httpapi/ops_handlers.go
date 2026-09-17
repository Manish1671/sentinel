package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/apps/api/internal/ops"
	"github.com/sentinel-dev/sentinel/apps/api/internal/paging"
)

func (s *Server) resolveIncidentID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	item, err := s.incidents.Resolve(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, r, s.log, err)
		return uuid.Nil, false
	}
	return item.ID, true
}

func (s *Server) listInvestigations(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	limit, err := paging.ParseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	items, page, err := s.ops.ListInvestigations(r.Context(), ops.ListFilter{
		IncidentID: r.URL.Query().Get("incident_id"),
		Status:     r.URL.Query().Get("status"),
		Limit:      limit,
		Cursor:     r.URL.Query().Get("cursor"),
	})
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, investigationDTO(item, false))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "page": pageDTO(page)})
}

func (s *Server) getInvestigation(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	item, err := s.ops.GetInvestigation(r.Context(), id)
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": investigationDTO(item, true)})
}

func (s *Server) incidentInvestigations(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	incidentID, ok := s.resolveIncidentID(w, r)
	if !ok {
		return
	}
	limit, err := paging.ParseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	items, page, err := s.ops.ListInvestigations(r.Context(), ops.ListFilter{
		IncidentID: incidentID.String(),
		Status:     r.URL.Query().Get("status"),
		Limit:      limit,
		Cursor:     r.URL.Query().Get("cursor"),
	})
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, investigationDTO(item, false))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "page": pageDTO(page)})
}

func (s *Server) incidentRecommendations(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	incidentID, ok := s.resolveIncidentID(w, r)
	if !ok {
		return
	}
	items, err := s.ops.ListRecommendations(r.Context(), incidentID, r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, recommendationDTO(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *Server) getRecommendation(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	item, err := s.ops.GetRecommendation(r.Context(), id)
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": recommendationDTO(item)})
}

func (s *Server) listRemediations(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	limit, err := paging.ParseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	items, page, err := s.ops.ListRemediations(r.Context(), ops.ListFilter{
		IncidentID: r.URL.Query().Get("incident_id"),
		Status:     r.URL.Query().Get("status"),
		Limit:      limit,
		Cursor:     r.URL.Query().Get("cursor"),
	})
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, remediationDTO(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "page": pageDTO(page)})
}

func (s *Server) getRemediation(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	item, err := s.ops.GetRemediation(r.Context(), id)
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": remediationDTO(item)})
}

func (s *Server) incidentRemediations(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	incidentID, ok := s.resolveIncidentID(w, r)
	if !ok {
		return
	}
	limit, err := paging.ParseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	items, page, err := s.ops.ListRemediations(r.Context(), ops.ListFilter{
		IncidentID: incidentID.String(),
		Status:     r.URL.Query().Get("status"),
		Limit:      limit,
		Cursor:     r.URL.Query().Get("cursor"),
	})
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, remediationDTO(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "page": pageDTO(page)})
}

func (s *Server) incidentAlerts(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	incidentID, ok := s.resolveIncidentID(w, r)
	if !ok {
		return
	}
	items, err := s.ops.ListAlerts(r.Context(), incidentID)
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, map[string]any{
			"id":           item.ID,
			"service_id":   item.ServiceID,
			"service_slug": item.ServiceSlug,
			"detector_id":  item.DetectorID,
			"severity":     item.Severity,
			"status":       item.Status,
			"title":        item.Title,
			"summary":      item.Summary,
			"started_at":   item.StartedAt.UTC().Format(time.RFC3339Nano),
			"labels":       rawJSON(item.Labels),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *Server) listDeployments(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	limit, err := paging.ParseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	items, page, err := s.ops.ListDeployments(r.Context(), ops.ListFilter{
		ServiceID: r.URL.Query().Get("service_id"),
		Limit:     limit,
		Cursor:    r.URL.Query().Get("cursor"),
	})
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, map[string]any{
			"id":           item.ID,
			"service_id":   item.ServiceID,
			"service_slug": item.ServiceSlug,
			"version":      item.Version,
			"git_sha":      item.GitSHA,
			"status":       item.Status,
			"started_at":   item.StartedAt.UTC().Format(time.RFC3339Nano),
			"completed_at": formatTimePtr(item.CompletedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "page": pageDTO(page)})
}

func (s *Server) approveRemediation(w http.ResponseWriter, r *http.Request) {
	s.decideRemediation(w, r, true)
}

func (s *Server) rejectRemediation(w http.ResponseWriter, r *http.Request) {
	s.decideRemediation(w, r, false)
}

func (s *Server) decideRemediation(w http.ResponseWriter, r *http.Request, approve bool) {
	user, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	var body struct {
		Comment string `json:"comment"`
	}
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, r, s.log, err)
			return
		}
	}
	data, err := s.ops.Decide(r.Context(), user, id, approve, sessionToken(r), r.Header.Get("Idempotency-Key"), body.Comment)
	if err != nil {
		writeError(w, r, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rawJSON(data)})
}

func rawJSON(raw json.RawMessage) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	return v
}

func investigationDTO(item ops.Investigation, includeEvidence bool) map[string]any {
	dto := map[string]any{
		"id":                     item.ID,
		"incident_id":            item.IncidentID,
		"status":                 item.Status,
		"requested_by_user_id":   item.RequestedByUserID,
		"model_name":             item.ModelName,
		"model_version":          item.ModelVersion,
		"root_cause_hypothesis":  item.RootCauseHypothesis,
		"reasoning_summary":      item.ReasoningSummary,
		"confidence":             item.Confidence,
		"risk_level":             item.RiskLevel,
		"tool_usage":             rawJSON(item.ToolUsage),
		"error_message":          item.ErrorMessage,
		"requested_at":           item.RequestedAt.UTC().Format(time.RFC3339Nano),
		"started_at":             formatTimePtr(item.StartedAt),
		"completed_at":           formatTimePtr(item.CompletedAt),
	}
	if includeEvidence {
		evidence := make([]map[string]any, 0, len(item.Evidence))
		for _, e := range item.Evidence {
			evidence = append(evidence, map[string]any{
				"id":           e.ID,
				"tool_name":    e.ToolName,
				"source_type":  e.SourceType,
				"summary":      e.Summary,
				"artifact_uri": e.ArtifactURI,
				"source_ref":   e.SourceRef,
				"metadata":     rawJSON(e.Metadata),
				"captured_at":  e.CapturedAt.UTC().Format(time.RFC3339Nano),
			})
		}
		dto["evidence"] = evidence
	}
	return dto
}

func recommendationDTO(item ops.Recommendation) map[string]any {
	return map[string]any{
		"id":                     item.ID,
		"incident_id":            item.IncidentID,
		"investigation_id":       item.InvestigationID,
		"action_type":            item.ActionType,
		"title":                  item.Title,
		"rationale":              item.Rationale,
		"target_service_id":      item.TargetServiceID,
		"parameters":             rawJSON(item.Parameters),
		"confidence":             item.Confidence,
		"risk_level":             item.RiskLevel,
		"required_approval_role": item.RequiredApprovalRole,
		"status":                 item.Status,
	}
}

func remediationDTO(item ops.Remediation) map[string]any {
	var approval any
	if item.Approval != nil {
		approval = map[string]any{
			"id":            item.Approval.ID,
			"decision":      item.Approval.Decision,
			"actor_user_id": item.Approval.ActorUserID,
			"comment":       item.Approval.Comment,
			"decided_at":    formatTimePtr(item.Approval.DecidedAt),
		}
	}
	return map[string]any{
		"id":                   item.ID,
		"incident_id":          item.IncidentID,
		"recommendation_id":    item.RecommendationID,
		"service_id":           item.ServiceID,
		"status":               item.Status,
		"action_type":          item.ActionType,
		"parameters":           rawJSON(item.Parameters),
		"requested_by_user_id": item.RequestedByUserID,
		"attempt_number":       item.AttemptNumber,
		"result_summary":       item.ResultSummary,
		"verification_status":  item.VerificationStatus,
		"verification_details": rawJSON(item.VerificationDetails),
		"error_message":        item.ErrorMessage,
		"started_at":           formatTimePtr(item.StartedAt),
		"completed_at":         formatTimePtr(item.CompletedAt),
		"approval":             approval,
	}
}
