package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/sentinel-dev/sentinel/services/incident/internal/correlation"
)

type Store struct {
	db *DB
}

func NewStore(db *DB) *Store { return &Store{db: db} }

func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("store is not configured")
	}
	return s.db.Ping(ctx)
}

func (s *Store) CountActive(ctx context.Context) (int, error) {
	var n int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM incidents WHERE status::text NOT IN ('resolved', 'closed')
	`).Scan(&n)
	return n, err
}

func (s *Store) DB() *DB { return s.db }

type Processed struct {
	Exists     bool
	Published  bool
	Outcome    string
	IncidentID *uuid.UUID
	AlertID    *uuid.UUID
}

type AlertRow struct {
	ID          uuid.UUID
	ServiceID   uuid.UUID
	DetectorID  string
	Severity    string
	Title       string
	Summary     string
	Fingerprint string
	Labels      map[string]string
	StartedAt   time.Time
	Slug        string
	Environment string
}

type IncidentRow struct {
	ID              uuid.UUID
	Reference       string
	ServiceID       uuid.UUID
	Title           string
	Summary         string
	Severity        string
	Status          string
	DetectedAt      time.Time
	ResolvedAt      *time.Time
	ClosedAt        *time.Time
	Version         int
	CreatedBy       *uuid.UUID
	Commander       *uuid.UUID
	Environment     string
	ServiceSlug     string
	AlertIDs        []uuid.UUID
}

type TimelineRow struct {
	ID         uuid.UUID
	Kind       string
	Actor      *uuid.UUID
	OccurredAt time.Time
	Payload    map[string]any
}

func (s *Store) GetProcessed(ctx context.Context, eventID uuid.UUID) (Processed, error) {
	var p Processed
	err := s.db.Pool.QueryRow(ctx, `
		SELECT published, outcome, incident_id, alert_id
		FROM incident_processed_events WHERE event_id = $1
	`, eventID).Scan(&p.Published, &p.Outcome, &p.IncidentID, &p.AlertID)
	if err == pgx.ErrNoRows {
		return Processed{}, nil
	}
	if err != nil {
		return Processed{}, err
	}
	p.Exists = true
	return p, nil
}

func (s *Store) LoadAlert(ctx context.Context, id uuid.UUID) (AlertRow, error) {
	var a AlertRow
	var labels []byte
	err := s.db.Pool.QueryRow(ctx, `
		SELECT a.id, a.service_id, a.detector_id, a.severity::text, a.title, a.summary,
		       a.fingerprint, a.labels, a.started_at, s.slug, s.environment::text
		FROM alerts a
		JOIN services s ON s.id = a.service_id
		WHERE a.id = $1
	`, id).Scan(&a.ID, &a.ServiceID, &a.DetectorID, &a.Severity, &a.Title, &a.Summary, &a.Fingerprint, &labels, &a.StartedAt, &a.Slug, &a.Environment)
	if err != nil {
		return AlertRow{}, err
	}
	_ = json.Unmarshal(labels, &a.Labels)
	if a.Labels == nil {
		a.Labels = map[string]string{}
	}
	return a, nil
}

func (s *Store) IncidentIDForAlert(ctx context.Context, alertID uuid.UUID) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := s.db.Pool.QueryRow(ctx, `SELECT incident_id FROM incident_alerts WHERE alert_id = $1`, alertID).Scan(&id)
	if err == pgx.ErrNoRows {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}

func (s *Store) Begin(ctx context.Context) (pgx.Tx, error) {
	return s.db.Pool.Begin(ctx)
}

func TxLockService(ctx context.Context, tx pgx.Tx, serviceID uuid.UUID) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1::text))`, serviceID.String())
	return err
}

