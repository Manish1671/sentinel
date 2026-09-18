package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Header struct {
	Key   string
	Value []byte
}

type headerCarrier []Header

func (c *headerCarrier) Get(key string) string {
	for _, h := range *c {
		if stringsEqualFold(h.Key, key) {
			return string(h.Value)
		}
	}
	return ""
}

func (c *headerCarrier) Set(key, value string) {
	for i := range *c {
		if stringsEqualFold((*c)[i].Key, key) {
			(*c)[i].Value = []byte(value)
			return
		}
	}
	*c = append(*c, Header{Key: key, Value: []byte(value)})
}

func (c *headerCarrier) Keys() []string {
	out := make([]string, 0, len(*c))
	for _, h := range *c {
		out = append(out, h.Key)
	}
	return out
}

func stringsEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func InjectKafka(ctx context.Context, headers []Header) []Header {
	c := headerCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, &c)
	if rid := RequestIDFrom(ctx); rid != "" {
		c.Set("request_id", rid)
	}
	return []Header(c)
}

func ExtractKafka(ctx context.Context, headers []Header) context.Context {
	c := headerCarrier(headers)
	ctx = otel.GetTextMapPropagator().Extract(ctx, &c)
	if rid := c.Get("request_id"); rid != "" {
		ctx = WithRequestID(ctx, rid)
	}
	return ctx
}

func KafkaPublish(ctx context.Context, topic string, publish func(context.Context) error) error {
	ctx, span := Start(ctx, "sentinel.kafka.publish", attribute.String("messaging.destination", topic))
	defer span.End()
	err := publish(ctx)
	if err != nil {
		Count(ctx, KafkaFailures, "operation", "publish", "topic", topic)
		End(span, err)
		return err
	}
	Count(ctx, KafkaPublished, "topic", topic, "operation", "publish")
	End(span, nil)
	return nil
}

func KafkaConsume(ctx context.Context, topic string, consume func(context.Context) error) error {
	ctx, span := Start(ctx, "sentinel.kafka.consume", attribute.String("messaging.destination", topic))
	defer span.End()
	Count(ctx, KafkaConsumed, "topic", topic, "operation", "consume")
	err := consume(ctx)
	if err != nil {
		Count(ctx, KafkaFailures, "operation", "consume", "topic", topic)
		End(span, err)
		return err
	}
	End(span, nil)
	return nil
}
