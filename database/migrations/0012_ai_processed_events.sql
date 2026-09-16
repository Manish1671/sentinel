-- Phase 6: durable investigation.requested processing for services/ai.
--
-- investigations.idempotency_key already prevents duplicate request rows.
-- Evidence and recommendation inserts use deterministic UUIDs (application-side).
-- ai_processed_events is the at-least-once Kafka ledger keyed by source event_id.

CREATE TABLE ai_processed_events (
  event_id uuid PRIMARY KEY,
  investigation_id uuid NOT NULL REFERENCES investigations (id),
  outcome text NOT NULL,
  published boolean NOT NULL DEFAULT false,
  processed_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ai_processed_events_investigation_idx
  ON ai_processed_events (investigation_id);

COMMENT ON TABLE ai_processed_events IS
  'Owned by services/ai. Deduplicates Kafka investigation.requested processing.';
