package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sentinel-dev/sentinel/apps/api/internal/apierr"
	"github.com/sentinel-dev/sentinel/apps/api/internal/paging"
)

type ServiceRecord struct {
	ID                  uuid.UUID
	Slug                string
	Name                string
	Environment         string
	Description         string
	OwnerUserID         *uuid.UUID
	HealthStatus        string
	CurrentVersion      *string
	ActiveIncidentCount int
}

type Deployment struct {
	ID          uuid.UUID
	Version     string
	GitSHA      *string
	Status      string
	StartedAt   time.Time
	CompletedAt *time.Time
}

type ListFilter struct {
	Environment  string
	HealthStatus string
	Limit        int
	Cursor       string
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]ServiceRecord, paging.Page, error) {
	if filter.Environment != "" && !oneOf(filter.Environment, "production", "staging", "development") {
		return nil, paging.Page{}, apierr.Validation([]map[string]string{
			apierr.Field("environment", "invalid", "environment is invalid"),
		})
	}
	if filter.HealthStatus != "" && !oneOf(filter.HealthStatus, "healthy", "degraded", "unhealthy", "unknown") {
		return nil, paging.Page{}, apierr.Validation([]map[string]string{
			apierr.Field("health_status", "invalid", "health_status is invalid"),
		})
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

	rows, err := s.repo.list(ctx, filter, cursorTime, cursorID)
	if err != nil {
		return nil, paging.Page{}, err
	}
	page := paging.Page{Limit: filter.Limit}
	if len(rows) == filter.Limit {
		last := rows[len(rows)-1]
		c := paging.EncodeTimeID(time.Time{}, last.ID)
		// catalog cursor is slug/id based; encode zero time + id is unique enough with slug order
		_ = last
		c = paging.EncodeTimeID(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), last.ID)
		page.NextCursor = &c
	}
	return rows, page, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (ServiceRecord, []Deployment, error) {
	rec, err := s.repo.get(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ServiceRecord{}, nil, apierr.NotFound("Service not found.")
		}
		return ServiceRecord{}, nil, err
	}
	deps, err := s.repo.recentDeployments(ctx, rec.ID, 5)
	if err != nil {
		return ServiceRecord{}, nil, err
	}
	return rec, deps, nil
}

func (s *Service) Resolve(ctx context.Context, idOrSlug string) (ServiceRecord, []Deployment, error) {
	if id, err := uuid.Parse(idOrSlug); err == nil {
		return s.Get(ctx, id)
	}
	rec, err := s.repo.getBySlug(ctx, idOrSlug)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ServiceRecord{}, nil, apierr.NotFound("Service not found.")
		}
		return ServiceRecord{}, nil, err
	}
	deps, err := s.repo.recentDeployments(ctx, rec.ID, 5)
	if err != nil {
		return ServiceRecord{}, nil, err
	}
	return rec, deps, nil
}

func (r *Repository) list(ctx context.Context, filter ListFilter, cursorTime time.Time, cursorID uuid.UUID) ([]ServiceRecord, error) {
	q := `
		SELECT s.id, s.slug, s.name, s.environment::text, s.description, s.owner_user_id, s.health_status::text,
		       (SELECT d.version FROM deployments d WHERE d.service_id = s.id ORDER BY d.started_at DESC LIMIT 1),
		       (SELECT COUNT(*) FROM incidents i WHERE i.service_id = s.id AND i.status::text NOT IN ('resolved', 'closed'))
		FROM services s
		WHERE ($1 = '' OR s.environment::text = $1)
		  AND ($2 = '' OR s.health_status::text = $2)
		  AND ($3 OR (s.slug, s.id) > ((SELECT slug FROM services WHERE id = $4), $4))
		ORDER BY s.slug, s.id
		LIMIT $5
	`
	useCursor := cursorID != uuid.Nil
	rows, err := r.pool.Query(ctx, q, filter.Environment, filter.HealthStatus, !useCursor, cursorID, filter.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ServiceRecord
	for rows.Next() {
		var rec ServiceRecord
		if err := rows.Scan(&rec.ID, &rec.Slug, &rec.Name, &rec.Environment, &rec.Description, &rec.OwnerUserID, &rec.HealthStatus, &rec.CurrentVersion, &rec.ActiveIncidentCount); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	_ = cursorTime
	return out, rows.Err()
}

func (r *Repository) get(ctx context.Context, id uuid.UUID) (ServiceRecord, error) {
	return r.getWhere(ctx, "s.id = $1", id)
}

func (r *Repository) getBySlug(ctx context.Context, slug string) (ServiceRecord, error) {
	return r.getWhere(ctx, "s.slug = $1", slug)
}

func (r *Repository) getWhere(ctx context.Context, where string, arg any) (ServiceRecord, error) {
	var rec ServiceRecord
	err := r.pool.QueryRow(ctx, `
		SELECT s.id, s.slug, s.name, s.environment::text, s.description, s.owner_user_id, s.health_status::text,
		       (SELECT d.version FROM deployments d WHERE d.service_id = s.id ORDER BY d.started_at DESC LIMIT 1),
		       (SELECT COUNT(*) FROM incidents i WHERE i.service_id = s.id AND i.status::text NOT IN ('resolved', 'closed'))
		FROM services s
		WHERE `+where, arg).Scan(&rec.ID, &rec.Slug, &rec.Name, &rec.Environment, &rec.Description, &rec.OwnerUserID, &rec.HealthStatus, &rec.CurrentVersion, &rec.ActiveIncidentCount)
	return rec, err
}

func (r *Repository) recentDeployments(ctx context.Context, serviceID uuid.UUID, limit int) ([]Deployment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, version, git_sha, status::text, started_at, completed_at
		FROM deployments
		WHERE service_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, serviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Deployment
	for rows.Next() {
		var d Deployment
		if err := rows.Scan(&d.ID, &d.Version, &d.GitSHA, &d.Status, &d.StartedAt, &d.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}
