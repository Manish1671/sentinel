package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

func DBOp(ctx context.Context, operation string, fn func(context.Context) error) error {
	ctx, span := Start(ctx, "sentinel.db."+operation, attribute.String("db.operation", operation))
	start := time.Now()
	err := fn(ctx)
	Observe(ctx, DBDuration, time.Since(start).Seconds(), "operation", operation)
	if err != nil {
		Count(ctx, DBErrors, "operation", operation)
		End(span, err)
		return err
	}
	End(span, nil)
	return nil
}
