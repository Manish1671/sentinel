package incidents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ServiceExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM services WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

func (r *Repository) AlertsForService(ctx context.Context, serviceID uuid.UUID, ids []uuid.UUID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM alerts WHERE service_id = $1 AND id = ANY($2)
	`, serviceID, ids).Scan(&n)
	return n, err
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (Incident, error) {
	return r.getWhere(ctx, `i.id = $1`, id)
}

func (r *Repository) GetByReference(ctx context.Context, reference string) (Incident, error) {
	return r.getWhere(ctx, `i.reference = $1`, reference)
}

func (r *Repository) getWhere(ctx context.Context, where string, arg any) (Incident, error) {
	var in Incident
	err := r.pool.QueryRow(ctx, `
		SELECT i.id, i.reference, i.service_id, s.slug, s.name, s.environment::text,
		       i.title, i.summary, i.severity::text, i.status::text,
		       i.created_by_user_id, i.commander_user_id, i.detected_at, i.resolved_at, i.closed_at, i.version
		FROM incidents i
		JOIN services s ON s.id = i.service_id
		WHERE `+where, arg).Scan(
		&in.ID, &in.Reference, &in.ServiceID, &in.ServiceSlug, &in.ServiceName, &in.Environment,
		&in.Title, &in.Summary, &in.Severity, &in.Status,
		&in.CreatedByUserID, &in.CommanderUserID, &in.DetectedAt, &in.ResolvedAt, &in.ClosedAt, &in.Version,
	)
	if err != nil {
		return Incident{}, err
	}
	ids, err := r.alertIDs(ctx, in.ID)
	if err != nil {
		return Incident{}, err
	}
	in.AlertIDs = ids
	return in, nil
}

func (r *Repository) alertIDs(ctx context.Context, incidentID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT alert_id FROM incident_alerts WHERE incident_id = $1 ORDER BY attached_at, alert_id
	`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if ids == nil {
		ids = []uuid.UUID{}
	}
	return ids, rows.Err()
}

