package ops

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sentinel-dev/sentinel/apps/api/internal/paging"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetInvestigation(ctx context.Context, id uuid.UUID) (Investigation, error) {
	var in Investigation
	var toolUsage []byte
	err := r.pool.QueryRow(ctx, `
		SELECT i.id, i.incident_id, inc.reference, i.status::text, i.requested_by_user_id, i.model_name, i.model_version,
		       i.root_cause_hypothesis, i.reasoning_summary, i.confidence, i.risk_level::text, i.tool_usage,
		       i.error_message, i.requested_at, i.started_at, i.completed_at
		FROM investigations i
		JOIN incidents inc ON inc.id = i.incident_id
		WHERE i.id = $1
	`, id).Scan(
		&in.ID, &in.IncidentID, &in.IncidentReference, &in.Status, &in.RequestedByUserID, &in.ModelName, &in.ModelVersion,
		&in.RootCauseHypothesis, &in.ReasoningSummary, &in.Confidence, &in.RiskLevel, &toolUsage,
		&in.ErrorMessage, &in.RequestedAt, &in.StartedAt, &in.CompletedAt,
	)
	if err != nil {
		return Investigation{}, err
	}
	in.ToolUsage = jsonBytes(toolUsage)
	evidence, err := r.listEvidence(ctx, id)
	if err != nil {
		return Investigation{}, err
	}
	in.Evidence = evidence
	return in, nil
}

func (r *Repository) listEvidence(ctx context.Context, investigationID uuid.UUID) ([]Evidence, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tool_name, source_type::text, summary, artifact_uri, source_ref, metadata, captured_at
		FROM evidence
		WHERE investigation_id = $1
		ORDER BY captured_at ASC, id ASC
	`, investigationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Evidence{}
	for rows.Next() {
		var e Evidence
		var meta []byte
		if err := rows.Scan(&e.ID, &e.ToolName, &e.SourceType, &e.Summary, &e.ArtifactURI, &e.SourceRef, &meta, &e.CapturedAt); err != nil {
			return nil, err
		}
		e.Metadata = jsonBytes(meta)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) ListInvestigations(ctx context.Context, filter ListFilter, incidentID uuid.UUID, cursorTime time.Time, cursorID uuid.UUID) ([]Investigation, error) {
	useCursor := cursorID != uuid.Nil
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.incident_id, inc.reference, i.status::text, i.requested_by_user_id, i.model_name, i.model_version,
		       i.root_cause_hypothesis, i.reasoning_summary, i.confidence, i.risk_level::text, i.tool_usage,
		       i.error_message, i.requested_at, i.started_at, i.completed_at
		FROM investigations i
		JOIN incidents inc ON inc.id = i.incident_id
		WHERE ($1::uuid IS NULL OR i.incident_id = $1)
		  AND ($2 = '' OR i.status::text = $2)
		  AND ($3 OR (i.requested_at, i.id) < ($4, $5))
		ORDER BY i.requested_at DESC, i.id DESC
		LIMIT $6
	`, uuidOrNil(incidentID), filter.Status, !useCursor, cursorTime, cursorID, filter.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Investigation
	for rows.Next() {
		var in Investigation
		var toolUsage []byte
		if err := rows.Scan(
			&in.ID, &in.IncidentID, &in.IncidentReference, &in.Status, &in.RequestedByUserID, &in.ModelName, &in.ModelVersion,
			&in.RootCauseHypothesis, &in.ReasoningSummary, &in.Confidence, &in.RiskLevel, &toolUsage,
			&in.ErrorMessage, &in.RequestedAt, &in.StartedAt, &in.CompletedAt,
		); err != nil {
			return nil, err
		}
		in.ToolUsage = jsonBytes(toolUsage)
		in.Evidence = []Evidence{}
		out = append(out, in)
	}
	return out, rows.Err()
}

