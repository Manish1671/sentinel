CREATE TABLE remediations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  incident_id uuid NOT NULL REFERENCES incidents (id),
  recommendation_id uuid NOT NULL REFERENCES recommendations (id),
  service_id uuid NOT NULL REFERENCES services (id),
  status remediation_status NOT NULL DEFAULT 'pending_approval',
  action_type text NOT NULL,
  parameters jsonb NOT NULL DEFAULT '{}'::jsonb,
  requested_by_user_id uuid REFERENCES users (id),
  attempt_number integer NOT NULL DEFAULT 1,
  idempotency_key text NOT NULL,
  result_summary text,
  verification_status verification_status NOT NULL DEFAULT 'pending',
  verification_details jsonb NOT NULL DEFAULT '{}'::jsonb,
  error_message text,
  started_at timestamptz,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT remediations_attempt_positive CHECK (attempt_number >= 1)
);

CREATE UNIQUE INDEX remediations_idempotency_uidx ON remediations (idempotency_key);
CREATE UNIQUE INDEX remediations_recommendation_attempt_uidx
  ON remediations (recommendation_id, attempt_number);
CREATE INDEX remediations_incident_idx ON remediations (incident_id, status);

CREATE TRIGGER remediations_set_updated_at
  BEFORE UPDATE ON remediations
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE remediations IS 'Owned by services/remediation. Never delete.';

CREATE TABLE approvals (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  remediation_id uuid NOT NULL REFERENCES remediations (id),
  incident_id uuid NOT NULL REFERENCES incidents (id),
  recommendation_id uuid NOT NULL REFERENCES recommendations (id),
  decision approval_decision NOT NULL DEFAULT 'pending',
  actor_user_id uuid REFERENCES users (id),
  comment text,
  created_at timestamptz NOT NULL DEFAULT now(),
  decided_at timestamptz,
  CONSTRAINT approvals_decided_shape CHECK (
    (decision = 'pending' AND decided_at IS NULL AND actor_user_id IS NULL)
    OR (decision <> 'pending' AND decided_at IS NOT NULL AND actor_user_id IS NOT NULL)
  )
);

CREATE UNIQUE INDEX approvals_remediation_uidx ON approvals (remediation_id);
CREATE INDEX approvals_incident_idx ON approvals (incident_id);

COMMENT ON TABLE approvals IS 'Human decision audit record. 1:1 with remediations.';
