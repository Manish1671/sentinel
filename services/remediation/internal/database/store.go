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
)

type Store struct {
	db *DB
}

func NewStore(db *DB) *Store { return &Store{db: db} }

func (s *Store) DB() *DB { return s.db }

func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("store is not configured")
	}
	return s.db.Ping(ctx)
}

type User struct {
	ID     uuid.UUID
	Email  string
	Role   string
	Status string
}

type Recommendation struct {
	ID                   uuid.UUID
	IncidentID           uuid.UUID
	InvestigationID      *uuid.UUID
	ActionType           string
	Title                string
	Rationale            string
	TargetServiceID      uuid.UUID
	Parameters           map[string]any
	Confidence           float64
	RiskLevel            string
	RequiredApprovalRole string
	Status               string
}

type Incident struct {
	ID        uuid.UUID
	Reference string
	ServiceID uuid.UUID
	Title     string
	Status    string
	Version   int
}

type Remediation struct {
	ID                 uuid.UUID
	IncidentID         uuid.UUID
	RecommendationID   uuid.UUID
	ServiceID          uuid.UUID
	Status             string
	ActionType         string
	Parameters         map[string]any
	RequestedBy        *uuid.UUID
	AttemptNumber      int
	IdempotencyKey     string
	ResultSummary      *string
	VerificationStatus string
	VerificationDetails map[string]any
	ErrorMessage       *string
	StartedAt          *time.Time
	CompletedAt        *time.Time
	CreatedAt          time.Time
	ApprovalID         *uuid.UUID
	ApprovalDecision   string
	ApprovalActor      *uuid.UUID
	ApprovalComment    *string
	DecidedAt          *time.Time
}

type Processed struct {
	Exists    bool
	Published bool
	Outcome   string
}

func (s *Store) GetUser(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, email, role::text, status::text FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.Role, &u.Status)
	return u, err
}

func (s *Store) GetRecommendation(ctx context.Context, id uuid.UUID) (Recommendation, error) {
	var r Recommendation
	var inv *uuid.UUID
	var params []byte
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, incident_id, investigation_id, action_type, title, rationale, target_service_id,
		       parameters, confidence, risk_level::text, required_approval_role::text, status::text
		FROM recommendations WHERE id = $1
	`, id).Scan(&r.ID, &r.IncidentID, &inv, &r.ActionType, &r.Title, &r.Rationale, &r.TargetServiceID,
		&params, &r.Confidence, &r.RiskLevel, &r.RequiredApprovalRole, &r.Status)
	if err != nil {
		return Recommendation{}, err
	}
	r.InvestigationID = inv
	r.Parameters = decodeMap(params)
	return r, nil
}

func (s *Store) ListRecommendationsByInvestigation(ctx context.Context, investigationID uuid.UUID) ([]Recommendation, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id FROM recommendations WHERE investigation_id = $1 ORDER BY created_at
	`, investigationID)
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
	out := make([]Recommendation, 0, len(ids))
	for _, id := range ids {
		r, err := s.GetRecommendation(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Store) GetIncident(ctx context.Context, id uuid.UUID) (Incident, error) {
	var i Incident
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, reference, service_id, title, status::text, version FROM incidents WHERE id = $1
	`, id).Scan(&i.ID, &i.Reference, &i.ServiceID, &i.Title, &i.Status, &i.Version)
	return i, err
}

func (s *Store) ServiceExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var n int
	err := s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM services WHERE id = $1`, id).Scan(&n)
	return n > 0, err
}

func (s *Store) GetRemediation(ctx context.Context, id uuid.UUID) (Remediation, error) {
	return s.scanRemediation(ctx, `
		SELECT r.id, r.incident_id, r.recommendation_id, r.service_id, r.status::text, r.action_type,
		       r.parameters, r.requested_by_user_id, r.attempt_number, r.idempotency_key, r.result_summary,
		       r.verification_status::text, r.verification_details, r.error_message, r.started_at, r.completed_at, r.created_at,
		       a.id, COALESCE(a.decision::text, 'pending'), a.actor_user_id, a.comment, a.decided_at
		FROM remediations r
		LEFT JOIN approvals a ON a.remediation_id = r.id
		WHERE r.id = $1
	`, id)
}

func (s *Store) GetRemediationByIdempotency(ctx context.Context, key string) (Remediation, error) {
	return s.scanRemediation(ctx, `
		SELECT r.id, r.incident_id, r.recommendation_id, r.service_id, r.status::text, r.action_type,
		       r.parameters, r.requested_by_user_id, r.attempt_number, r.idempotency_key, r.result_summary,
		       r.verification_status::text, r.verification_details, r.error_message, r.started_at, r.completed_at, r.created_at,
		       a.id, COALESCE(a.decision::text, 'pending'), a.actor_user_id, a.comment, a.decided_at
		FROM remediations r
		LEFT JOIN approvals a ON a.remediation_id = r.id
		WHERE r.idempotency_key = $1
	`, key)
}

