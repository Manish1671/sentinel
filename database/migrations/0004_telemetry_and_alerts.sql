CREATE TABLE telemetry_events (
  id uuid PRIMARY KEY,
  service_id uuid NOT NULL REFERENCES services (id),
  kind telemetry_kind NOT NULL,
  source text NOT NULL,
  occurred_at timestamptz NOT NULL,
  ingested_at timestamptz NOT NULL DEFAULT now(),
  correlation_id uuid,
  payload jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX telemetry_events_service_time_idx
  ON telemetry_events (service_id, occurred_at DESC);
CREATE INDEX telemetry_events_kind_time_idx
  ON telemetry_events (kind, occurred_at DESC);

COMMENT ON TABLE telemetry_events IS
  'Owned by services/ingestion. Retained samples only; Kafka is the hot path.';

CREATE TABLE alerts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  service_id uuid NOT NULL REFERENCES services (id),
  detector_id text NOT NULL,
  severity severity_level NOT NULL,
  status alert_status NOT NULL DEFAULT 'open',
  title text NOT NULL,
  summary text NOT NULL DEFAULT '',
  fingerprint text NOT NULL,
  labels jsonb NOT NULL DEFAULT '{}'::jsonb,
  triggering_event_id uuid REFERENCES telemetry_events (id),
  started_at timestamptz NOT NULL,
  ended_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT alerts_ended_after_start CHECK (
    ended_at IS NULL OR ended_at >= started_at
  )
);

CREATE UNIQUE INDEX alerts_open_fingerprint_uidx
  ON alerts (service_id, fingerprint)
  WHERE status IN ('open', 'acknowledged');

CREATE INDEX alerts_service_status_idx ON alerts (service_id, status);
CREATE INDEX alerts_started_idx ON alerts (started_at DESC);

CREATE TRIGGER alerts_set_updated_at
  BEFORE UPDATE ON alerts
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE alerts IS 'Owned by services/detection.';
