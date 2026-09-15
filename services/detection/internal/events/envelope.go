package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	Source           = "services.detection"
	EventVersion     = 1
	TypeMetric       = "telemetry.metric"
	TypeLog          = "telemetry.log"
	TypeTrace        = "telemetry.trace"
	TypeDeployment   = "deployment.created"
	TypeAlert        = "alert.created"
	TopicTelemetry   = "telemetry.events"
	TopicDeployments = "deployments.created"
	TopicAlerts      = "alerts.created"
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

func Parse(raw []byte) (Envelope, error) {
	var aux struct {
		EventID        string          `json:"event_id"`
		EventType      string          `json:"event_type"`
		EventVersion   int             `json:"event_version"`
		OccurredAt     string          `json:"occurred_at"`
		Source         string          `json:"source"`
		CorrelationID  string          `json:"correlation_id"`
		CausationID    *string         `json:"causation_id"`
		TenantID       string          `json:"tenant_id"`
		IdempotencyKey string          `json:"idempotency_key"`
		Payload        json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(raw, &aux); err != nil {
		return Envelope{}, fmt.Errorf("malformed envelope json")
	}
	if strings.TrimSpace(aux.EventID) == "" {
		return Envelope{}, fmt.Errorf("missing event_id")
	}
	eid, err := uuid.Parse(aux.EventID)
	if err != nil {
		return Envelope{}, fmt.Errorf("invalid event_id")
	}
	if strings.TrimSpace(aux.EventType) == "" {
		return Envelope{}, fmt.Errorf("missing event_type")
	}
	occurred, err := time.Parse(time.RFC3339Nano, aux.OccurredAt)
	if err != nil {
		occurred, err = time.Parse(time.RFC3339, aux.OccurredAt)
	}
	if err != nil {
		return Envelope{}, fmt.Errorf("invalid occurred_at")
	}
	var corr uuid.UUID
	if strings.TrimSpace(aux.CorrelationID) != "" {
		corr, err = uuid.Parse(aux.CorrelationID)
		if err != nil {
			return Envelope{}, fmt.Errorf("invalid correlation_id")
		}
	}
	payload := map[string]any{}
	if len(aux.Payload) > 0 && string(aux.Payload) != "null" {
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return Envelope{}, fmt.Errorf("invalid payload")
		}
	}
	var causation *uuid.UUID
	if aux.CausationID != nil && strings.TrimSpace(*aux.CausationID) != "" {
		id, err := uuid.Parse(*aux.CausationID)
		if err != nil {
			return Envelope{}, fmt.Errorf("invalid causation_id")
		}
		causation = &id
	}
	return Envelope{
		EventID:        eid,
		EventType:      aux.EventType,
		EventVersion:   aux.EventVersion,
		OccurredAt:     occurred.UTC(),
		Source:         aux.Source,
		CorrelationID:  corr,
		CausationID:    causation,
		TenantID:       aux.TenantID,
		IdempotencyKey: aux.IdempotencyKey,
		Payload:        payload,
	}, nil
}

func ServiceID(payload map[string]any) (uuid.UUID, error) {
	raw, _ := payload["service_id"].(string)
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, fmt.Errorf("payload.service_id is required")
	}
	return id, nil
}

func StringField(payload map[string]any, key string) string {
	v, _ := payload[key].(string)
	return strings.TrimSpace(v)
}

func FloatField(payload map[string]any, key string) (float64, bool) {
	switch v := payload[key].(type) {
	case float64:
		return v, true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	default:
		return 0, false
	}
}

func BuildAlert(alertID, serviceID, corr uuid.UUID, cause uuid.UUID, occurred time.Time, payload map[string]any) Envelope {
	eventID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("alert.created:"+alertID.String()))
	return Envelope{
		EventID:        eventID,
		EventType:      TypeAlert,
		EventVersion:   EventVersion,
		OccurredAt:     occurred.UTC(),
		Source:         Source,
		CorrelationID:  corr,
		CausationID:    &cause,
		TenantID:       DefaultTenant,
		IdempotencyKey: "alert:" + alertID.String(),
		Payload:        payload,
	}
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
