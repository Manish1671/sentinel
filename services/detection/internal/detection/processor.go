package detection

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/services/detection/internal/config"
	"github.com/sentinel-dev/sentinel/services/detection/internal/database"
	"github.com/sentinel-dev/sentinel/services/detection/internal/events"
	"github.com/sentinel-dev/sentinel/services/detection/internal/kafka"
	"github.com/sentinel-dev/sentinel/services/detection/internal/rules"
)

type Processor struct {
	cfg    config.Config
	log    *slog.Logger
	store  *database.Store
	engine *rules.Engine
	pub    kafka.Publisher
	now    func() time.Time
}

func NewProcessor(cfg config.Config, log *slog.Logger, store *database.Store, engine *rules.Engine, pub kafka.Publisher) *Processor {
	return &Processor{cfg: cfg, log: log, store: store, engine: engine, pub: pub, now: time.Now}
}

type HandleResult struct {
	Commit     bool
	Outcome    string
	EventID    uuid.UUID
	EventType  string
	ServiceID  uuid.UUID
	AlertIDs   []uuid.UUID
	NewPublish int
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

	res := HandleResult{EventID: env.EventID, EventType: env.EventType, Commit: true}
	svcID, svcErr := events.ServiceID(env.Payload)

	p.log.Info("detection_received",
		"event_id", env.EventID,
		"event_type", env.EventType,
		"service_id", svcID,
		"kafka_topic", topic,
		"partition", partition,
		"offset", offset,
	)

	prev, err := p.store.GetProcessed(ctx, env.EventID)
	if err != nil {
		return HandleResult{}, err
	}
	if prev.Exists {
		if prev.Published || prev.Outcome != "created" {
			p.log.Info("event_duplicate",
				"event_id", env.EventID,
				"outcome", prev.Outcome,
				"published", prev.Published,
			)
			res.Outcome = "duplicate"
			res.AlertIDs = prev.AlertIDs
			return res, nil
		}
		if err := p.publishAlerts(ctx, env, prev.AlertIDs); err != nil {
			return HandleResult{}, err
		}
		if err := p.store.MarkPublished(ctx, env.EventID); err != nil {
			return HandleResult{}, err
		}
		res.Outcome = "republished"
		res.AlertIDs = prev.AlertIDs
		res.NewPublish = len(prev.AlertIDs)
		return res, nil
	}

	switch env.EventType {
	case events.TypeMetric, events.TypeLog, events.TypeTrace, events.TypeDeployment:
	default:
		if err := p.store.RecordProcessed(ctx, env.EventID, env.EventType, "ignored_type", nil, true); err != nil {
			return HandleResult{}, err
		}
		res.Outcome = "ignored_type"
		return res, nil
	}

	if svcErr != nil {
		if err := p.store.RecordProcessed(ctx, env.EventID, env.EventType, "invalid_service", nil, true); err != nil {
			return HandleResult{}, err
		}
		p.log.Error("invalid_service_identity", "event_id", env.EventID, "error", svcErr.Error())
		res.Outcome = "invalid_service"
		return res, nil
	}
	res.ServiceID = svcID

	exists, err := p.store.ServiceExists(ctx, svcID)
	if err != nil {
		return HandleResult{}, err
	}
	if !exists {
		p.engine.Evaluate(env)
		if err := p.store.RecordProcessed(ctx, env.EventID, env.EventType, "unknown_service", nil, true); err != nil {
			return HandleResult{}, err
		}
		p.log.Info("unknown_service", "event_id", env.EventID, "service_id", svcID)
		res.Outcome = "unknown_service"
		return res, nil
	}

	hits := p.engine.Evaluate(env)
	if len(hits) == 0 {
		if err := p.store.RecordProcessed(ctx, env.EventID, env.EventType, "no_match", nil, true); err != nil {
			return HandleResult{}, err
		}
		res.Outcome = "no_match"
		p.log.Info("detection_complete",
			"event_id", env.EventID,
			"event_type", env.EventType,
			"service_id", svcID,
			"outcome", res.Outcome,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return res, nil
	}

	var created []uuid.UUID
	var all []uuid.UUID
	for _, hit := range hits {
		id, isNew, err := p.persistHit(ctx, env, svcID, hit)
		if err != nil {
			return HandleResult{}, err
		}
		if id == uuid.Nil {
			continue
		}
		all = append(all, id)
		if isNew {
			created = append(created, id)
		}
		p.log.Info("rule_fired",
			"event_id", env.EventID,
			"service_id", svcID,
			"rule", hit.DetectorID,
			"alert_id", id,
			"severity", hit.Severity,
			"new", isNew,
		)
	}

	outcome := "suppressed"
	if len(created) > 0 {
		outcome = "created"
	}
	published := len(created) == 0
	if err := p.store.RecordProcessed(ctx, env.EventID, env.EventType, outcome, created, published); err != nil {
		return HandleResult{}, err
	}
	if len(created) > 0 {
		if err := p.publishAlerts(ctx, env, created); err != nil {
			return HandleResult{}, err
		}
		if err := p.store.MarkPublished(ctx, env.EventID); err != nil {
			return HandleResult{}, err
		}
	}
	res.Outcome = outcome
	res.AlertIDs = all
	res.NewPublish = len(created)
	p.log.Info("detection_complete",
		"event_id", env.EventID,
		"event_type", env.EventType,
		"service_id", svcID,
		"outcome", outcome,
		"alert_ids", fmt.Sprint(all),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return res, nil
}

func (p *Processor) persistHit(ctx context.Context, env events.Envelope, svcID uuid.UUID, hit rules.Hit) (uuid.UUID, bool, error) {
	existing, ok, err := p.store.LatestAlert(ctx, svcID, hit.Fingerprint)
	if err != nil {
		return uuid.Nil, false, err
	}
	if ok && (existing.Status == "open" || existing.Status == "acknowledged") {
		_ = p.store.TouchAlert(ctx, existing.ID, hit.Summary, hit.Labels)
		return existing.ID, false, nil
	}
	if ok && p.now().UTC().Sub(existing.StartedAt) < p.cfg.Rules.AlertCooldown {
		return existing.ID, false, nil
	}
	id := uuid.New()
	err = p.store.InsertAlert(ctx, database.InsertAlert{
		ID:                id,
		ServiceID:         svcID,
		DetectorID:        hit.DetectorID,
		Severity:          hit.Severity,
		Title:             hit.Title,
		Summary:           hit.Summary,
		Fingerprint:       hit.Fingerprint,
		Labels:            hit.Labels,
		TriggeringEventID: env.EventID,
		StartedAt:         env.OccurredAt,
	})
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}

func (p *Processor) publishAlerts(ctx context.Context, src events.Envelope, ids []uuid.UUID) error {
	for _, id := range ids {
		a, err := p.store.AlertByID(ctx, id)
		if err != nil {
			return err
		}
		payload := map[string]any{
			"alert_id":            a.ID.String(),
			"service_id":          a.ServiceID.String(),
			"detector_id":         a.DetectorID,
			"severity":            a.Severity,
			"title":               a.Title,
			"summary":             a.Summary,
			"fingerprint":         a.Fingerprint,
			"labels":              a.Labels,
			"triggering_event_id": a.TriggeringEventID.String(),
			"started_at":          a.StartedAt.UTC().Format(time.RFC3339Nano),
		}
		corr := src.CorrelationID
		if corr == uuid.Nil {
			corr = uuid.New()
		}
		env := events.BuildAlert(a.ID, a.ServiceID, corr, src.EventID, a.StartedAt, payload)
		body, err := json.Marshal(env)
		if err != nil {
			return err
		}
		if err := p.pub.Publish(ctx, kafka.Message{
			Topic: events.TopicAlerts,
			Key:   a.ServiceID.String(),
			Value: body,
		}); err != nil {
			return err
		}
		p.log.Info("alert_published",
			"event_id", src.EventID,
			"alert_id", a.ID,
			"service_id", a.ServiceID,
			"rule", a.DetectorID,
			"severity", a.Severity,
			"kafka_topic", events.TopicAlerts,
		)
	}
	return nil
}
