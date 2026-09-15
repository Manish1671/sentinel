-- Phase 5: durable alert.created processing for services/incident.
--
-- incident_alerts already prevents the same alert from attaching to the same
-- incident twice (composite PK). A unique alert_id index ensures one alert
-- belongs to at most one incident. incident_processed_events is the at-least-once
-- Kafka ledger keyed by source event_id.

CREATE UNIQUE INDEX IF NOT EXISTS incident_alerts_alert_uidx
  ON incident_alerts (alert_id);

CREATE UNIQUE INDEX IF NOT EXISTS incident_events_alert_attached_uidx
  ON incident_events (incident_id, ((payload->>'alert_id')))
  WHERE kind = 'alert_attached' AND payload ? 'alert_id';

CREATE TABLE incident_processed_events (
  event_id uuid PRIMARY KEY,
  alert_id uuid,
  incident_id uuid REFERENCES incidents (id),
  outcome text NOT NULL,
  published boolean NOT NULL DEFAULT false,
  processed_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX incident_processed_events_alert_idx
  ON incident_processed_events (alert_id);

COMMENT ON TABLE incident_processed_events IS
  'Owned by services/incident. Deduplicates Kafka alert.created processing.';