func TxCandidates(ctx context.Context, tx pgx.Tx, serviceID uuid.UUID) ([]correlation.Candidate, error) {
	rows, err := tx.Query(ctx, `
		SELECT i.id, i.service_id, s.environment::text, i.status::text, i.severity::text,
		       i.detected_at,
		       COALESCE((
		         SELECT MAX(a.started_at) FROM incident_alerts ia
		         JOIN alerts a ON a.id = ia.alert_id WHERE ia.incident_id = i.id
		       ), i.detected_at) AS last_activity
		FROM incidents i
		JOIN services s ON s.id = i.service_id
		WHERE i.service_id = $1
		  AND i.status IN ('open', 'investigating', 'remediating', 'verifying')
		FOR UPDATE OF i
	`, serviceID)
	if err != nil {
		return nil, err
	}
	var out []correlation.Candidate
	for rows.Next() {
		var c correlation.Candidate
		if err := rows.Scan(&c.ID, &c.ServiceID, &c.Environment, &c.Status, &c.Severity, &c.DetectedAt, &c.LastActivityAt); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for i := range out {
		dets, deps, err := txAlertEvidence(ctx, tx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].DetectorIDs = dets
		out[i].DeploymentIDs = deps
	}
	return out, nil
}

func txAlertEvidence(ctx context.Context, tx pgx.Tx, incidentID uuid.UUID) ([]string, []string, error) {
	rows, err := tx.Query(ctx, `
		SELECT a.detector_id, COALESCE(a.labels->>'deployment_id', '')
		FROM incident_alerts ia
		JOIN alerts a ON a.id = ia.alert_id
		WHERE ia.incident_id = $1
	`, incidentID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var dets, deps []string
	seenDep := map[string]struct{}{}
	for rows.Next() {
		var d, dep string
		if err := rows.Scan(&d, &dep); err != nil {
			return nil, nil, err
		}
		dets = append(dets, d)
		if dep != "" {
			if _, ok := seenDep[dep]; !ok {
				seenDep[dep] = struct{}{}
				deps = append(deps, dep)
			}
		}
	}
	return dets, deps, rows.Err()
}

func TxNextReference(ctx context.Context, tx pgx.Tx, at time.Time) (string, error) {
	year := at.UTC().Year()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('sentinel_incident_reference'))`); err != nil {
		return "", err
	}
	var n int
	prefix := fmt.Sprintf("INC-%d-", year)
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(substring(reference from 10)::int), 0)
		FROM incidents WHERE reference LIKE $1
	`, prefix+"%").Scan(&n)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("INC-%d-%04d", year, n+1), nil
}

func TxInsertIncident(ctx context.Context, tx pgx.Tx, in IncidentRow, dedup string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO incidents (
			id, reference, service_id, title, summary, severity, status,
			detected_at, dedup_key, version
		) VALUES ($1,$2,$3,$4,$5,$6::severity_level,'open',$7,$8,1)
	`, in.ID, in.Reference, in.ServiceID, in.Title, in.Summary, in.Severity, in.DetectedAt, dedup)
	return err
}

func TxAttach(ctx context.Context, tx pgx.Tx, incidentID, alertID uuid.UUID, at time.Time) (bool, error) {
	tag, err := tx.Exec(ctx, `
		INSERT INTO incident_alerts (incident_id, alert_id, attached_at)
		VALUES ($1,$2,$3)
		ON CONFLICT (alert_id) DO NOTHING
	`, incidentID, alertID, at)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func TxTimeline(ctx context.Context, tx pgx.Tx, incidentID uuid.UUID, kind string, at time.Time, payload map[string]any) error {
	body, _ := json.Marshal(payload)
	_, err := tx.Exec(ctx, `
		INSERT INTO incident_events (incident_id, kind, occurred_at, payload)
		VALUES ($1, $2::incident_event_kind, $3, $4::jsonb)
	`, incidentID, kind, at, body)
	return err
}

func TxTimelineAlertAttached(ctx context.Context, tx pgx.Tx, incidentID uuid.UUID, at time.Time, payload map[string]any) (bool, error) {
	body, _ := json.Marshal(payload)
	_, err := tx.Exec(ctx, `
		INSERT INTO incident_events (incident_id, kind, occurred_at, payload)
		VALUES ($1, 'alert_attached', $2, $3::jsonb)
	`, incidentID, at, body)
	if err != nil {
		if isUniqueViolation(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func TxUpdateIncident(ctx context.Context, tx pgx.Tx, id uuid.UUID, severity, summary string) (int, error) {
	var version int
	err := tx.QueryRow(ctx, `
		UPDATE incidents
		SET severity = $2::severity_level, summary = $3, version = version + 1
		WHERE id = $1
		RETURNING version
	`, id, severity, summary).Scan(&version)
	return version, err
}

func TxGetIncident(ctx context.Context, tx pgx.Tx, id uuid.UUID) (IncidentRow, error) {
	var in IncidentRow
	err := tx.QueryRow(ctx, `
		SELECT i.id, i.reference, i.service_id, i.title, i.summary, i.severity::text, i.status::text,
		       i.detected_at, i.resolved_at, i.closed_at, i.version, i.created_by_user_id, i.commander_user_id,
		       s.environment::text, s.slug
		FROM incidents i
		JOIN services s ON s.id = i.service_id
		WHERE i.id = $1
	`, id).Scan(&in.ID, &in.Reference, &in.ServiceID, &in.Title, &in.Summary, &in.Severity, &in.Status,
		&in.DetectedAt, &in.ResolvedAt, &in.ClosedAt, &in.Version, &in.CreatedBy, &in.Commander, &in.Environment, &in.ServiceSlug)
	if err != nil {
		return IncidentRow{}, err
	}
	rows, err := tx.Query(ctx, `SELECT alert_id FROM incident_alerts WHERE incident_id = $1 ORDER BY attached_at`, id)
	if err != nil {
		return IncidentRow{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var aid uuid.UUID
		if err := rows.Scan(&aid); err != nil {
			return IncidentRow{}, err
		}
		in.AlertIDs = append(in.AlertIDs, aid)
	}
	return in, rows.Err()
}

func TxRecordProcessed(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, alertID *uuid.UUID, incidentID *uuid.UUID, outcome string, published bool) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO incident_processed_events (event_id, alert_id, incident_id, outcome, published)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (event_id) DO UPDATE SET
			published = EXCLUDED.published,
			incident_id = EXCLUDED.incident_id,
			outcome = EXCLUDED.outcome,
			alert_id = EXCLUDED.alert_id
	`, eventID, alertID, incidentID, outcome, published)
	return err
}

