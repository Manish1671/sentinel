package remediation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sentinel-dev/sentinel/services/remediation/internal/approvals"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/config"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/database"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/events"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/executor"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/kafka"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/recommendations"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/verification"
)

type HandleResult struct {
	Commit bool
}

type Processor struct {
	cfg  config.Config
	log  *slog.Logger
	store *database.Store
	exec executor.Executor
	pub  kafka.Publisher
}

func NewProcessor(cfg config.Config, log *slog.Logger, store *database.Store, exec executor.Executor, pub kafka.Publisher) *Processor {
	return &Processor{cfg: cfg, log: log, store: store, exec: exec, pub: pub}
}

func (p *Processor) Handle(ctx context.Context, raw []byte, topic string, partition int, offset int64) (HandleResult, error) {
	env, err := events.Parse(raw)
	if err != nil {
		p.log.Error("poison_message", "error", err.Error(), "kafka_topic", topic, "offset", offset)
		return HandleResult{Commit: true}, nil
	}
	prev, err := p.store.GetProcessed(ctx, env.EventID)
	if err != nil {
		return HandleResult{}, err
	}
	if prev.Exists && prev.Published {
		return HandleResult{Commit: true}, nil
	}
	switch env.EventType {
	case events.TypeInvCompleted:
		err = p.ingestInvestigation(ctx, env)
	case events.TypeRequested:
		err = p.executeRequested(ctx, env)
	default:
		p.log.Info("ignored_type", "event_type", env.EventType, "event_id", env.EventID)
		return HandleResult{Commit: true}, nil
	}
	if err != nil {
		return HandleResult{}, err
	}
	return HandleResult{Commit: true}, nil
}

func (p *Processor) ingestInvestigation(ctx context.Context, env events.Envelope) error {
	invID, err := events.UUIDField(env.Payload, "investigation_id")
	if err != nil {
		p.log.Error("investigation_completed_invalid", "error", err.Error())
		return p.store.RecordProcessed(ctx, env.EventID, env.EventType, nil, nil, "ignored", true)
	}
	status, _ := env.Payload["status"].(string)
	if status != "completed" {
		return p.store.RecordProcessed(ctx, env.EventID, env.EventType, nil, &invID, "ignored", true)
	}
	recs, err := p.store.ListRecommendationsByInvestigation(ctx, invID)
	if err != nil {
		return err
	}
	for _, rec := range recs {
		if _, err := p.CreateFromRecommendation(ctx, rec, nil); err != nil {
			p.log.Info("recommendation_skipped", "recommendation_id", rec.ID, "error", err.Error())
		}
	}
	return p.store.RecordProcessed(ctx, env.EventID, env.EventType, nil, &invID, "ingested", true)
}

func (p *Processor) CreateFromRecommendation(ctx context.Context, rec database.Recommendation, requestedBy *uuid.UUID) (database.Remediation, error) {
	action, err := recommendations.CanonicalAction(rec.ActionType)
	if err != nil {
		return database.Remediation{}, err
	}
	params, err := recommendations.NormalizeParams(action, rec.Parameters)
	if err != nil {
		return database.Remediation{}, err
	}
	if !recommendations.EligibleRecommendation(rec.Status) {
		return database.Remediation{}, fmt.Errorf("recommendation status %s is not executable", rec.Status)
	}
	inc, err := p.store.GetIncident(ctx, rec.IncidentID)
	if err != nil {
		return database.Remediation{}, fmt.Errorf("incident not found")
	}
	if !recommendations.EligibleIncident(inc.Status) {
		return database.Remediation{}, fmt.Errorf("incident status %s is not eligible", inc.Status)
	}
	if rec.IncidentID != inc.ID {
		return database.Remediation{}, fmt.Errorf("recommendation is not associated with this incident")
	}
	target := rec.TargetServiceID
	if target == uuid.Nil {
		target = inc.ServiceID
	}
	ok, err := p.store.ServiceExists(ctx, target)
	if err != nil {
		return database.Remediation{}, err
	}
	if !ok {
		return database.Remediation{}, fmt.Errorf("target service does not exist")
	}
	if !recommendations.TargetMatches(rec.TargetServiceID, inc.ServiceID, uuid.Nil) {
		return database.Remediation{}, fmt.Errorf("target is outside incident service scope")
	}
	idem := fmt.Sprintf("recommendation:%s:1", rec.ID)
	remID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("remediation:"+rec.ID.String()+":1"))
	approvalID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("approval:"+remID.String()))
	row, err := p.store.InsertPending(ctx, database.Remediation{
		ID:             remID,
		IncidentID:     rec.IncidentID,
		RecommendationID: rec.ID,
		ServiceID:      target,
		ActionType:     action,
		Parameters:     params,
		RequestedBy:    requestedBy,
		IdempotencyKey: idem,
	}, approvalID)
	if err != nil {
		return database.Remediation{}, err
	}
	_ = p.store.AppendEvent(ctx, rec.IncidentID, "recommendation_added", requestedBy, map[string]any{
		"source":          events.Source,
		"audit_event":     "remediation.created",
		"event_key":       "remediation.created:" + row.ID.String(),
		"remediation_id":  row.ID.String(),
		"incident_id":     rec.IncidentID.String(),
		"action":          action,
		"from":            "",
		"to":              "pending_approval",
		"recommendation_id": rec.ID.String(),
	})
	return row, nil
}

