package incidents

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/apps/api/internal/apierr"
	"github.com/sentinel-dev/sentinel/apps/api/internal/paging"
)

type Incident struct {
	ID              uuid.UUID
	Reference       string
	ServiceID       uuid.UUID
	Title           string
	Summary         string
	Severity        string
	Status          string
	CreatedByUserID *uuid.UUID
	CommanderUserID *uuid.UUID
	DetectedAt      time.Time
	ResolvedAt      *time.Time
	ClosedAt        *time.Time
	Version         int
	AlertIDs        []uuid.UUID
}

type TimelineEvent struct {
	ID          uuid.UUID
	Kind        string
	ActorUserID *uuid.UUID
	OccurredAt  time.Time
	Payload     map[string]any
	Source      string
	Summary     string
	Metadata    map[string]any
}

type ListFilter struct {
	Status    string
	ServiceID string
	Severity  string
	Limit     int
	Cursor    string
}

type CreateInput struct {
	ServiceID uuid.UUID
	Title     string
	Summary   string
	Severity  string
	AlertIDs  []uuid.UUID
}

func validateCreate(in CreateInput) error {
	var fields []map[string]string
	if in.ServiceID == uuid.Nil {
		fields = append(fields, apierr.Field("service_id", "required", "service_id is required"))
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		fields = append(fields, apierr.Field("title", "required", "title is required"))
	} else if utf8.RuneCountInString(title) > 200 {
		fields = append(fields, apierr.Field("title", "max_length", "title must be at most 200 characters"))
	}
	if !oneOf(in.Severity, "critical", "high", "medium", "low") {
		fields = append(fields, apierr.Field("severity", "invalid", "severity is invalid"))
	}
	if len(fields) > 0 {
		return apierr.Validation(fields)
	}
	return nil
}

func validateList(filter ListFilter) (uuid.UUID, error) {
	var fields []map[string]string
	if filter.Status != "" && !oneOf(filter.Status, "open", "investigating", "remediating", "verifying", "resolved", "closed") {
		fields = append(fields, apierr.Field("status", "invalid", "status is invalid"))
	}
	if filter.Severity != "" && !oneOf(filter.Severity, "critical", "high", "medium", "low") {
		fields = append(fields, apierr.Field("severity", "invalid", "severity is invalid"))
	}
	var serviceID uuid.UUID
	if filter.ServiceID != "" {
		id, err := uuid.Parse(filter.ServiceID)
		if err != nil {
			fields = append(fields, apierr.Field("service_id", "invalid", "service_id must be a UUID"))
		} else {
			serviceID = id
		}
	}
	if len(fields) > 0 {
		return uuid.Nil, apierr.Validation(fields)
	}
	return serviceID, nil
}

func requestHash(in CreateInput) (string, error) {
	ids := make([]string, 0, len(in.AlertIDs))
	for _, id := range in.AlertIDs {
		ids = append(ids, id.String())
	}
	sort.Strings(ids)
	payload := struct {
		ServiceID string   `json:"service_id"`
		Title     string   `json:"title"`
		Summary   string   `json:"summary"`
		Severity  string   `json:"severity"`
		AlertIDs  []string `json:"alert_ids"`
	}{
		ServiceID: in.ServiceID.String(),
		Title:     strings.TrimSpace(in.Title),
		Summary:   strings.TrimSpace(in.Summary),
		Severity:  in.Severity,
		AlertIDs:  ids,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func validateIdempotencyKey(key string) error {
	if len(key) < 8 || len(key) > 128 {
		return apierr.Validation([]map[string]string{
			apierr.Field("Idempotency-Key", "invalid", "Idempotency-Key must be 8-128 characters"),
		})
	}
	for _, c := range key {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return apierr.Validation([]map[string]string{
				apierr.Field("Idempotency-Key", "invalid", "Idempotency-Key contains invalid characters"),
			})
		}
	}
	return nil
}

func decorateTimeline(ev TimelineEvent) TimelineEvent {
	ev.Metadata = ev.Payload
	if ev.Payload != nil {
		if src, ok := ev.Payload["source"].(string); ok {
			ev.Source = src
		}
	}
	if ev.Source == "" {
		if ev.ActorUserID != nil {
			ev.Source = "user"
		} else {
			ev.Source = "system"
		}
	}
	switch ev.Kind {
	case "created":
		ev.Summary = "Incident opened"
	case "status_changed":
		ev.Summary = "Status changed"
	case "alert_attached":
		ev.Summary = "Alert attached"
	case "comment":
		ev.Summary = "Comment added"
	case "commander_changed":
		ev.Summary = "Commander changed"
	case "investigation_requested":
		ev.Summary = "Investigation requested"
	case "investigation_completed":
		ev.Summary = "Investigation completed"
	case "recommendation_added":
		ev.Summary = "Recommendation added"
	case "approval_recorded":
		ev.Summary = "Approval recorded"
	case "remediation_requested":
		ev.Summary = "Remediation requested"
	case "remediation_completed":
		ev.Summary = "Remediation completed"
	case "remediation_failed":
		ev.Summary = "Remediation failed"
	default:
		ev.Summary = ev.Kind
	}
	return ev
}

func nextListPage(limit int, rows []Incident) paging.Page {
	page := paging.Page{Limit: limit}
	if len(rows) == limit {
		last := rows[len(rows)-1]
		c := paging.EncodeTimeID(last.DetectedAt, last.ID)
		page.NextCursor = &c
	}
	return page
}

func nextTimelinePage(limit int, rows []TimelineEvent) paging.Page {
	page := paging.Page{Limit: limit}
	if len(rows) == limit {
		last := rows[len(rows)-1]
		c := paging.EncodeTimeID(last.OccurredAt, last.ID)
		page.NextCursor = &c
	}
	return page
}

func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}
