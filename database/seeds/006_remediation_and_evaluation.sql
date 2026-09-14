INSERT INTO remediations (
  id, incident_id, recommendation_id, service_id, status, action_type, parameters,
  requested_by_user_id, attempt_number, idempotency_key, result_summary,
  verification_status, verification_details, error_message, started_at, completed_at,
  created_at, updated_at
)
VALUES
  (
    '66666666-6666-4666-8666-666666666661',
    '33333333-3333-4333-8333-333333333334',
    '55555555-5555-4555-8555-555555555551',
    '22222222-2222-4222-8222-222222222221',
    'pending_approval',
    'rollback_deployment',
    '{"to_version":"1.17.4","to_deployment_id":"aaaaaaa1-0000-4000-8000-000000000004"}'::jsonb,
    '11111111-1111-4111-8111-111111111113',
    1,
    'recommendation:55555555-5555-4555-8555-555555555551:1',
    NULL,
    'pending',
    '{}'::jsonb,
    NULL,
    NULL,
    NULL,
    '2026-09-14 04:27:10+00',
    '2026-09-14 04:27:10+00'
  ),
  (
    '66666666-6666-4666-8666-666666666662',
    '33333333-3333-4333-8333-333333333332',
    '55555555-5555-4555-8555-555555555552',
    '22222222-2222-4222-8222-222222222223',
    'succeeded',
    'rollback_deployment',
    '{"to_version":"2.4.2","to_deployment_id":"aaaaaaa1-0000-4000-8000-000000000003"}'::jsonb,
    '11111111-1111-4111-8111-111111111112',
    1,
    'recommendation:55555555-5555-4555-8555-555555555552:1',
    'Deployed inventory-worker 2.4.2. Memory watermark stable; lag drained.',
    'passed',
    '{"health":"healthy","restarts":0}'::jsonb,
    NULL,
    '2026-09-07 16:05:00+00',
    '2026-09-07 16:20:00+00',
    '2026-09-07 15:10:00+00',
    '2026-09-07 16:20:00+00'
  );

INSERT INTO approvals (
  id, remediation_id, incident_id, recommendation_id, decision, actor_user_id, comment, created_at, decided_at
)
VALUES
  (
    '88888888-0000-4000-8000-000000000001',
    '66666666-6666-4666-8666-666666666661',
    '33333333-3333-4333-8333-333333333334',
    '55555555-5555-4555-8555-555555555551',
    'pending',
    NULL,
    NULL,
    '2026-09-14 04:27:10+00',
    NULL
  ),
  (
    '88888888-0000-4000-8000-000000000002',
    '66666666-6666-4666-8666-666666666662',
    '33333333-3333-4333-8333-333333333332',
    '55555555-5555-4555-8555-555555555552',
    'approved',
    '11111111-1111-4111-8111-111111111112',
    'Forward-fix 2.4.2 approved; do not only bump memory.',
    '2026-09-07 15:10:00+00',
    '2026-09-07 15:12:00+00'
  );

INSERT INTO evaluation_runs (
  id, name, status, labeled_incident_id, model_name, model_version, metrics,
  error_message, started_at, completed_at, created_at
)
VALUES
  (
    '99999999-0000-4000-8000-000000000001',
    'replay-inc-2026-0002-investigator-2026.09.01',
    'completed',
    '33333333-3333-4333-8333-333333333332',
    'sentinel-investigator',
    '2026.09.01',
    '{"root_cause_match":1.0,"action_match":1.0,"confidence_calibration":0.91,"notes":"Labeled cause: 2.4.1 batch size OOM. Model recommended 2.4.2 batch cap."}'::jsonb,
    NULL,
    '2026-09-10 12:00:00+00',
    '2026-09-10 12:04:00+00',
    '2026-09-10 12:00:00+00'
  );