func (s *Store) scanRemediation(ctx context.Context, q string, arg any) (Remediation, error) {
	var r Remediation
	var params, details []byte
	err := s.db.Pool.QueryRow(ctx, q, arg).Scan(
		&r.ID, &r.IncidentID, &r.RecommendationID, &r.ServiceID, &r.Status, &r.ActionType,
		&params, &r.RequestedBy, &r.AttemptNumber, &r.IdempotencyKey, &r.ResultSummary,
		&r.VerificationStatus, &details, &r.ErrorMessage, &r.StartedAt, &r.CompletedAt, &r.CreatedAt,
		&r.ApprovalID, &r.ApprovalDecision, &r.ApprovalActor, &r.ApprovalComment, &r.DecidedAt,
	)
	if err != nil {
		return Remediation{}, err
	}
	r.Parameters = decodeMap(params)
	r.VerificationDetails = decodeMap(details)
	if r.ApprovalDecision == "" {
		r.ApprovalDecision = "pending"
	}
	return r, nil
}

func (s *Store) InsertPending(ctx context.Context, rem Remediation, approvalID uuid.UUID) (Remediation, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return Remediation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	params, _ := json.Marshal(rem.Parameters)
	_, err = tx.Exec(ctx, `
		INSERT INTO remediations (
			id, incident_id, recommendation_id, service_id, status, action_type, parameters,
			requested_by_user_id, attempt_number, idempotency_key, verification_status
		) VALUES ($1,$2,$3,$4,'pending_approval',$5,$6::jsonb,$7,1,$8,'pending')
		ON CONFLICT (idempotency_key) DO NOTHING
	`, rem.ID, rem.IncidentID, rem.RecommendationID, rem.ServiceID, rem.ActionType, params,
		rem.RequestedBy, rem.IdempotencyKey)
	if err != nil {
		return Remediation{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO approvals (id, remediation_id, incident_id, recommendation_id, decision)
		SELECT $1, r.id, r.incident_id, r.recommendation_id, 'pending'
		FROM remediations r WHERE r.idempotency_key = $2
		ON CONFLICT (remediation_id) DO NOTHING
	`, approvalID, rem.IdempotencyKey)
	if err != nil {
		return Remediation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Remediation{}, err
	}
	return s.GetRemediationByIdempotency(ctx, rem.IdempotencyKey)
}

func (s *Store) AppendEvent(ctx context.Context, incidentID uuid.UUID, kind string, actor *uuid.UUID, payload map[string]any) error {
	body, _ := json.Marshal(payload)
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("ie:%s:%s:%v", incidentID, kind, payload["audit_event"])))
	if raw, ok := payload["event_key"].(string); ok && raw != "" {
		id = uuid.NewSHA1(uuid.NameSpaceOID, []byte(raw))
	}
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO incident_events (id, incident_id, kind, actor_user_id, payload)
		VALUES ($1,$2,$3::incident_event_kind,$4,$5::jsonb)
		ON CONFLICT (id) DO NOTHING
	`, id, incidentID, kind, actor, body)
	return err
}

func (s *Store) TransitionIncident(ctx context.Context, id uuid.UUID, from []string, to string) (Incident, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return Incident{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	inc, err := s.GetIncident(ctx, id)
	if err != nil {
		return Incident{}, err
	}
	allowed := false
	for _, f := range from {
		if inc.Status == f {
			allowed = true
			break
		}
	}
	if !allowed {
		return inc, nil
	}
	if inc.Status == to {
		return inc, nil
	}
	if to == "resolved" {
		_, err = tx.Exec(ctx, `
			UPDATE incidents SET status = 'resolved'::incident_status, resolved_at = now(), version = version + 1
			WHERE id = $1 AND status::text = ANY($2::text[])
		`, id, from)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE incidents SET status = $2::incident_status, resolved_at = NULL, closed_at = NULL, version = version + 1
			WHERE id = $1 AND status::text = ANY($3::text[])
		`, id, to, from)
	}
	if err != nil {
		return Incident{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Incident{}, err
	}
	return s.GetIncident(ctx, id)
}

func (s *Store) ClaimRunning(ctx context.Context, id uuid.UUID) (bool, error) {
	tag, err := s.db.Pool.Exec(ctx, `
		UPDATE remediations
		SET status = 'running', started_at = COALESCE(started_at, now())
		WHERE id = $1 AND status = 'approved'
	`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (s *Store) MarkVerifying(ctx context.Context, id uuid.UUID, details map[string]any) error {
	body, _ := json.Marshal(details)
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE remediations
		SET status = 'verifying', verification_status = 'pending', verification_details = $2::jsonb
		WHERE id = $1 AND status IN ('running', 'verifying')
	`, id, body)
	return err
}

func (s *Store) MarkSucceeded(ctx context.Context, id uuid.UUID, summary string, details map[string]any) error {
	body, _ := json.Marshal(details)
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		UPDATE remediations
		SET status = 'succeeded', verification_status = 'passed', result_summary = $2,
		    verification_details = $3::jsonb, completed_at = now(), error_message = NULL
		WHERE id = $1 AND status IN ('verifying', 'running')
	`, id, summary, body)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE recommendations SET status = 'accepted'
		WHERE id = (SELECT recommendation_id FROM remediations WHERE id = $1)
		  AND status IN ('proposed', 'accepted')
	`, id)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) MarkFailed(ctx context.Context, id uuid.UUID, message string, details map[string]any) error {
	body, _ := json.Marshal(details)
	if details == nil {
		body = []byte("{}")
	}
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE remediations
		SET status = 'failed', verification_status = 'failed', error_message = $2,
		    verification_details = $3::jsonb, completed_at = now()
		WHERE id = $1 AND status IN ('approved', 'running', 'verifying')
	`, id, message, body)
	return err
}

func (s *Store) Decide(ctx context.Context, remID uuid.UUID, decision string, actor uuid.UUID, comment string) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE remediations SET status = $2
		WHERE id = $1 AND status = 'pending_approval'
	`, remID, mapDecisionToStatus(decision))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("not_pending")
	}
	_, err = tx.Exec(ctx, `
		UPDATE approvals
		SET decision = $2::approval_decision, actor_user_id = $3, comment = $4, decided_at = now()
		WHERE remediation_id = $1 AND decision = 'pending'
	`, remID, decision, actor, nullIfEmpty(comment))
	if err != nil {
		return err
	}
	recStatus := "accepted"
	if decision == "rejected" {
		recStatus = "rejected"
	}
	_, err = tx.Exec(ctx, `
		UPDATE recommendations SET status = $2::recommendation_status
		WHERE id = (SELECT recommendation_id FROM remediations WHERE id = $1)
		  AND status = 'proposed'
	`, remID, recStatus)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func mapDecisionToStatus(d string) string {
	if d == "rejected" {
		return "rejected"
	}
	return "approved"
}

