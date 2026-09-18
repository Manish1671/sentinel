package kafka

import (
	"context"
	"fmt"
	"net"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/sentinel-dev/sentinel/packages/telemetry"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/config"
)

type Message struct {
	Topic string
	Key   string
	Value []byte
}

type Producer struct {
	writer *kafkago.Writer
	addrs  []string
}

func NewProducer(cfg config.Config) *Producer {
	return &Producer{
		addrs: cfg.KafkaBrokers,
		writer: &kafkago.Writer{
			Addr:                   kafkago.TCP(cfg.KafkaBrokers...),
			Balancer:               &kafkago.Hash{},
			RequiredAcks:           kafkago.RequireAll,
			AllowAutoTopicCreation: true,
			BatchTimeout:           10 * time.Millisecond,
			WriteTimeout:           cfg.PublishTimeout,
			ReadTimeout:            cfg.PublishTimeout,
			MaxAttempts:            3,
			Async:                  false,
			Transport: &kafkago.Transport{
				ClientID: cfg.KafkaClientID,
			},
		},
	}
}

func (p *Producer) Publish(ctx context.Context, msg Message) error {
	if p == nil || p.writer == nil {
		return fmt.Errorf("kafka producer is not configured")
	}
	return telemetry.KafkaPublish(ctx, msg.Topic, func(ctx context.Context) error {
		headers := toKafkaHeaders(telemetry.InjectKafka(ctx, nil))
		var last error
		for attempt := 1; attempt <= 5; attempt++ {
			last = p.writer.WriteMessages(ctx, kafkago.Message{
				Topic:   msg.Topic,
				Key:     []byte(msg.Key),
				Value:   msg.Value,
				Time:    time.Now().UTC(),
				Headers: headers,
			})
			if last == nil {
				return nil
			}
			if ctx.Err() != nil {
				return fmt.Errorf("publish %s: %w", msg.Topic, last)
			}
			time.Sleep(time.Duration(attempt) * 150 * time.Millisecond)
		}
		return fmt.Errorf("publish %s: %w", msg.Topic, last)
	})
}

func toKafkaHeaders(in []telemetry.Header) []kafkago.Header {
	out := make([]kafkago.Header, 0, len(in))
	for _, h := range in {
		out = append(out, kafkago.Header{Key: h.Key, Value: h.Value})
	}
	return out
}

func (p *Producer) Ready(ctx context.Context) error {
	if len(p.addrs) == 0 {
		return fmt.Errorf("kafka brokers are not configured")
	}
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", p.addrs[0])
	if err != nil {
		return err
	}
	return conn.Close()
}

func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
