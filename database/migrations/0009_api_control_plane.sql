-- API control-plane runtime tables (Phase 2).
-- Does not alter Phase 1 domain tables.

CREATE TABLE auth_sessions (
  id uuid PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES users (id),
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX auth_sessions_user_idx ON auth_sessions (user_id);
CREATE INDEX auth_sessions_expires_idx ON auth_sessions (expires_at);

COMMENT ON TABLE auth_sessions IS 'Signed-token sessions for apps/api. Logout sets revoked_at.';

CREATE TABLE http_idempotency_keys (
  scope text NOT NULL,
  key text NOT NULL,
  actor_user_id uuid NOT NULL REFERENCES users (id),
  request_hash text NOT NULL,
  resource_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (scope, key, actor_user_id),
  CONSTRAINT http_idempotency_keys_key_format CHECK (key ~ '^[A-Za-z0-9._:-]{8,128}$')
);

CREATE INDEX http_idempotency_keys_resource_idx ON http_idempotency_keys (resource_id);

COMMENT ON TABLE http_idempotency_keys IS 'PostgreSQL source of truth for HTTP idempotency.';
