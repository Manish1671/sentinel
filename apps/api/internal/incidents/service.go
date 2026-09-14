package incidents

import (
	"context"
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
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Incident, paging.Page, error) {
	serviceID, err := validateList(filter)
	if err != nil {
		return nil, paging.Page{}, err
	}
	var cursorTime time.Time
	var cursorID uuid.UUID
	if filter.Cursor != "" {
		t, id, err := paging.DecodeTimeID(filter.Cursor)
		if err != nil {
			return nil, paging.Page{}, err
		}
		cursorTime, cursorID = t, id
	}
	rows, err := s.repo.List(ctx, filter, serviceID, cursorTime, cursorID)
	if err != nil {
		return nil, paging.Page{}, err
	}
	return rows, nextListPage(filter.Limit, rows), nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Incident, error) {
	in, err := s.repo.Get(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Incident{}, apierr.NotFound("Incident not found.")
	}
	return in, err
}

func (s *Service) Timeline(ctx context.Context, incidentID uuid.UUID, limit int, cursor string) ([]TimelineEvent, paging.Page, error) {
	if _, err := s.Get(ctx, incidentID); err != nil {
		return nil, paging.Page{}, err
	}
	var cursorTime time.Time
	var cursorID uuid.UUID
	if cursor != "" {
		t, id, err := paging.DecodeTimeID(cursor)
		if err != nil {
			return nil, paging.Page{}, err
		}
		cursorTime, cursorID = t, id
	}
	rows, err := s.repo.Timeline(ctx, incidentID, limit, cursorTime, cursorID)
	if err != nil {
		return nil, paging.Page{}, err
	}
	return rows, nextTimelinePage(limit, rows), nil
}

func (s *Service) Create(ctx context.Context, actor auth.User, idempotencyKey string, in CreateInput) (Incident, bool, error) {
	if !auth.CanWriteIncidents(actor.Role) {
		return Incident{}, false, apierr.Forbidden()
	}
	if err := validateIdempotencyKey(idempotencyKey); err != nil {
		return Incident{}, false, err
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Summary = strings.TrimSpace(in.Summary)
	if err := validateCreate(in); err != nil {
		return Incident{}, false, err
	}
	exists, err := s.repo.ServiceExists(ctx, in.ServiceID)
	if err != nil {
		return Incident{}, false, err
	}
	if !exists {
		return Incident{}, false, apierr.NotFound("Service not found.")
	}
	if len(in.AlertIDs) > 0 {
		n, err := s.repo.AlertsForService(ctx, in.ServiceID, in.AlertIDs)
		if err != nil {
			return Incident{}, false, err
		}
		if n != len(in.AlertIDs) {
			return Incident{}, false, apierr.Validation([]map[string]string{
				apierr.Field("alert_ids", "invalid", "one or more alerts were not found for this service"),
			})
		}
	}
	hash, err := requestHash(in)
	if err != nil {
		return Incident{}, false, err
	}
	created, isNew, err := s.repo.Create(ctx, actor.ID, idempotencyKey, hash, in)
	if errors.Is(err, errIdempotencyConflict) {
		return Incident{}, false, apierr.IdempotencyConflict()
	}
	return created, isNew, err
}
