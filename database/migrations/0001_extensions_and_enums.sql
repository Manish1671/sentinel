-- Sentinel Phase 1: extensions, enums, shared trigger.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$;

CREATE TYPE user_role AS ENUM ('viewer', 'responder', 'approver', 'admin');
CREATE TYPE user_status AS ENUM ('active', 'disabled');

CREATE TYPE service_environment AS ENUM ('production', 'staging', 'development');
CREATE TYPE service_health_status AS ENUM ('healthy', 'degraded', 'unhealthy', 'unknown');

CREATE TYPE deployment_status AS ENUM ('in_progress', 'succeeded', 'failed', 'rolled_back');

CREATE TYPE telemetry_kind AS ENUM ('metric', 'log', 'trace', 'health');

CREATE TYPE severity_level AS ENUM ('critical', 'high', 'medium', 'low');
CREATE TYPE alert_status AS ENUM ('open', 'acknowledged', 'resolved', 'suppressed');

CREATE TYPE incident_status AS ENUM (
  'open',
  'investigating',
  'remediating',
  'verifying',
  'resolved',
  'closed'
);

CREATE TYPE incident_event_kind AS ENUM (
  'created',
  'status_changed',
  'alert_attached',
  'comment',
  'commander_changed',
  'investigation_requested',
  'investigation_completed',
  'recommendation_added',
  'approval_recorded',
  'remediation_requested',
  'remediation_completed',
  'remediation_failed'
);

CREATE TYPE investigation_status AS ENUM (
  'requested',
  'running',
  'completed',
  'failed',
  'cancelled'
);

CREATE TYPE evidence_source_type AS ENUM (
  'logs',
  'metrics',
  'trace',
  'deployment',
  'runbook',
  'previous_incident',
  'health'
);

CREATE TYPE recommendation_status AS ENUM (
  'proposed',
  'accepted',
  'rejected',
  'superseded',
  'expired'
);

CREATE TYPE risk_level AS ENUM ('low', 'medium', 'high');
CREATE TYPE approval_role AS ENUM ('approver', 'admin');

CREATE TYPE remediation_status AS ENUM (
  'pending_approval',
  'approved',
  'rejected',
  'running',
  'verifying',
  'succeeded',
  'failed',
  'cancelled'
);

CREATE TYPE approval_decision AS ENUM ('pending', 'approved', 'rejected');
CREATE TYPE verification_status AS ENUM ('pending', 'passed', 'failed', 'skipped');
CREATE TYPE runbook_status AS ENUM ('draft', 'published');
CREATE TYPE evaluation_status AS ENUM ('pending', 'running', 'completed', 'failed');
