package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	Source         = "services.incident"
	EventVersion   = 1
	TypeAlert      = "alert.created"
	TypeCreated    = "incident.created"
	TypeUpdated    = "incident.updated"
	TopicAlerts    = "alerts.created"
	TopicCreated   = "incidents.created"
	TopicUpdated   = "incidents.updated"
	DefaultTenant  = "default"
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

func UUIDField(payload map[string]any, key string) (uuid.UUID, error) {
	raw, _ := payload[key].(string)
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, fmt.Errorf("payload.%s is required", key)
	}
	return id, nil
}

func StringField(payload map[string]any, key string) string {
	v, _ := payload[key].(string)
	return strings.TrimSpace(v)
}

func Labels(payload map[string]any) map[string]string {
	out := map[string]string{}
	raw, ok := payload["labels"]
	if !ok {
		return out
	}
	switch m := raw.(type) {
	case map[string]string:
		return m
	case map[string]any:
		for k, v := range m {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

func Build(eventType string, eventID, corr uuid.UUID, cause uuid.UUID, occurred time.Time, payload map[string]any) Envelope {
	return Envelope{
		EventID:        eventID,
		EventType:      eventType,
		EventVersion:   EventVersion,
		OccurredAt:     occurred.UTC(),
		Source:         Source,
		CorrelationID:  corr,
		CausationID:    &cause,
		TenantID:       DefaultTenant,
		IdempotencyKey: eventType + ":" + eventID.String(),
		Payload:        payload,
	}
}

func CreatedEventID(incidentID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("incident.created:"+incidentID.String()))
}

func UpdatedEventID(incidentID uuid.UUID, version int) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("incident.updated:%s:%d", incidentID, version)))
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
