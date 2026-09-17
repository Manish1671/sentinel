package ops

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sentinel-dev/sentinel/apps/api/internal/apierr"
	"github.com/sentinel-dev/sentinel/apps/api/internal/auth"
	"github.com/sentinel-dev/sentinel/apps/api/internal/paging"
)

type Service struct {
	repo     *Repository
	decisions DecisionClient
}

func NewService(repo *Repository, decisions DecisionClient) *Service {
	return &Service{repo: repo, decisions: decisions}
}

func (s *Service) GetInvestigation(ctx context.Context, id uuid.UUID) (Investigation, error) {
	item, err := s.repo.GetInvestigation(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Investigation{}, apierr.NotFound("Investigation not found.")
	}
	return item, err
}

func (s *Service) ListInvestigations(ctx context.Context, filter ListFilter) ([]Investigation, paging.Page, error) {
	incidentID, err := parseOptionalUUID(filter.IncidentID, "incident_id")
	if err != nil {
		return nil, paging.Page{}, err
	}
	if filter.Status != "" && !oneOf(filter.Status, "requested", "running", "completed", "failed") {
		return nil, paging.Page{}, apierr.Validation([]map[string]string{
			apierr.Field("status", "invalid", "status is invalid"),
		})
	}
	cursorTime, cursorID, err := decodeCursor(filter.Cursor)
	if err != nil {
		return nil, paging.Page{}, err
	}
	rows, err := s.repo.ListInvestigations(ctx, filter, incidentID, cursorTime, cursorID)
	if err != nil {
		return nil, paging.Page{}, err
	}
	page := paging.Page{Limit: filter.Limit}
	if len(rows) == filter.Limit {
		last := rows[len(rows)-1]
		page = nextPage(filter.Limit, len(rows), last.RequestedAt, last.ID)
	}
	return rows, page, nil
}

func (s *Service) GetRecommendation(ctx context.Context, id uuid.UUID) (Recommendation, error) {
	item, err := s.repo.GetRecommendation(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Recommendation{}, apierr.NotFound("Recommendation not found.")
	}
	return item, err
}

func (s *Service) ListRecommendations(ctx context.Context, incidentID uuid.UUID, status string) ([]Recommendation, error) {
	if status != "" && !oneOf(status, "proposed", "accepted", "rejected", "superseded") {
		return nil, apierr.Validation([]map[string]string{
			apierr.Field("status", "invalid", "status is invalid"),
		})
	}
	return s.repo.ListRecommendations(ctx, incidentID, status)
}

func (s *Service) GetRemediation(ctx context.Context, id uuid.UUID) (Remediation, error) {
	item, err := s.repo.GetRemediation(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Remediation{}, apierr.NotFound("Remediation not found.")
	}
	return item, err
}

func (s *Service) ListRemediations(ctx context.Context, filter ListFilter) ([]Remediation, paging.Page, error) {
	incidentID, err := parseOptionalUUID(filter.IncidentID, "incident_id")
	if err != nil {
		return nil, paging.Page{}, err
	}
	if filter.Status != "" && !oneOf(filter.Status, "pending_approval", "approved", "rejected", "running", "verifying", "succeeded", "failed") {
		return nil, paging.Page{}, apierr.Validation([]map[string]string{
			apierr.Field("status", "invalid", "status is invalid"),
		})
	}
	cursorTime, cursorID, err := decodeCursor(filter.Cursor)
	if err != nil {
		return nil, paging.Page{}, err
	}
	rows, err := s.repo.ListRemediations(ctx, filter, incidentID, cursorTime, cursorID)
	if err != nil {
		return nil, paging.Page{}, err
	}
	page := paging.Page{Limit: filter.Limit}
	if len(rows) == filter.Limit {
		last := rows[len(rows)-1]
		page = nextPage(filter.Limit, len(rows), last.CreatedAt, last.ID)
	}
	return rows, page, nil
}

func (s *Service) ListAlerts(ctx context.Context, incidentID uuid.UUID) ([]Alert, error) {
	return s.repo.ListAlertsForIncident(ctx, incidentID)
}

func (s *Service) ListDeployments(ctx context.Context, filter ListFilter) ([]Deployment, paging.Page, error) {
	serviceID, err := parseOptionalUUID(filter.ServiceID, "service_id")
	if err != nil {
		return nil, paging.Page{}, err
	}
	cursorTime, cursorID, err := decodeCursor(filter.Cursor)
	if err != nil {
		return nil, paging.Page{}, err
	}
	rows, err := s.repo.ListDeployments(ctx, serviceID, cursorTime, cursorID, filter.Limit)
	if err != nil {
		return nil, paging.Page{}, err
	}
	page := paging.Page{Limit: filter.Limit}
	if len(rows) == filter.Limit {
		last := rows[len(rows)-1]
		page = nextPage(filter.Limit, len(rows), last.StartedAt, last.ID)
	}
	return rows, page, nil
}

func (s *Service) Decide(ctx context.Context, actor auth.User, id uuid.UUID, approve bool, token, idempotencyKey, comment string) (json.RawMessage, error) {
	if !auth.CanApproveRemediations(actor.Role) {
		return nil, apierr.Forbidden()
	}
	if err := validateIdempotencyKey(idempotencyKey); err != nil {
		return nil, err
	}
	if len(comment) > 2000 {
		return nil, apierr.Validation([]map[string]string{
			apierr.Field("comment", "max_length", "comment must be at most 2000 characters"),
		})
	}
	item, err := s.GetRemediation(ctx, id)
	if err != nil {
		return nil, err
	}
	rec, err := s.GetRecommendation(ctx, item.RecommendationID)
	if err != nil {
		return nil, err
	}
	if !auth.MeetsApprovalRole(actor.Role, rec.RequiredApprovalRole) {
		return nil, apierr.Forbidden()
	}
	if s.decisions == nil {
		return nil, apierr.Unavailable("The remediation service is unavailable.")
	}
	return s.decisions.Decide(ctx, id, approve, token, idempotencyKey, comment, RequestIDFrom(ctx))
}

func parseOptionalUUID(raw, field string) (uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.Nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apierr.Validation([]map[string]string{
			apierr.Field(field, "invalid", field+" must be a UUID"),
		})
	}
	return id, nil
}

func decodeCursor(cursor string) (time.Time, uuid.UUID, error) {
	if cursor == "" {
		return time.Time{}, uuid.Nil, nil
	}
	return paging.DecodeTimeID(cursor)
}

func validateIdempotencyKey(key string) error {
	if len(key) < 8 || len(key) > 128 {
		return apierr.Validation([]map[string]string{
			apierr.Field("Idempotency-Key", "invalid", "Idempotency-Key must be 8-128 characters"),
		})
	}
	for _, c := range key {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return apierr.Validation([]map[string]string{
				apierr.Field("Idempotency-Key", "invalid", "Idempotency-Key contains invalid characters"),
			})
		}
	}
	return nil
}

func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}
