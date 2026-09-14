package paging

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sentinel-dev/sentinel/apps/api/internal/apierr"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type Page struct {
	Limit      int
	NextCursor *string
}

func ParseLimit(raw string) (int, error) {
	if raw == "" {
		return DefaultLimit, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, apierr.Validation([]map[string]string{
			apierr.Field("limit", "invalid", "limit must be a positive integer"),
		})
	}
	if n > MaxLimit {
		n = MaxLimit
	}
	return n, nil
}

func EncodeTimeID(t time.Time, id uuid.UUID) string {
	raw := t.UTC().Format(time.RFC3339Nano) + "|" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeTimeID(cursor string) (time.Time, uuid.UUID, error) {
	invalid := apierr.Validation([]map[string]string{
		apierr.Field("cursor", "invalid", "cursor is invalid"),
	})
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, invalid
	}
	parts := strings.Split(string(b), "|")
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, invalid
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, invalid
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, invalid
	}
	return t, id, nil
}

func Next(limit int, n int, lastTime time.Time, lastID uuid.UUID) Page {
	page := Page{Limit: limit}
	if n == limit {
		c := EncodeTimeID(lastTime, lastID)
		page.NextCursor = &c
	}
	return page
}

func RequireUUID(raw, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apierr.Validation([]map[string]string{
			apierr.Field(field, "invalid", fmt.Sprintf("%s must be a UUID", field)),
		})
	}
	return id, nil
}
