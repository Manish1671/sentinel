package incidents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sentinel-dev/sentinel/services/incident/internal/config"
	"github.com/sentinel-dev/sentinel/services/incident/internal/correlation"
	"github.com/sentinel-dev/sentinel/services/incident/internal/database"
	"github.com/sentinel-dev/sentinel/services/incident/internal/events"
	"github.com/sentinel-dev/sentinel/services/incident/internal/kafka"
)

type Processor struct {
	cfg  config.Config
	log  *slog.Logger
	store *database.Store
	pub  kafka.Publisher
	now  func() time.Time
}

func NewProcessor(cfg config.Config, log *slog.Logger, store *database.Store, pub kafka.Publisher) *Processor {
	return &Processor{cfg: cfg, log: log, store: store, pub: pub, now: time.Now}
}

type HandleResult struct {
	Commit     bool
	Outcome    string
	EventID    uuid.UUID
	AlertID    uuid.UUID
	IncidentID uuid.UUID
	Decision   string
	Created    bool
	Updated    bool
}

func (p *Processor) Handle(ctx context.Context, raw []byte, topic string, partition int, offset int64) (HandleResult, error) {
	start := time.Now()
	env, err := events.Parse(raw)
	if err != nil {
		p.log.Error("poison_message",
			"error", err.Error(),
			"kafka_topic", topic,
			"partition", partition,
			"offset", offset,
		)
		return HandleResult{Commit: true, Outcome: "poison"}, nil
	}

	res := HandleResult{EventID: env.EventID, Commit: true}
	p.log.Info("alert_received",
		"event_id", env.EventID,
		"event_type", env.EventType,
		"kafka_topic", topic,
		"partition", partition,
		"offset", offset,
	)

	if env.EventType != events.TypeAlert {
		if err := p.store.RecordProcessed(ctx, env.EventID, nil, nil, "ignored_type", true); err != nil {
			return HandleResult{}, err
		}
		res.Outcome = "ignored_type"
		return res, nil
	}

	alertID, err := events.UUIDField(env.Payload, "alert_id")
	if err != nil {
		if recErr := p.store.RecordProcessed(ctx, env.EventID, nil, nil, "invalid_alert", true); recErr != nil {
			return HandleResult{}, recErr
		}
		p.log.Error("invalid_alert_identity", "event_id", env.EventID, "error", err.Error())
		res.Outcome = "invalid_alert"
		return res, nil
	}
	res.AlertID = alertID

	prev, err := p.store.GetProcessed(ctx, env.EventID)
	if err != nil {
		return HandleResult{}, err
	}
	if prev.Exists {
		if prev.IncidentID != nil {
			res.IncidentID = *prev.IncidentID
		}
		if prev.Published {
			p.log.Info("event_duplicate",
				"event_id", env.EventID,
				"alert_id", alertID,
				"incident_id", res.IncidentID,
				"outcome", prev.Outcome,
				"correlation_decision", "duplicate",
				"kafka_topic", topic,
				"partition", partition,
				"offset", offset,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			res.Outcome = "duplicate"
			res.Decision = "duplicate"
			return res, nil
		}
		if prev.Outcome == "created" || prev.Outcome == "attached" {
			if err := p.republish(ctx, env, prev.Outcome, *prev.IncidentID); err != nil {
				return HandleResult{}, err
			}
			if err := p.store.MarkPublished(ctx, env.EventID); err != nil {
				return HandleResult{}, err
			}
			res.Outcome = "republished"
			res.Decision = prev.Outcome
			res.Created = prev.Outcome == "created"
			res.Updated = prev.Outcome == "attached"
			return res, nil
		}
		res.Outcome = "duplicate"
		res.Decision = prev.Outcome
		return res, nil
	}

	alert, err := p.store.LoadAlert(ctx, alertID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return HandleResult{}, fmt.Errorf("alert %s not persisted yet", alertID)
		}
		return HandleResult{}, err
	}
	res.AlertID = alert.ID

	outcome, incident, created, attached, decision, err := p.correlate(ctx, env, alert)
	if err != nil {
		return HandleResult{}, err
	}
	res.Outcome = outcome
	res.Decision = decision
	if incident.ID != uuid.Nil {
		res.IncidentID = incident.ID
	}
	res.Created = created
	res.Updated = attached && !created

	if created {
		if err := p.publishCreated(ctx, env, incident); err != nil {
			return HandleResult{}, err
		}
		if err := p.store.MarkPublished(ctx, env.EventID); err != nil {
			return HandleResult{}, err
		}
	} else if attached {
		if err := p.publishUpdated(ctx, env, incident); err != nil {
			return HandleResult{}, err
		}
		if err := p.store.MarkPublished(ctx, env.EventID); err != nil {
			return HandleResult{}, err
		}
	}

	p.log.Info("correlation_complete",
		"event_id", env.EventID,
		"alert_id", alert.ID,
		"incident_id", res.IncidentID,
		"service_id", alert.ServiceID,
		"severity", incident.Severity,
		"correlation_decision", decision,
		"outcome", outcome,
		"kafka_topic", topic,
		"partition", partition,
		"offset", offset,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return res, nil
}

func (p *Processor) correlate(ctx context.Context, env events.Envelope, alert database.AlertRow) (outcome string, incident database.IncidentRow, created, attached bool, decision string, err error) {
	tx, err := p.store.Begin(ctx)
	if err != nil {
		return "", database.IncidentRow{}, false, false, "", err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = database.TxLockService(ctx, tx, alert.ServiceID); err != nil {
		return "", database.IncidentRow{}, false, false, "", err
	}

	if existingID, ok, findErr := incidentIDInTx(ctx, tx, alert.ID); findErr != nil {
		err = findErr
		return "", database.IncidentRow{}, false, false, "", err
	} else if ok {
		incident, err = database.TxGetIncident(ctx, tx, existingID)
		if err != nil {
			return "", database.IncidentRow{}, false, false, "", err
		}
		if recErr := database.TxRecordProcessed(ctx, tx, env.EventID, &alert.ID, &existingID, "duplicate", true); recErr != nil {
			err = recErr
			return "", database.IncidentRow{}, false, false, "", err
		}
		if err = tx.Commit(ctx); err != nil {
			return "", database.IncidentRow{}, false, false, "", err
		}
		return "duplicate", incident, false, false, "already_associated", nil
	}

	candidates, err := database.TxCandidates(ctx, tx, alert.ServiceID)
	if err != nil {
		return "", database.IncidentRow{}, false, false, "", err
	}
	corrAlert := correlation.Alert{
		ID:          alert.ID,
		ServiceID:   alert.ServiceID,
		Environment: alert.Environment,
		DetectorID:  alert.DetectorID,
		Severity:    alert.Severity,
		StartedAt:   alert.StartedAt,
		Labels:      alert.Labels,
	}
	match, exp := correlation.Select(corrAlert, candidates, p.cfg.CorrelationWindow)

	now := p.now().UTC()
	if match == nil {
		incident, err = p.createIncident(ctx, tx, env, alert, exp, now)
		if err != nil {
			return "", database.IncidentRow{}, false, false, "", err
		}
		if recErr := database.TxRecordProcessed(ctx, tx, env.EventID, &alert.ID, &incident.ID, "created", false); recErr != nil {
			err = recErr
			return "", database.IncidentRow{}, false, false, "", err
		}
		if err = tx.Commit(ctx); err != nil {
			return "", database.IncidentRow{}, false, false, "", err
		}
		return "created", incident, true, true, "new_incident", nil
	}

	incident, attached, err = p.attachAlert(ctx, tx, env, alert, *match, exp, now)
	if err != nil {
		return "", database.IncidentRow{}, false, false, "", err
	}
	out := "attached"
	decision = "matched_existing"
	if !attached {
		out = "duplicate"
		decision = "already_associated"
	}
	pub := !attached
	if recErr := database.TxRecordProcessed(ctx, tx, env.EventID, &alert.ID, &incident.ID, out, pub); recErr != nil {
		err = recErr
		return "", database.IncidentRow{}, false, false, "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", database.IncidentRow{}, false, false, "", err
	}
	return out, incident, false, attached, decision, nil
}

func incidentIDInTx(ctx context.Context, tx pgx.Tx, alertID uuid.UUID) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT incident_id FROM incident_alerts WHERE alert_id = $1`, alertID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}

func (p *Processor) createIncident(ctx context.Context, tx pgx.Tx, env events.Envelope, alert database.AlertRow, exp correlation.Explanation, now time.Time) (database.IncidentRow, error) {
	ref, err := database.TxNextReference(ctx, tx, alert.StartedAt)
	if err != nil {
		return database.IncidentRow{}, err
	}
	id := uuid.New()
	detectors := []string{alert.DetectorID}
	title := correlation.Title(alert.Slug, alert.Environment, detectors)
	summary := correlation.Summary(alert.Slug, detectors, alert.Labels["deployment_version"])
	row := database.IncidentRow{
		ID:         id,
		Reference:  ref,
		ServiceID:  alert.ServiceID,
		Title:      title,
		Summary:    summary,
		Severity:   alert.Severity,
		Status:     "open",
		DetectedAt: alert.StartedAt,
	}
	if err := database.TxInsertIncident(ctx, tx, row, "alert:"+alert.ID.String()); err != nil {
		return database.IncidentRow{}, err
	}
	createdPayload := timelinePayload(env, alert, exp, "Incident opened")
	if err := database.TxTimeline(ctx, tx, id, "created", now, createdPayload); err != nil {
		return database.IncidentRow{}, err
	}
	ok, err := database.TxAttach(ctx, tx, id, alert.ID, now)
	if err != nil {
		return database.IncidentRow{}, err
	}
	if !ok {
		return database.IncidentRow{}, fmt.Errorf("alert %s already associated", alert.ID)
	}
	attachPayload := timelinePayload(env, alert, exp, "Alert attached")
	if _, err := database.TxTimelineAlertAttached(ctx, tx, id, now, attachPayload); err != nil {
		return database.IncidentRow{}, err
	}
	return database.TxGetIncident(ctx, tx, id)
}

func (p *Processor) attachAlert(ctx context.Context, tx pgx.Tx, env events.Envelope, alert database.AlertRow, match correlation.Candidate, exp correlation.Explanation, now time.Time) (database.IncidentRow, bool, error) {
	ok, err := database.TxAttach(ctx, tx, match.ID, alert.ID, now)
	if err != nil {
		return database.IncidentRow{}, false, err
	}
	if !ok {
		existing, loadErr := database.TxGetIncident(ctx, tx, match.ID)
		if loadErr != nil {
			if existingID, found, findErr := incidentIDInTx(ctx, tx, alert.ID); findErr == nil && found {
				row, err := database.TxGetIncident(ctx, tx, existingID)
				return row, false, err
			}
			return database.IncidentRow{}, false, loadErr
		}
		return existing, false, nil
	}
	sev := correlation.MaxSeverity(match.Severity, alert.Severity)
	dets := append(append([]string{}, match.DetectorIDs...), alert.DetectorID)
	summary := correlation.Summary(alert.Slug, dets, alert.Labels["deployment_version"])
	if _, err := database.TxUpdateIncident(ctx, tx, match.ID, sev, summary); err != nil {
		return database.IncidentRow{}, false, err
	}
	attachPayload := timelinePayload(env, alert, exp, "Alert attached")
	if _, err := database.TxTimelineAlertAttached(ctx, tx, match.ID, now, attachPayload); err != nil {
		return database.IncidentRow{}, false, err
	}
	row, err := database.TxGetIncident(ctx, tx, match.ID)
	return row, true, err
}

func timelinePayload(env events.Envelope, alert database.AlertRow, exp correlation.Explanation, summary string) map[string]any {
	return map[string]any{
		"source":     events.Source,
		"summary":    summary,
		"alert_id":   alert.ID.String(),
		"event_id":   env.EventID.String(),
		"actor":      nil,
		"detector_id": alert.DetectorID,
		"deployment_id":      alert.Labels["deployment_id"],
		"deployment_version": alert.Labels["deployment_version"],
		"correlation":        exp,
	}
}

func (p *Processor) republish(ctx context.Context, env events.Envelope, outcome string, incidentID uuid.UUID) error {
	in, err := p.store.GetIncident(ctx, incidentID)
	if err != nil {
		return err
	}
	if outcome == "created" {
		return p.publishCreated(ctx, env, in)
	}
	return p.publishUpdated(ctx, env, in)
}

func (p *Processor) publishCreated(ctx context.Context, src events.Envelope, in database.IncidentRow) error {
	payload := map[string]any{
		"incident_id": in.ID.String(),
		"reference":   in.Reference,
		"service_id":  in.ServiceID.String(),
		"title":       in.Title,
		"summary":     in.Summary,
		"severity":    in.Severity,
		"status":      "open",
		"alert_ids":   uuidStrings(in.AlertIDs),
		"detected_at": in.DetectedAt.UTC().Format(time.RFC3339Nano),
		"dedup_key":   "alert:" + firstAlert(in.AlertIDs),
	}
	corr := src.CorrelationID
	if corr == uuid.Nil {
		corr = in.ID
	}
	env := events.Build(events.TypeCreated, events.CreatedEventID(in.ID), corr, src.EventID, in.DetectedAt, payload)
	return p.publish(ctx, events.TopicCreated, in.ID.String(), env)
}

func (p *Processor) publishUpdated(ctx context.Context, src events.Envelope, in database.IncidentRow) error {
	payload := map[string]any{
		"incident_id":     in.ID.String(),
		"version":         in.Version,
		"status":          in.Status,
		"previous_status": in.Status,
		"changed_fields":  []string{"severity", "summary", "alert_ids"},
		"reason":          "alert attached by correlation",
		"actor_user_id":   nil,
	}
	corr := src.CorrelationID
	if corr == uuid.Nil {
		corr = in.ID
	}
	env := events.Build(events.TypeUpdated, events.UpdatedEventID(in.ID, in.Version), corr, src.EventID, p.now().UTC(), payload)
	return p.publish(ctx, events.TopicUpdated, in.ID.String(), env)
}

func (p *Processor) publish(ctx context.Context, topic, key string, env events.Envelope) error {
	body, err := json.Marshal(env)
	if err != nil {
		return err
	}
	if err := p.pub.Publish(ctx, kafka.Message{Topic: topic, Key: key, Value: body}); err != nil {
		return err
	}
	p.log.Info("incident_published",
		"event_id", env.EventID,
		"event_type", env.EventType,
		"incident_id", key,
		"kafka_topic", topic,
	)
	return nil
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func firstAlert(ids []uuid.UUID) string {
	if len(ids) == 0 {
		return ""
	}
	return ids[0].String()
}
