CREATE TABLE incidents (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  reference text NOT NULL,
  service_id uuid NOT NULL REFERENCES services (id),
  title text NOT NULL,
  summary text NOT NULL DEFAULT '',
  severity severity_level NOT NULL,
  status incident_status NOT NULL DEFAULT 'open',
  created_by_user_id uuid REFERENCES users (id),
  commander_user_id uuid REFERENCES users (id),
  detected_at timestamptz NOT NULL,
  resolved_at timestamptz,
  closed_at timestamptz,
  dedup_key text,
  version integer NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT incidents_reference_format CHECK (reference ~ '^INC-[0-9]{4}-[0-9]{4}$'),
  CONSTRAINT incidents_version_positive CHECK (version >= 1),
  CONSTRAINT incidents_resolved_when_needed CHECK (
    (status IN ('resolved', 'closed') AND resolved_at IS NOT NULL)
    OR (status NOT IN ('resolved', 'closed') AND resolved_at IS NULL)
  ),
  CONSTRAINT incidents_closed_when_needed CHECK (
    (status = 'closed' AND closed_at IS NOT NULL)
    OR (status <> 'closed' AND closed_at IS NULL)
  )
);

CREATE UNIQUE INDEX incidents_reference_uidx ON incidents (reference);
CREATE UNIQUE INDEX incidents_dedup_key_uidx ON incidents (dedup_key)
  WHERE dedup_key IS NOT NULL;
CREATE INDEX incidents_status_detected_idx ON incidents (status, detected_at DESC);
CREATE INDEX incidents_service_status_idx ON incidents (service_id, status);

CREATE TRIGGER incidents_set_updated_at
  BEFORE UPDATE ON incidents
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE incidents IS 'Owned by services/incident.';

CREATE TABLE incident_alerts (
  incident_id uuid NOT NULL REFERENCES incidents (id),
  alert_id uuid NOT NULL REFERENCES alerts (id),
  attached_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (incident_id, alert_id)
);

CREATE INDEX incident_alerts_alert_idx ON incident_alerts (alert_id);

CREATE TABLE incident_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  incident_id uuid NOT NULL REFERENCES incidents (id),
  kind incident_event_kind NOT NULL,
  actor_user_id uuid REFERENCES users (id),
  occurred_at timestamptz NOT NULL DEFAULT now(),
  payload jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX incident_events_incident_time_idx
  ON incident_events (incident_id, occurred_at);

COMMENT ON TABLE incident_events IS 'Append-only incident timeline. Owned by services/incident.';

CREATE VIEW historical_incidents AS
SELECT
  i.id,
  i.reference,
  i.service_id,
  i.title,
  i.summary,
  i.severity,
  i.detected_at,
  i.resolved_at,
  i.closed_at,
  i.commander_user_id
FROM incidents i
WHERE i.status IN ('resolved', 'closed');

COMMENT ON VIEW historical_incidents IS
  'Read model of resolved/closed incidents. Not a separate write table.';