func (p *Processor) Approve(ctx context.Context, remID uuid.UUID, actor approvals.Actor, comment, idemKey, reqHash string) (database.Remediation, error) {
	return p.decide(ctx, remID, actor, "approved", comment, "remediation.approve", idemKey, reqHash)
}

func (p *Processor) Reject(ctx context.Context, remID uuid.UUID, actor approvals.Actor, comment, idemKey, reqHash string) (database.Remediation, error) {
	return p.decide(ctx, remID, actor, "rejected", comment, "remediation.reject", idemKey, reqHash)
}

func (p *Processor) decide(ctx context.Context, remID uuid.UUID, actor approvals.Actor, decision, comment, scope, idemKey, reqHash string) (database.Remediation, error) {
	if !approvals.CanApprove(actor.Role, "approver") {
		return database.Remediation{}, errForbidden
	}
	if idemKey != "" {
		res, hash, ok, err := p.store.HTTPIdempotency(ctx, scope, idemKey, actor.ID)
		if err != nil {
			return database.Remediation{}, err
		}
		if ok {
			if hash != reqHash {
				return database.Remediation{}, errIdempotencyConflict
			}
			row, err := p.store.GetRemediation(ctx, res)
			if err != nil {
				return database.Remediation{}, err
			}
			return row, nil
		}
	}
	row, err := p.store.GetRemediation(ctx, remID)
	if err != nil {
		return database.Remediation{}, err
	}
	rec, err := p.store.GetRecommendation(ctx, row.RecommendationID)
	if err != nil {
		return database.Remediation{}, err
	}
	if !approvals.CanApprove(actor.Role, rec.RequiredApprovalRole) {
		return database.Remediation{}, errForbidden
	}
	if row.Status != "pending_approval" {
		if row.ApprovalDecision == decision {
			return row, nil
		}
		return database.Remediation{}, errAlreadyDecided
	}
	if err := p.store.Decide(ctx, remID, decision, actor.ID, comment); err != nil {
		if err.Error() == "not_pending" {
			again, e2 := p.store.GetRemediation(ctx, remID)
			if e2 == nil && again.ApprovalDecision == decision {
				return again, nil
			}
			return database.Remediation{}, errAlreadyDecided
		}
		return database.Remediation{}, err
	}
	audit := "remediation.approved"
	kind := "approval_recorded"
	if decision == "rejected" {
		audit = "remediation.rejected"
	}
	_ = p.store.AppendEvent(ctx, row.IncidentID, kind, &actor.ID, map[string]any{
		"source":         events.Source,
		"audit_event":    audit,
		"event_key":      audit + ":" + remID.String(),
		"remediation_id": remID.String(),
		"incident_id":    row.IncidentID.String(),
		"action":         row.ActionType,
		"from":           "pending_approval",
		"to":             map[bool]string{true: "approved", false: "rejected"}[decision == "approved"],
	})
	if idemKey != "" {
		_ = p.store.PutHTTPIdempotency(ctx, scope, idemKey, actor.ID, remID, reqHash)
	}
	updated, err := p.store.GetRemediation(ctx, remID)
	if err != nil {
		return database.Remediation{}, err
	}
	if decision == "approved" {
		if err := p.publishRequested(ctx, updated); err != nil {
			return updated, err
		}
	}
	return updated, nil
}