func (r *Repository) GetRecommendation(ctx context.Context, id uuid.UUID) (Recommendation, error) {
	var rec Recommendation
	var params []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, incident_id, investigation_id, action_type, title, rationale, target_service_id,
		       parameters, confidence, risk_level::text, required_approval_role::text, status::text, created_at
		FROM recommendations
		WHERE id = $1
	`, id).Scan(
		&rec.ID, &rec.IncidentID, &rec.InvestigationID, &rec.ActionType, &rec.Title, &rec.Rationale, &rec.TargetServiceID,
		&params, &rec.Confidence, &rec.RiskLevel, &rec.RequiredApprovalRole, &rec.Status, &rec.CreatedAt,
	)
	if err != nil {
		return Recommendation{}, err
	}
	rec.Parameters = jsonBytes(params)
	return rec, nil
}

func (r *Repository) ListRecommendations(ctx context.Context, incidentID uuid.UUID, status string) ([]Recommendation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, incident_id, investigation_id, action_type, title, rationale, target_service_id,
		       parameters, confidence, risk_level::text, required_approval_role::text, status::text, created_at
		FROM recommendations
		WHERE incident_id = $1
		  AND ($2 = '' OR status::text = $2)
		ORDER BY created_at DESC, id DESC
	`, incidentID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Recommendation{}
	for rows.Next() {
		var rec Recommendation
		var params []byte
		if err := rows.Scan(
			&rec.ID, &rec.IncidentID, &rec.InvestigationID, &rec.ActionType, &rec.Title, &rec.Rationale, &rec.TargetServiceID,
			&params, &rec.Confidence, &rec.RiskLevel, &rec.RequiredApprovalRole, &rec.Status, &rec.CreatedAt,
		); err != nil {
			return nil, err
		}
		rec.Parameters = jsonBytes(params)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *Repository) GetRemediation(ctx context.Context, id uuid.UUID) (Remediation, error) {
	row, err := r.scanRemediation(ctx, `
		SELECT r.id, r.incident_id, inc.reference, r.recommendation_id, r.service_id, s.slug, r.status::text, r.action_type,
		       r.parameters, r.requested_by_user_id, r.attempt_number, r.result_summary,
		       r.verification_status::text, r.verification_details, r.error_message,
		       r.started_at, r.completed_at, r.created_at,
		       a.id, a.decision::text, a.actor_user_id, a.comment, a.decided_at
		FROM remediations r
		JOIN incidents inc ON inc.id = r.incident_id
		JOIN services s ON s.id = r.service_id
		LEFT JOIN approvals a ON a.remediation_id = r.id
		WHERE r.id = $1
	`, id)
	return row, err
}

func (r *Repository) ListRemediations(ctx context.Context, filter ListFilter, incidentID uuid.UUID, cursorTime time.Time, cursorID uuid.UUID) ([]Remediation, error) {
	useCursor := cursorID != uuid.Nil
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.incident_id, inc.reference, r.recommendation_id, r.service_id, s.slug, r.status::text, r.action_type,
		       r.parameters, r.requested_by_user_id, r.attempt_number, r.result_summary,
		       r.verification_status::text, r.verification_details, r.error_message,
		       r.started_at, r.completed_at, r.created_at,
		       a.id, a.decision::text, a.actor_user_id, a.comment, a.decided_at
		FROM remediations r
		JOIN incidents inc ON inc.id = r.incident_id
		JOIN services s ON s.id = r.service_id
		LEFT JOIN approvals a ON a.remediation_id = r.id
		WHERE ($1::uuid IS NULL OR r.incident_id = $1)
		  AND ($2 = '' OR r.status::text = $2)
		  AND ($3 OR (r.created_at, r.id) < ($4, $5))
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT $6
	`, uuidOrNil(incidentID), filter.Status, !useCursor, cursorTime, cursorID, filter.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Remediation
	for rows.Next() {
		item, err := scanRemediationRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) ListAlertsForIncident(ctx context.Context, incidentID uuid.UUID) ([]Alert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.service_id, s.slug, a.detector_id, a.severity::text, a.status::text, a.title, a.summary, a.started_at, a.labels
		FROM incident_alerts ia
		JOIN alerts a ON a.id = ia.alert_id
		JOIN services s ON s.id = a.service_id
		WHERE ia.incident_id = $1
		ORDER BY ia.attached_at ASC, a.id ASC
	`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Alert{}
	for rows.Next() {
		var a Alert
		var labels []byte
		if err := rows.Scan(&a.ID, &a.ServiceID, &a.ServiceSlug, &a.DetectorID, &a.Severity, &a.Status, &a.Title, &a.Summary, &a.StartedAt, &labels); err != nil {
			return nil, err
		}
		a.Labels = jsonBytes(labels)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) ListDeployments(ctx context.Context, serviceID uuid.UUID, cursorTime time.Time, cursorID uuid.UUID, limit int) ([]Deployment, error) {
	useCursor := cursorID != uuid.Nil
	rows, err := r.pool.Query(ctx, `
		SELECT d.id, d.service_id, s.slug, s.environment::text, d.version, d.git_sha, d.status::text, d.started_at, d.completed_at
		FROM deployments d
		JOIN services s ON s.id = d.service_id
		WHERE ($1::uuid IS NULL OR d.service_id = $1)
		  AND ($2 OR (d.started_at, d.id) < ($3, $4))
		ORDER BY d.started_at DESC, d.id DESC
		LIMIT $5
	`, uuidOrNil(serviceID), !useCursor, cursorTime, cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Deployment
	for rows.Next() {
		var d Deployment
		if err := rows.Scan(&d.ID, &d.ServiceID, &d.ServiceSlug, &d.Environment, &d.Version, &d.GitSHA, &d.Status, &d.StartedAt, &d.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) scanRemediation(ctx context.Context, q string, id uuid.UUID) (Remediation, error) {
	row := r.pool.QueryRow(ctx, q, id)
	return scanRemediationRow(row)
}

func scanRemediationRow(row rowScanner) (Remediation, error) {
	var item Remediation
	var params, details []byte
	var approvalID *uuid.UUID
	var decision *string
	var actor *uuid.UUID
	var comment *string
	var decided *time.Time
	err := row.Scan(
		&item.ID, &item.IncidentID, &item.IncidentReference, &item.RecommendationID, &item.ServiceID, &item.ServiceSlug, &item.Status, &item.ActionType,
		&params, &item.RequestedByUserID, &item.AttemptNumber, &item.ResultSummary,
		&item.VerificationStatus, &details, &item.ErrorMessage,
		&item.StartedAt, &item.CompletedAt, &item.CreatedAt,
		&approvalID, &decision, &actor, &comment, &decided,
	)
	if err != nil {
		return Remediation{}, err
	}
	item.Parameters = jsonBytes(params)
	item.VerificationDetails = jsonBytes(details)
	if approvalID != nil && decision != nil {
		item.Approval = &Approval{
			ID:          *approvalID,
			Decision:    *decision,
			ActorUserID: actor,
			Comment:     comment,
			DecidedAt:   decided,
		}
	}
	return item, nil
}

func jsonBytes(raw []byte) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage([]byte("null"))
	}
	return json.RawMessage(raw)
}

func uuidOrNil(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

func nextPage(limit int, n int, lastTime time.Time, lastID uuid.UUID) paging.Page {
	return paging.Next(limit, n, lastTime, lastID)
}
