package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sentinel-dev/sentinel/apps/api/internal/apierr"
	"github.com/sentinel-dev/sentinel/apps/api/internal/middleware"
)

type errorBody struct {
	Error errorFields `json:"error"`
}

type errorFields struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details"`
	RequestID string         `json:"request_id"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	requestID := middleware.RequestIDFrom(r.Context())
	api, ok := apierr.As(err)
	if !ok {
		if log != nil {
			log.Error("unhandled error", "request_id", requestID, "err", err)
		}
		api = apierr.Internal()
	}
	details := api.Details
	if details == nil {
		details = map[string]any{}
	}
	writeJSON(w, api.Status, errorBody{Error: errorFields{
		Code:      api.Code,
		Message:   api.Message,
		Details:   details,
		RequestID: requestID,
	}})
}

func decodeJSON(r *http.Request, dest any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return apierr.Validation([]map[string]string{
			apierr.Field("body", "invalid", "request body must be valid JSON"),
		})
	}
	return nil
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && h[:len(prefix)] == prefix {
		return h[len(prefix):]
	}
	return ""
}