func (r *Repository) List(ctx context.Context, filter ListFilter, serviceID uuid.UUID, cursorTime time.Time, cursorID uuid.UUID) ([]Incident, error) {
	useCursor := cursorID != uuid.Nil
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.reference, i.service_id, s.slug, s.name, s.environment::text,
		       i.title, i.summary, i.severity::text, i.status::text,
		       i.created_by_user_id, i.commander_user_id, i.detected_at, i.resolved_at, i.closed_at, i.version
		FROM incidents i
		JOIN services s ON s.id = i.service_id
		WHERE ($1 = '' OR i.status::text = $1)
		  AND ($2::uuid IS NULL OR i.service_id = $2)
		  AND ($3 = '' OR i.severity::text = $3)
		  AND ($4 OR (i.detected_at, i.id) < ($5, $6))
		ORDER BY i.detected_at DESC, i.id DESC
		LIMIT $7
	`, filter.Status, uuidOrNil(serviceID), filter.Severity, !useCursor, cursorTime, cursorID, filter.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Incident
	for rows.Next() {
		var in Incident
		if err := rows.Scan(
			&in.ID, &in.Reference, &in.ServiceID, &in.ServiceSlug, &in.ServiceName, &in.Environment,
			&in.Title, &in.Summary, &in.Severity, &in.Status,
			&in.CreatedByUserID, &in.CommanderUserID, &in.DetectedAt, &in.ResolvedAt, &in.ClosedAt, &in.Version,
		); err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	return out, rows.Err()
}

func (r *Repository) Timeline(ctx context.Context, incidentID uuid.UUID, limit int, cursorTime time.Time, cursorID uuid.UUID) ([]TimelineEvent, error) {
	useCursor := cursorID != uuid.Nil
	rows, err := r.pool.Query(ctx, `
		SELECT id, kind::text, actor_user_id, occurred_at, payload
		FROM incident_events
		WHERE incident_id = $1
		  AND ($2 OR (occurred_at, id) > ($3, $4))
		ORDER BY occurred_at ASC, id ASC
		LIMIT $5
	`, incidentID, !useCursor, cursorTime, cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TimelineEvent
	for rows.Next() {
		var ev TimelineEvent
		var payload []byte
		if err := rows.Scan(&ev.ID, &ev.Kind, &ev.ActorUserID, &ev.OccurredAt, &payload); err != nil {
			return nil, err
		}
		ev.Payload = map[string]any{}
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &ev.Payload); err != nil {
				return nil, err
			}
		}
		out = append(out, decorateTimeline(ev))
	}
	return out, rows.Err()
}

func (r *Repository) Create(ctx context.Context, actorID uuid.UUID, key, hash string, in CreateInput) (Incident, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Incident{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "incident.create:"+actorID.String()+":"+key); err != nil {
		return Incident{}, false, err
	}

	var existingID uuid.UUID
	var existingHash string
	err = tx.QueryRow(ctx, `
		SELECT resource_id, request_hash
		FROM http_idempotency_keys
		WHERE scope = 'incident.create' AND key = $1 AND actor_user_id = $2
	`, key, actorID).Scan(&existingID, &existingHash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Incident{}, false, err
	}
	if err == nil {
		if existingHash != hash {
			return Incident{}, false, errIdempotencyConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return Incident{}, false, err
		}
		got, err := r.Get(ctx, existingID)
		return got, false, err
	}

	year := time.Now().UTC().Year()
	var next int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(SUBSTRING(reference FROM 10)::int), 0) + 1
		FROM incidents
		WHERE reference LIKE $1
		`, fmt.Sprintf("INC-%d-%%", year)).Scan(&next); err != nil {
		return Incident{}, false, err
	}
	ref := fmt.Sprintf("INC-%d-%04d", year, next)
	now := time.Now().UTC()
	id := uuid.New()
	createdBy := actorID

	_, err = tx.Exec(ctx, `
		INSERT INTO incidents (
			id, reference, service_id, title, summary, severity, status,
			created_by_user_id, commander_user_id, detected_at, version
		) VALUES (
			$1, $2, $3, $4, $5, $6::severity_level, 'open',
			$7, $7, $8, 1
		)
	`, id, ref, in.ServiceID, strings.TrimSpace(in.Title), strings.TrimSpace(in.Summary), in.Severity, createdBy, now)
	if err != nil {
		return Incident{}, false, err
	}

	payload, _ := json.Marshal(map[string]any{"source": "api"})
	if _, err := tx.Exec(ctx, `
		INSERT INTO incident_events (incident_id, kind, actor_user_id, occurred_at, payload)
		VALUES ($1, 'created', $2, $3, $4)
	`, id, createdBy, now, payload); err != nil {
		return Incident{}, false, err
	}

	for _, alertID := range in.AlertIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO incident_alerts (incident_id, alert_id, attached_at)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING
		`, id, alertID, now); err != nil {
			return Incident{}, false, err
		}
		alertPayload, _ := json.Marshal(map[string]any{"alert_id": alertID.String()})
		if _, err := tx.Exec(ctx, `
			INSERT INTO incident_events (incident_id, kind, actor_user_id, occurred_at, payload)
			VALUES ($1, 'alert_attached', $2, $3, $4)
		`, id, createdBy, now, alertPayload); err != nil {
			return Incident{}, false, err
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO http_idempotency_keys (scope, key, actor_user_id, request_hash, resource_id)
		VALUES ('incident.create', $1, $2, $3, $4)
	`, key, actorID, hash, id); err != nil {
		return Incident{}, false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Incident{}, false, err
	}
	got, err := r.Get(ctx, id)
	return got, true, err
}

var errIdempotencyConflict = errors.New("idempotency_key_conflict")

func uuidOrNil(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
