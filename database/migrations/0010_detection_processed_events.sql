-- Phase 4: Detection durable event processing.
--
-- alerts.triggering_event_id referenced telemetry_events(id). Ingestion publishes
-- to Kafka and does not insert a telemetry_events row on the hot path, so Kafka
-- event_id values cannot satisfy that FK. The column is kept; the FK is dropped.
--
-- detection_processed_events is the durable source-event_id ledger for at-least-once
-- Kafka. Redis is not used for correctness.

ALTER TABLE alerts DROP CONSTRAINT IF EXISTS alerts_triggering_event_id_fkey;

CREATE TABLE detection_processed_events (
  event_id uuid PRIMARY KEY,
  event_type text NOT NULL,
  outcome text NOT NULL,
  alert_ids uuid[] NOT NULL DEFAULT '{}',
  published boolean NOT NULL DEFAULT false,
  processed_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX detection_processed_events_processed_idx
  ON detection_processed_events (processed_at DESC);

COMMENT ON TABLE detection_processed_events IS
  'Owned by services/detection. Deduplicates Kafka event_id processing.';
