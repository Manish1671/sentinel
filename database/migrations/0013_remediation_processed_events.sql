-- Phase 7: Kafka ledger and local simulator world for services/remediation.
--
-- remediations / approvals / recommendations / incident_events are unchanged.
-- remediation_processed_events deduplicates at-least-once Kafka (investigation.completed
-- and remediation.requested) the same way detection/incident/ai ledgers do.
--
-- remediation_simulator_state holds explicit simulated version/health/replicas/metrics.
-- The catalog (services.health_status) is updated to match so operators see recovery,
-- but no shell, kubectl, or cloud APIs are used.

CREATE TABLE remediation_processed_events (
  event_id uuid PRIMARY KEY,
  event_type text NOT NULL,
  remediation_id uuid REFERENCES remediations (id),
  investigation_id uuid,
  outcome text NOT NULL,
  published boolean NOT NULL DEFAULT false,
  processed_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX remediation_processed_events_remediation_idx
  ON remediation_processed_events (remediation_id);

COMMENT ON TABLE remediation_processed_events IS
  'Owned by services/remediation. Deduplicates Kafka investigation.completed and remediation.requested.';

CREATE TABLE remediation_simulator_state (
  service_id uuid PRIMARY KEY REFERENCES services (id),
  current_version text NOT NULL DEFAULT 'unknown',
  previous_version text,
  health_status text NOT NULL DEFAULT 'unknown',
  replicas integer NOT NULL DEFAULT 2,
  error_rate double precision NOT NULL DEFAULT 0,
  latency_ms double precision NOT NULL DEFAULT 0,
  db_utilization double precision NOT NULL DEFAULT 0,
  last_action text,
  last_remediation_id uuid,
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT remediation_simulator_replicas_positive CHECK (replicas >= 1)
);

COMMENT ON TABLE remediation_simulator_state IS
  'Local controlled executor world. Not a production control plane.';
