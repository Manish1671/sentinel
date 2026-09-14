CREATE TABLE investigations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  incident_id uuid NOT NULL REFERENCES incidents (id),
  status investigation_status NOT NULL DEFAULT 'requested',
  requested_by_user_id uuid REFERENCES users (id),
  model_name text,
  model_version text,
  root_cause_hypothesis text,
  reasoning_summary text,
  confidence numeric(4, 3),
  risk_level risk_level,
  tool_usage jsonb NOT NULL DEFAULT '[]'::jsonb,
  error_message text,
  idempotency_key text NOT NULL,
  requested_at timestamptz NOT NULL DEFAULT now(),
  started_at timestamptz,
  completed_at timestamptz,
  CONSTRAINT investigations_confidence_range CHECK (
    confidence IS NULL OR (confidence >= 0 AND confidence <= 1)
  )
);

CREATE UNIQUE INDEX investigations_idempotency_uidx ON investigations (idempotency_key);
CREATE INDEX investigations_incident_idx ON investigations (incident_id, requested_at DESC);
CREATE INDEX investigations_status_idx ON investigations (status);

COMMENT ON TABLE investigations IS
  'Request owned by services/incident; results owned by services/ai.';

CREATE TABLE evidence (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  investigation_id uuid NOT NULL REFERENCES investigations (id),
  tool_name text NOT NULL,
  source_type evidence_source_type NOT NULL,
  summary text NOT NULL,
  artifact_uri text,
  source_ref uuid,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  captured_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX evidence_investigation_idx ON evidence (investigation_id);

COMMENT ON TABLE evidence IS 'Append-only investigation artifacts. Owned by services/ai.';

CREATE TABLE recommendations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  incident_id uuid NOT NULL REFERENCES incidents (id),
  investigation_id uuid REFERENCES investigations (id),
  action_type text NOT NULL,
  title text NOT NULL,
  rationale text NOT NULL,
  target_service_id uuid NOT NULL REFERENCES services (id),
  parameters jsonb NOT NULL DEFAULT '{}'::jsonb,
  confidence numeric(4, 3) NOT NULL,
  risk_level risk_level NOT NULL,
  required_approval_role approval_role NOT NULL DEFAULT 'approver',
  status recommendation_status NOT NULL DEFAULT 'proposed',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT recommendations_confidence_range CHECK (confidence >= 0 AND confidence <= 1),
  CONSTRAINT recommendations_action_type_known CHECK (
    action_type IN (
      'rollback_deployment',
      'restart_service',
      'scale_replicas',
      'disable_feature_flag',
      'run_runbook',
      'page_owner',
      'other'
    )
  )
);

CREATE INDEX recommendations_incident_idx ON recommendations (incident_id, status);
CREATE INDEX recommendations_investigation_idx ON recommendations (investigation_id);

CREATE TRIGGER recommendations_set_updated_at
  BEFORE UPDATE ON recommendations
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE recommendations IS 'Proposed actions. Authored by services/ai; status via apps/api.';