func (s *Store) GetProcessed(ctx context.Context, eventID uuid.UUID) (Processed, error) {
	var p Processed
	err := s.db.Pool.QueryRow(ctx, `
		SELECT outcome, published FROM remediation_processed_events WHERE event_id = $1
	`, eventID).Scan(&p.Outcome, &p.Published)
	if errors.Is(err, pgx.ErrNoRows) {
		return Processed{}, nil
	}
	if err != nil {
		return Processed{}, err
	}
	p.Exists = true
	return p, nil
}

func (s *Store) RecordProcessed(ctx context.Context, eventID uuid.UUID, eventType string, remID, invID *uuid.UUID, outcome string, published bool) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO remediation_processed_events (event_id, event_type, remediation_id, investigation_id, outcome, published)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (event_id) DO UPDATE SET outcome = EXCLUDED.outcome, published = EXCLUDED.published
	`, eventID, eventType, remID, invID, outcome, published)
	return err
}

func (s *Store) HTTPIdempotency(ctx context.Context, scope, key string, actor uuid.UUID) (resource uuid.UUID, hash string, ok bool, err error) {
	err = s.db.Pool.QueryRow(ctx, `
		SELECT resource_id, request_hash FROM http_idempotency_keys
		WHERE scope = $1 AND key = $2 AND actor_user_id = $3
	`, scope, key, actor).Scan(&resource, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", false, nil
	}
	if err != nil {
		return uuid.Nil, "", false, err
	}
	return resource, hash, true, nil
}

func (s *Store) PutHTTPIdempotency(ctx context.Context, scope, key string, actor, resource uuid.UUID, hash string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO http_idempotency_keys (scope, key, actor_user_id, request_hash, resource_id)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (scope, key, actor_user_id) DO NOTHING
	`, scope, key, actor, hash, resource)
	if isUnique(err) {
		return nil
	}
	return err
}

func (s *Store) ListByIncident(ctx context.Context, incidentID uuid.UUID) ([]Remediation, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT id FROM remediations WHERE incident_id = $1 ORDER BY created_at`, incidentID)
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
	out := make([]Remediation, 0, len(ids))
	for _, id := range ids {
		r, err := s.GetRemediation(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Store) Timeline(ctx context.Context, incidentID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, kind::text, actor_user_id, occurred_at, payload
		FROM incident_events WHERE incident_id = $1 ORDER BY occurred_at
	`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id uuid.UUID
		var kind string
		var actor *uuid.UUID
		var at time.Time
		var payload []byte
		if err := rows.Scan(&id, &kind, &actor, &at, &payload); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "kind": kind, "actor_user_id": actor,
			"occurred_at": at.UTC().Format(time.RFC3339Nano),
			"payload":     decodeMap(payload),
		})
	}
	return out, rows.Err()
}

func (s *Store) SetServiceHealthy(ctx context.Context, serviceID uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, `UPDATE services SET health_status = 'healthy' WHERE id = $1`, serviceID)
	if err != nil {
		return err
	}
	_, err = s.db.Pool.Exec(ctx, `
		UPDATE remediation_simulator_state SET health_status = 'healthy', updated_at = now() WHERE service_id = $1
	`, serviceID)
	return err
}

func decodeMap(b []byte) map[string]any {
	if len(b) == 0 {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{}
	}
	return m
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func isUnique(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}