func (s *Store) RecordProcessed(ctx context.Context, eventID uuid.UUID, alertID *uuid.UUID, incidentID *uuid.UUID, outcome string, published bool) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO incident_processed_events (event_id, alert_id, incident_id, outcome, published)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (event_id) DO UPDATE SET
			published = EXCLUDED.published,
			incident_id = EXCLUDED.incident_id,
			outcome = EXCLUDED.outcome,
			alert_id = EXCLUDED.alert_id
	`, eventID, alertID, incidentID, outcome, published)
	return err
}

func (s *Store) MarkPublished(ctx context.Context, eventID uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, `UPDATE incident_processed_events SET published = true WHERE event_id = $1`, eventID)
	return err
}

func (s *Store) GetIncident(ctx context.Context, id uuid.UUID) (IncidentRow, error) {
	var in IncidentRow
	err := s.db.Pool.QueryRow(ctx, `
		SELECT i.id, i.reference, i.service_id, i.title, i.summary, i.severity::text, i.status::text,
		       i.detected_at, i.resolved_at, i.closed_at, i.version, i.created_by_user_id, i.commander_user_id,
		       s.environment::text, s.slug
		FROM incidents i
		JOIN services s ON s.id = i.service_id
		WHERE i.id = $1
	`, id).Scan(&in.ID, &in.Reference, &in.ServiceID, &in.Title, &in.Summary, &in.Severity, &in.Status,
		&in.DetectedAt, &in.ResolvedAt, &in.ClosedAt, &in.Version, &in.CreatedBy, &in.Commander, &in.Environment, &in.ServiceSlug)
	if err != nil {
		return IncidentRow{}, err
	}
	rows, err := s.db.Pool.Query(ctx, `SELECT alert_id FROM incident_alerts WHERE incident_id = $1 ORDER BY attached_at`, id)
	if err != nil {
		return IncidentRow{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var aid uuid.UUID
		if err := rows.Scan(&aid); err != nil {
			return IncidentRow{}, err
		}
		in.AlertIDs = append(in.AlertIDs, aid)
	}
	return in, rows.Err()
}

func (s *Store) ListIncidents(ctx context.Context, status string, limit int) ([]IncidentRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := `
		SELECT i.id, i.reference, i.service_id, i.title, i.summary, i.severity::text, i.status::text,
		       i.detected_at, i.resolved_at, i.closed_at, i.version, i.created_by_user_id, i.commander_user_id,
		       s.environment::text, s.slug
		FROM incidents i
		JOIN services s ON s.id = i.service_id
	`
	args := []any{}
	if status != "" {
		q += ` WHERE i.status = $1::incident_status`
		args = append(args, status)
		q += ` ORDER BY i.detected_at DESC LIMIT $2`
		args = append(args, limit)
	} else {
		q += ` ORDER BY i.detected_at DESC LIMIT $1`
		args = append(args, limit)
	}
	rows, err := s.db.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IncidentRow
	for rows.Next() {
		var in IncidentRow
		if err := rows.Scan(&in.ID, &in.Reference, &in.ServiceID, &in.Title, &in.Summary, &in.Severity, &in.Status,
			&in.DetectedAt, &in.ResolvedAt, &in.ClosedAt, &in.Version, &in.CreatedBy, &in.Commander, &in.Environment, &in.ServiceSlug); err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	return out, rows.Err()
}

func (s *Store) Timeline(ctx context.Context, incidentID uuid.UUID) ([]TimelineRow, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, kind::text, actor_user_id, occurred_at, payload
		FROM incident_events WHERE incident_id = $1
		ORDER BY occurred_at ASC, id ASC
	`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TimelineRow
	for rows.Next() {
		var ev TimelineRow
		var payload []byte
		if err := rows.Scan(&ev.ID, &ev.Kind, &ev.Actor, &ev.OccurredAt, &payload); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(payload, &ev.Payload)
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (s *Store) CountOpenForServiceSince(ctx context.Context, serviceID uuid.UUID, since time.Time) (int, error) {
	var n int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM incidents
		WHERE service_id = $1 AND status = 'open' AND detected_at >= $2
	`, serviceID, since).Scan(&n)
	return n, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
