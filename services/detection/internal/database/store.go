package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sentinel-dev/sentinel/packages/telemetry"
)

type Processed struct {
	Exists    bool
	Published bool
	Outcome   string
	AlertIDs  []uuid.UUID
}

type OpenAlert struct {
	ID        uuid.UUID
	StartedAt time.Time
	Status    string
}

type Store struct {
	db *DB
}

func NewStore(db *DB) *Store { return &Store{db: db} }

func (s *Store) ServiceExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var ok bool
	err := s.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM services WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

func (s *Store) GetProcessed(ctx context.Context, eventID uuid.UUID) (Processed, error) {
	var p Processed
	var ids []uuid.UUID
	found := false
	err := telemetry.DBOp(ctx, "get_processed", func(ctx context.Context) error {
		qerr := s.db.Pool.QueryRow(ctx, `
		SELECT published, outcome, alert_ids FROM detection_processed_events WHERE event_id = $1
	`, eventID).Scan(&p.Published, &p.Outcome, &ids)
		if qerr == pgx.ErrNoRows {
			return nil
		}
		if qerr != nil {
			return qerr
		}
		found = true
		return nil
	})
	if err != nil {
		return Processed{}, err
	}
	if !found {
		return Processed{}, nil
	}
	p.Exists = true
	p.AlertIDs = ids
	return p, nil
}

func (s *Store) LatestAlert(ctx context.Context, serviceID uuid.UUID, fingerprint string) (OpenAlert, bool, error) {
	var a OpenAlert
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, started_at, status::text
		FROM alerts
		WHERE service_id = $1 AND fingerprint = $2
		ORDER BY started_at DESC
		LIMIT 1
	`, serviceID, fingerprint).Scan(&a.ID, &a.StartedAt, &a.Status)
	if err == pgx.ErrNoRows {
		return OpenAlert{}, false, nil
	}
	if err != nil {
		return OpenAlert{}, false, err
	}
	return a, true, nil
}

type InsertAlert struct {
	ID                uuid.UUID
	ServiceID         uuid.UUID
	DetectorID        string
	Severity          string
	Title             string
	Summary           string
	Fingerprint       string
	Labels            map[string]string
	TriggeringEventID uuid.UUID
	StartedAt         time.Time
}

func (s *Store) InsertAlert(ctx context.Context, in InsertAlert) error {
	labels, _ := json.Marshal(in.Labels)
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO alerts (
			id, service_id, detector_id, severity, status, title, summary,
			fingerprint, labels, triggering_event_id, started_at
		) VALUES (
			$1, $2, $3, $4::severity_level, 'open', $5, $6, $7, $8::jsonb, $9, $10
		)
	`, in.ID, in.ServiceID, in.DetectorID, in.Severity, in.Title, in.Summary, in.Fingerprint, labels, in.TriggeringEventID, in.StartedAt)
	return err
}

func (s *Store) TouchAlert(ctx context.Context, id uuid.UUID, summary string, labels map[string]string) error {
	labelsJSON, _ := json.Marshal(labels)
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE alerts SET summary = $2, labels = $3::jsonb WHERE id = $1
	`, id, summary, labelsJSON)
	return err
}

func (s *Store) RecordProcessed(ctx context.Context, eventID uuid.UUID, eventType, outcome string, alertIDs []uuid.UUID, published bool) error {
	if alertIDs == nil {
		alertIDs = []uuid.UUID{}
	}
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO detection_processed_events (event_id, event_type, outcome, alert_ids, published)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id) DO UPDATE SET
			published = EXCLUDED.published,
			alert_ids = EXCLUDED.alert_ids,
			outcome = EXCLUDED.outcome
	`, eventID, eventType, outcome, alertIDs, published)
	return err
}

func (s *Store) MarkPublished(ctx context.Context, eventID uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE detection_processed_events SET published = true WHERE event_id = $1
	`, eventID)
	return err
}

func (s *Store) AlertByID(ctx context.Context, id uuid.UUID) (InsertAlert, error) {
	var in InsertAlert
	var labels []byte
	var sev string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, service_id, detector_id, severity::text, title, summary, fingerprint, labels, triggering_event_id, started_at
		FROM alerts WHERE id = $1
	`, id).Scan(&in.ID, &in.ServiceID, &in.DetectorID, &sev, &in.Title, &in.Summary, &in.Fingerprint, &labels, &in.TriggeringEventID, &in.StartedAt)
	if err != nil {
		return InsertAlert{}, err
	}
	in.Severity = sev
	_ = json.Unmarshal(labels, &in.Labels)
	return in, nil
}

func (s *Store) CountOpenByFingerprint(ctx context.Context, serviceID uuid.UUID, fingerprint string) (int, error) {
	var n int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM alerts
		WHERE service_id = $1 AND fingerprint = $2 AND status IN ('open', 'acknowledged')
	`, serviceID, fingerprint).Scan(&n)
	return n, err
}

func (s *Store) Begin(ctx context.Context) (pgx.Tx, error) {
	return s.db.Pool.BeginTx(ctx, pgx.TxOptions{})
}

func TxInsertAlert(ctx context.Context, tx pgx.Tx, in InsertAlert) error {
	labels, _ := json.Marshal(in.Labels)
	_, err := tx.Exec(ctx, `
		INSERT INTO alerts (
			id, service_id, detector_id, severity, status, title, summary,
			fingerprint, labels, triggering_event_id, started_at
		) VALUES (
			$1, $2, $3, $4::severity_level, 'open', $5, $6, $7, $8::jsonb, $9, $10
		)
	`, in.ID, in.ServiceID, in.DetectorID, in.Severity, in.Title, in.Summary, in.Fingerprint, labels, in.TriggeringEventID, in.StartedAt)
	return err
}

func TxTouchAlert(ctx context.Context, tx pgx.Tx, id uuid.UUID, summary string, labels map[string]string) error {
	labelsJSON, _ := json.Marshal(labels)
	_, err := tx.Exec(ctx, `UPDATE alerts SET summary = $2, labels = $3::jsonb WHERE id = $1`, id, summary, labelsJSON)
	return err
}

func TxRecordProcessed(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, eventType, outcome string, alertIDs []uuid.UUID, published bool) error {
	if alertIDs == nil {
		alertIDs = []uuid.UUID{}
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO detection_processed_events (event_id, event_type, outcome, alert_ids, published)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id) DO UPDATE SET
			published = EXCLUDED.published,
			alert_ids = EXCLUDED.alert_ids,
			outcome = EXCLUDED.outcome
	`, eventID, eventType, outcome, alertIDs, published)
	return err
}

func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("store is not configured")
	}
	return s.db.Ping(ctx)
}