func (p *Processor) publishRequested(ctx context.Context, row database.Remediation) error {
	eid := events.RequestedEventID(row.ID)
	payload := map[string]any{
		"remediation_id":      row.ID.String(),
		"incident_id":         row.IncidentID.String(),
		"recommendation_id":   row.RecommendationID.String(),
		"service_id":          row.ServiceID.String(),
		"action_type":         row.ActionType,
		"parameters":          row.Parameters,
		"approval_id":         "",
		"approved_by_user_id": nil,
	}
	if row.ApprovalID != nil {
		payload["approval_id"] = row.ApprovalID.String()
	}
	if row.ApprovalActor != nil {
		payload["approved_by_user_id"] = row.ApprovalActor.String()
	}
	if row.RequestedBy != nil {
		payload["requested_by_user_id"] = row.RequestedBy.String()
	}
	env := events.Build(events.TypeRequested, eid, row.IncidentID, nil, time.Now().UTC(), payload)
	body, err := json.Marshal(env)
	if err != nil {
		return err
	}
	if p.pub != nil {
		if err := p.pub.Publish(ctx, kafka.Message{Topic: events.TopicRequested, Key: row.ID.String(), Value: body}); err != nil {
			return err
		}
	}
	return nil
}

func (p *Processor) executeRequested(ctx context.Context, env events.Envelope) error {
	remID, err := events.UUIDField(env.Payload, "remediation_id")
	if err != nil {
		return p.store.RecordProcessed(ctx, env.EventID, env.EventType, nil, nil, "ignored", true)
	}
	row, err := p.store.GetRemediation(ctx, remID)
	if errors.Is(err, pgx.ErrNoRows) {
		return p.store.RecordProcessed(ctx, env.EventID, env.EventType, &remID, nil, "missing", true)
	}
	if err != nil {
		return err
	}
	if row.Status == "rejected" || row.Status == "cancelled" {
		return p.store.RecordProcessed(ctx, env.EventID, env.EventType, &remID, nil, "blocked", true)
	}
	if row.Status == "succeeded" || row.Status == "failed" {
		return p.finishPublish(ctx, env, row)
	}
	if row.Status == "pending_approval" {
		return p.store.RecordProcessed(ctx, env.EventID, env.EventType, &remID, nil, "not_approved", true)
	}
	rec, err := p.store.GetRecommendation(ctx, row.RecommendationID)
	if err != nil {
		return err
	}
	if rec.Status == "expired" || rec.Status == "superseded" || rec.Status == "rejected" {
		_ = p.store.MarkFailed(ctx, remID, "recommendation is not executable", map[string]any{"recommendation_status": rec.Status})
		failed, _ := p.store.GetRemediation(ctx, remID)
		return p.fail(ctx, env, failed, "recommendation_not_executable", rec.Status, false)
	}

	claimed, err := p.store.ClaimRunning(ctx, remID)
	if err != nil {
		return err
	}
	if claimed {
		if err := p.onStartIncident(ctx, row); err != nil {
			return err
		}
		_ = p.store.AppendEvent(ctx, row.IncidentID, "remediation_requested", row.ApprovalActor, map[string]any{
			"source":         events.Source,
			"audit_event":    "remediation.started",
			"event_key":      "remediation.started:" + remID.String(),
			"remediation_id": remID.String(),
			"incident_id":    row.IncidentID.String(),
			"action":         row.ActionType,
			"from":           "approved",
			"to":             "running",
		})
		res, execErr := p.exec.Execute(ctx, executor.Action{
			Type:          row.ActionType,
			ServiceID:     row.ServiceID,
			RemediationID: remID,
			Parameters:    row.Parameters,
		})
		if execErr != nil {
			_ = p.store.MarkFailed(ctx, remID, execErr.Error(), map[string]any{"phase": "execute"})
			_ = p.returnIncidentActive(ctx, row.IncidentID)
			failed, _ := p.store.GetRemediation(ctx, remID)
			_ = p.store.AppendEvent(ctx, row.IncidentID, "remediation_failed", nil, map[string]any{
				"source":         events.Source,
				"audit_event":    "remediation.failed",
				"event_key":      "remediation.failed:" + remID.String(),
				"remediation_id": remID.String(),
				"incident_id":    row.IncidentID.String(),
				"action":         row.ActionType,
				"from":           "running",
				"to":             "failed",
				"error":          execErr.Error(),
			})
			return p.fail(ctx, env, failed, "executor_failed", execErr.Error(), false)
		}
		details := map[string]any{"execution": res.State, "summary": res.Summary}
		if err := p.store.MarkVerifying(ctx, remID, details); err != nil {
			return err
		}
		_, _ = p.store.TransitionIncident(ctx, row.IncidentID, []string{"remediating"}, "verifying")
		_ = p.store.AppendEvent(ctx, row.IncidentID, "status_changed", nil, map[string]any{
			"source":         events.Source,
			"audit_event":    "remediation.verifying",
			"event_key":      "remediation.verifying:" + remID.String(),
			"remediation_id": remID.String(),
			"incident_id":    row.IncidentID.String(),
			"action":         row.ActionType,
			"from":           "running",
			"to":             "verifying",
		})
	}

	row, err = p.store.GetRemediation(ctx, remID)
	if err != nil {
		return err
	}
	st, err := p.exec.State(ctx, row.ServiceID)
	if err != nil {
		return err
	}
	out := verification.Evaluate(row.ActionType, st)
	details := row.VerificationDetails
	if details == nil {
		details = map[string]any{}
	}
	details["verification"] = out.Checks
	if !out.Passed {
		_ = p.store.MarkFailed(ctx, remID, out.Summary, details)
		_ = p.returnIncidentActive(ctx, row.IncidentID)
		failed, _ := p.store.GetRemediation(ctx, remID)
		_ = p.store.AppendEvent(ctx, row.IncidentID, "remediation_failed", nil, map[string]any{
			"source":         events.Source,
			"audit_event":    "remediation.failed",
			"event_key":      "remediation.failed:" + remID.String(),
			"remediation_id": remID.String(),
			"incident_id":    row.IncidentID.String(),
			"action":         row.ActionType,
			"from":           "verifying",
			"to":             "failed",
		})
		return p.fail(ctx, env, failed, "verification_failed", out.Summary, false)
	}
	if err := p.store.SetServiceHealthy(ctx, row.ServiceID); err != nil {
		return err
	}
	if err := p.store.MarkSucceeded(ctx, remID, out.Summary, details); err != nil {
		return err
	}
	_, _ = p.store.TransitionIncident(ctx, row.IncidentID, []string{"verifying", "remediating"}, "resolved")
	_ = p.store.AppendEvent(ctx, row.IncidentID, "remediation_completed", nil, map[string]any{
		"source":         events.Source,
		"audit_event":    "remediation.succeeded",
		"event_key":      "remediation.succeeded:" + remID.String(),
		"remediation_id": remID.String(),
		"incident_id":    row.IncidentID.String(),
		"action":         row.ActionType,
		"from":           "verifying",
		"to":             "succeeded",
	})
	_ = p.store.AppendEvent(ctx, row.IncidentID, "status_changed", nil, map[string]any{
		"source":      events.Source,
		"audit_event": "incident.resolved",
		"event_key":   "incident.resolved:" + remID.String(),
		"from":        "verifying",
		"to":          "resolved",
	})
	done, _ := p.store.GetRemediation(ctx, remID)
	return p.succeed(ctx, env, done)
}

