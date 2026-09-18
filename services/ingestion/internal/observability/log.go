package observability

import (
	"log/slog"

	"github.com/sentinel-dev/sentinel/packages/telemetry"
)

func NewLogger(level string) *slog.Logger {
	return telemetry.NewLogger(level, telemetry.ServiceName(), telemetry.Environment())
}
