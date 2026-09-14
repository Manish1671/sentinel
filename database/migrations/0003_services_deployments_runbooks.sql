CREATE TABLE services (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug text NOT NULL,
  name text NOT NULL,
  environment service_environment NOT NULL,
  description text NOT NULL DEFAULT '',
  owner_user_id uuid REFERENCES users (id),
  health_status service_health_status NOT NULL DEFAULT 'unknown',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT services_slug_format CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$')
);

CREATE UNIQUE INDEX services_slug_environment_uidx ON services (slug, environment);
CREATE INDEX services_health_idx ON services (health_status);

CREATE TRIGGER services_set_updated_at
  BEFORE UPDATE ON services
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE services IS 'Owned by apps/api catalog.';

CREATE TABLE deployments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  service_id uuid NOT NULL REFERENCES services (id),
  version text NOT NULL,
  git_sha text,
  status deployment_status NOT NULL DEFAULT 'in_progress',
  changelog text NOT NULL DEFAULT '',
  deployed_by_user_id uuid REFERENCES users (id),
  started_at timestamptz NOT NULL,
  completed_at timestamptz,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT deployments_completed_after_start CHECK (
    completed_at IS NULL OR completed_at >= started_at
  )
);

CREATE INDEX deployments_service_started_idx ON deployments (service_id, started_at DESC);

COMMENT ON TABLE deployments IS 'Owned by apps/api catalog; ingestion may emit deployment.created.';

CREATE TABLE runbooks (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  service_id uuid REFERENCES services (id),
  slug text NOT NULL,
  title text NOT NULL,
  failure_class text NOT NULL,
  body text NOT NULL,
  status runbook_status NOT NULL DEFAULT 'draft',
  created_by_user_id uuid REFERENCES users (id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT runbooks_slug_format CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$')
);

CREATE UNIQUE INDEX runbooks_slug_uidx ON runbooks (slug);
CREATE INDEX runbooks_service_idx ON runbooks (service_id);
CREATE INDEX runbooks_failure_class_idx ON runbooks (failure_class);

CREATE TRIGGER runbooks_set_updated_at
  BEFORE UPDATE ON runbooks
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE runbooks IS 'Human-owned operational guidance. Owned by apps/api.';