func (p *Processor) onStartIncident(ctx context.Context, row database.Remediation) error {
	inc, err := p.store.GetIncident(ctx, row.IncidentID)
	if err != nil {
		return err
	}
	if inc.Status == "open" {
		_, err = p.store.TransitionIncident(ctx, row.IncidentID, []string{"open"}, "investigating")
		if err != nil {
			return err
		}
		_ = p.store.AppendEvent(ctx, row.IncidentID, "status_changed", nil, map[string]any{
			"source": events.Source, "audit_event": "incident.investigating",
			"event_key": "incident.investigating:" + row.ID.String(), "from": "open", "to": "investigating",
		})
	}
	_, err = p.store.TransitionIncident(ctx, row.IncidentID, []string{"investigating", "open", "remediating"}, "remediating")
	if err != nil {
		return err
	}
	_ = p.store.AppendEvent(ctx, row.IncidentID, "status_changed", nil, map[string]any{
		"source": events.Source, "audit_event": "incident.remediating",
		"event_key": "incident.remediating:" + row.ID.String(), "from": inc.Status, "to": "remediating",
	})
	return nil
}

func (p *Processor) returnIncidentActive(ctx context.Context, incidentID uuid.UUID) error {
	_, err := p.store.TransitionIncident(ctx, incidentID, []string{"remediating", "verifying"}, "investigating")
	if err != nil {
		return err
	}
	_ = p.store.AppendEvent(ctx, incidentID, "status_changed", nil, map[string]any{
		"source": events.Source, "audit_event": "incident.investigating",
		"event_key": "incident.active-after-failure:" + incidentID.String() + ":" + time.Now().UTC().Format(time.RFC3339Nano),
		"from": "remediating|verifying", "to": "investigating",
	})
	return nil
}

