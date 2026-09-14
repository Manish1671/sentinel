package events

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	Source           = "services.ingestion"
	EventVersion     = 1
	TypeMetric       = "telemetry.metric"
	TypeLog          = "telemetry.log"
	TypeTrace        = "telemetry.trace"
	TypeDeployment   = "deployment.created"
	TopicTelemetry   = "telemetry.events"
	TopicDeployments = "deployments.created"
	DefaultTenant    = "default"
)

type Envelope struct {
	EventID        uuid.UUID      `json:"event_id"`
	EventType      string         `json:"event_type"`
	EventVersion   int            `json:"event_version"`
	OccurredAt     time.Time      `json:"occurred_at"`
	Source         string         `json:"source"`
	CorrelationID  uuid.UUID      `json:"correlation_id"`
	CausationID    *uuid.UUID     `json:"causation_id"`
	TenantID       string         `json:"tenant_id,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Payload        map[string]any `json:"payload"`
}

func (e Envelope) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		EventID        uuid.UUID      `json:"event_id"`
		EventType      string         `json:"event_type"`
		EventVersion   int            `json:"event_version"`
		OccurredAt     string         `json:"occurred_at"`
		Source         string         `json:"source"`
		CorrelationID  uuid.UUID      `json:"correlation_id"`
		CausationID    *uuid.UUID     `json:"causation_id"`
		TenantID       string         `json:"tenant_id,omitempty"`
		IdempotencyKey string         `json:"idempotency_key,omitempty"`
		Payload        map[string]any `json:"payload"`
	}{
		EventID:        e.EventID,
		EventType:      e.EventType,
		EventVersion:   e.EventVersion,
		OccurredAt:     e.OccurredAt.UTC().Format(time.RFC3339Nano),
		Source:         e.Source,
		CorrelationID:  e.CorrelationID,
		CausationID:    e.CausationID,
		TenantID:       e.TenantID,
		IdempotencyKey: e.IdempotencyKey,
		Payload:        e.Payload,
	})
}

func TopicFor(eventType string) string {
	if eventType == TypeDeployment {
		return TopicDeployments
	}
	return TopicTelemetry
}

func Build(eventType string, eventID, correlationID uuid.UUID, occurredAt time.Time, idempotencyKey string, payload map[string]any) Envelope {
	if eventID == uuid.Nil {
		eventID = uuid.New()
	}
	if correlationID == uuid.Nil {
		correlationID = uuid.New()
	}
	if idempotencyKey == "" {
		idempotencyKey = eventID.String()
	}
	return Envelope{
		EventID:        eventID,
		EventType:      eventType,
		EventVersion:   EventVersion,
		OccurredAt:     occurredAt.UTC(),
		Source:         Source,
		CorrelationID:  correlationID,
		CausationID:    nil,
		TenantID:       DefaultTenant,
		IdempotencyKey: idempotencyKey,
		Payload:        payload,
	}
}

func CanonicalHash(eventType string, payload map[string]any) (string, error) {
	body, err := json.Marshal(struct {
		Type    string         `json:"event_type"`
		Payload map[string]any `json:"payload"`
	}{Type: eventType, Payload: payload})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
