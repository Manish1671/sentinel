package httpapi

import (
	"time"

	"github.com/sentinel-dev/sentinel/apps/api/internal/incidents"
	"github.com/sentinel-dev/sentinel/apps/api/internal/paging"
)

func pageDTO(page paging.Page) map[string]any {
	return map[string]any{
		"next_cursor": page.NextCursor,
		"limit":       page.Limit,
	}
}

func formatTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func incidentListDTO(item incidents.Incident) map[string]any {
	return map[string]any{
		"id":                item.ID,
		"reference":         item.Reference,
		"service_id":        item.ServiceID,
		"title":             item.Title,
		"severity":          item.Severity,
		"status":            item.Status,
		"commander_user_id": item.CommanderUserID,
		"detected_at":       item.DetectedAt.UTC().Format(time.RFC3339Nano),
		"version":           item.Version,
	}
}

func incidentDetailDTO(item incidents.Incident) map[string]any {
	return map[string]any{
		"id":                 item.ID,
		"reference":          item.Reference,
		"service_id":         item.ServiceID,
		"title":              item.Title,
		"summary":            item.Summary,
		"severity":           item.Severity,
		"status":             item.Status,
		"created_by_user_id": item.CreatedByUserID,
		"commander_user_id":  item.CommanderUserID,
		"detected_at":        item.DetectedAt.UTC().Format(time.RFC3339Nano),
		"resolved_at":        formatTimePtr(item.ResolvedAt),
		"closed_at":          formatTimePtr(item.ClosedAt),
		"version":            item.Version,
		"alert_ids":          item.AlertIDs,
	}
}