func (p *Processor) succeed(ctx context.Context, env events.Envelope, row database.Remediation) error {
	eid := events.CompletedEventID(row.ID)
	cause := env.EventID
	completed := time.Now().UTC()
	if row.CompletedAt != nil {
		completed = *row.CompletedAt
	}
	payload := map[string]any{
		"remediation_id":        row.ID.String(),
		"incident_id":           row.IncidentID.String(),
		"status":                "succeeded",
		"result_summary":        deref(row.ResultSummary),
		"verification_status":   "passed",
		"verification_details":  row.VerificationDetails,
		"completed_at":          completed.Format(time.RFC3339Nano),
	}
	out := events.Build(events.TypeCompleted, eid, row.IncidentID, &cause, completed, payload)
	body, _ := json.Marshal(out)
	if p.pub != nil {
		if err := p.pub.Publish(ctx, kafka.Message{Topic: events.TopicCompleted, Key: row.ID.String(), Value: body}); err != nil {
			return err
		}
	}
	return p.store.RecordProcessed(ctx, env.EventID, env.EventType, &row.ID, nil, "succeeded", true)
}

func (p *Processor) fail(ctx context.Context, env events.Envelope, row database.Remediation, code, message string, retryable bool) error {
	eid := events.FailedEventID(row.ID)
	cause := env.EventID
	failedAt := time.Now().UTC()
	payload := map[string]any{
		"remediation_id":      row.ID.String(),
		"incident_id":         row.IncidentID.String(),
		"error_code":          code,
		"error_message":       message,
		"retryable":           retryable,
		"verification_status": "failed",
		"failed_at":           failedAt.Format(time.RFC3339Nano),
	}
	out := events.Build(events.TypeFailed, eid, row.IncidentID, &cause, failedAt, payload)
	body, _ := json.Marshal(out)
	if p.pub != nil {
		if err := p.pub.Publish(ctx, kafka.Message{Topic: events.TopicFailed, Key: row.ID.String(), Value: body}); err != nil {
			return err
		}
	}
	return p.store.RecordProcessed(ctx, env.EventID, env.EventType, &row.ID, nil, "failed", true)
}

func (p *Processor) finishPublish(ctx context.Context, env events.Envelope, row database.Remediation) error {
	if row.Status == "succeeded" {
		return p.succeed(ctx, env, row)
	}
	return p.fail(ctx, env, row, "already_failed", deref(row.ErrorMessage), false)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func HashRequest(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(h[:])
}

var (
	errForbidden           = fmt.Errorf("forbidden")
	errAlreadyDecided      = fmt.Errorf("approval_already_decided")
	errIdempotencyConflict = fmt.Errorf("idempotency_key_conflict")
)

func IsForbidden(err error) bool           { return errors.Is(err, errForbidden) }
func IsAlreadyDecided(err error) bool      { return errors.Is(err, errAlreadyDecided) }
func IsIdempotencyConflict(err error) bool { return errors.Is(err, errIdempotencyConflict) }
