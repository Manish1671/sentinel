package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/apps/api/internal/apierr"
	"github.com/sentinel-dev/sentinel/apps/api/internal/middleware"
)

type DecisionClient interface {
	Decide(ctx context.Context, id uuid.UUID, approve bool, token, idempotencyKey, comment, requestID string) (json.RawMessage, error)
}

type HTTPDecisionClient struct {
	base   string
	client *http.Client
}

func NewHTTPDecisionClient(baseURL string) *HTTPDecisionClient {
	return &HTTPDecisionClient{
		base: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *HTTPDecisionClient) Decide(ctx context.Context, id uuid.UUID, approve bool, token, idempotencyKey, comment, requestID string) (json.RawMessage, error) {
	if c == nil || c.base == "" {
		return nil, apierr.Unavailable("The remediation service is unavailable.")
	}
	action := "reject"
	if approve {
		action = "approve"
	}
	body, _ := json.Marshal(map[string]string{"comment": comment})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/v1/remediations/"+id.String()+"/"+action, bytes.NewReader(body))
	if err != nil {
		return nil, apierr.Unavailable("The remediation service is unavailable.")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", idempotencyKey)
	if requestID != "" {
		req.Header.Set("X-Request-Id", requestID)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, apierr.Unavailable("The remediation service is unavailable.")
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Data) == 0 {
			return json.RawMessage(raw), nil
		}
		return envelope.Data, nil
	}
	return nil, mapDownstreamError(res.StatusCode, raw)
}

func mapDownstreamError(status int, raw []byte) error {
	var env struct {
		Error any `json:"error"`
	}
	_ = json.Unmarshal(raw, &env)
	code := ""
	switch v := env.Error.(type) {
	case string:
		code = v
	case map[string]any:
		if c, ok := v["code"].(string); ok {
			code = c
		}
	}
	switch {
	case status == http.StatusUnauthorized:
		return apierr.Unauthenticated()
	case status == http.StatusForbidden:
		return apierr.Forbidden()
	case status == http.StatusNotFound:
		return apierr.NotFound("Remediation not found.")
	case status == http.StatusConflict && code == "idempotency_key_conflict":
		return apierr.IdempotencyConflict()
	case status == http.StatusConflict:
		return apierr.ApprovalAlreadyDecided()
	case status == http.StatusBadRequest:
		return apierr.Validation([]map[string]string{
			apierr.Field("body", "invalid", "Remediation request was rejected."),
		})
	case status >= 500:
		return apierr.Unavailable("The remediation service is unavailable.")
	default:
		return apierr.Unavailable(fmt.Sprintf("Remediation service returned HTTP %d.", status))
	}
}

func RequestIDFrom(ctx context.Context) string {
	return middleware.RequestIDFrom(ctx)
}
