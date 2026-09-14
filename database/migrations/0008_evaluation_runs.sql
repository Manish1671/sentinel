CREATE TABLE evaluation_runs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  status evaluation_status NOT NULL DEFAULT 'pending',
  labeled_incident_id uuid REFERENCES incidents (id),
  model_name text,
  model_version text,
  metrics jsonb NOT NULL DEFAULT '{}'::jsonb,
  error_message text,
  started_at timestamptz,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX evaluation_runs_incident_idx ON evaluation_runs (labeled_incident_id);
CREATE INDEX evaluation_runs_status_idx ON evaluation_runs (status);

COMMENT ON TABLE evaluation_runs IS
  'Owned by services/ai. Must not mutate production incident state.';
