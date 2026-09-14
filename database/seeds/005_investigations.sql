INSERT INTO investigations (
  id, incident_id, status, requested_by_user_id, model_name, model_version,
  root_cause_hypothesis, reasoning_summary, confidence, risk_level, tool_usage,
  error_message, idempotency_key, requested_at, started_at, completed_at
)
VALUES
  (
    '44444444-4444-4444-8444-444444444432',
    '33333333-3333-4333-8333-333333333332',
    'completed',
    '11111111-1111-4111-8111-111111111112',
    'sentinel-investigator',
    '2026.09.01',
    'Deploy 2.4.1 raised reservation batch size to 500, exceeding the 512Mi working set and triggering OOMKill.',
    'Memory saturation began minutes after 2.4.1. Logs show batch_size=500. Runbook inventory-oom-batch-cap matches. Health recovered after 2.4.2.',
    0.910,
    'high',
    '[{"tool":"get_recent_deployments","call_count":1,"status":"ok"},{"tool":"get_recent_logs","call_count":1,"status":"ok"},{"tool":"search_runbooks","call_count":1,"status":"ok"}]'::jsonb,
    NULL,
    'incident:33333333-3333-4333-8333-333333333332:investigate:1',
    '2026-09-07 14:45:00+00',
    '2026-09-07 14:45:02+00',
    '2026-09-07 14:48:00+00'
  ),
  (
    '44444444-4444-4444-8444-444444444441',
    '33333333-3333-4333-8333-333333333334',
    'completed',
    '11111111-1111-4111-8111-111111111113',
    'sentinel-investigator',
    '2026.09.01',
    'payments-api 1.18.0 added a synchronous inventory.reserve call with a 150ms timeout. Inventory p99 exceeds that budget, so capture fails and checkout degrades.',
    'Error rate and latency stepped at 04:17 after 1.18.0. Traces show inventory.reserve status=error at 150ms. Prior similar class: INC-2026-0002. Runbook payments-rollback-error-rate recommends rollback.',
    0.860,
    'high',
    '[{"tool":"get_recent_deployments","call_count":1,"status":"ok"},{"tool":"get_metrics","call_count":2,"status":"ok"},{"tool":"get_recent_logs","call_count":1,"status":"ok"},{"tool":"get_trace","call_count":1,"status":"ok"},{"tool":"search_runbooks","call_count":1,"status":"ok"},{"tool":"search_previous_incidents","call_count":1,"status":"ok"},{"tool":"get_deployment_diff","call_count":1,"status":"ok"}]'::jsonb,
    NULL,
    'incident:33333333-3333-4333-8333-333333333334:investigate:1',
    '2026-09-14 04:21:00+00',
    '2026-09-14 04:21:03+00',
    '2026-09-14 04:27:00+00'
  );

INSERT INTO evidence (
  id, investigation_id, tool_name, source_type, summary, artifact_uri, source_ref, metadata, captured_at
)
VALUES
  (
    '77777777-0000-4000-8000-000000000001',
    '44444444-4444-4444-8444-444444444441',
    'get_recent_deployments',
    'deployment',
    'payments-api 1.18.0 completed 04:17 UTC (e4b21aa), 1.17.4 is last good.',
    NULL,
    'aaaaaaa1-0000-4000-8000-000000000005',
    '{"version":"1.18.0"}'::jsonb,
    '2026-09-14 04:21:10+00'
  ),
  (
    '77777777-0000-4000-8000-000000000002',
    '44444444-4444-4444-8444-444444444441',
    'get_metrics',
    'metrics',
    'payments.capture.error_rate = 0.083; p99 = 1640ms.',
    NULL,
    'ccccccc1-0000-4000-8000-000000000006',
    '{"error_rate":0.083,"p99_ms":1640}'::jsonb,
    '2026-09-14 04:21:20+00'
  ),
  (
    '77777777-0000-4000-8000-000000000003',
    '44444444-4444-4444-8444-444444444441',
    'get_recent_logs',
    'logs',
    'Repeated inventory reservation timeout after 150ms.',
    's3://sentinel-local/evidence/inc-2026-0004/logs.json',
    'ccccccc1-0000-4000-8000-000000000008',
    '{"count":42}'::jsonb,
    '2026-09-14 04:22:00+00'
  ),
  (
    '77777777-0000-4000-8000-000000000004',
    '44444444-4444-4444-8444-444444444441',
    'get_trace',
    'trace',
    'Trace 4bf2c91a7e3d00aa span inventory.reserve duration_ms=150 status=error.',
    NULL,
    'ccccccc1-0000-4000-8000-000000000009',
    '{"trace_id":"4bf2c91a7e3d00aa"}'::jsonb,
    '2026-09-14 04:22:30+00'
  ),
  (
    '77777777-0000-4000-8000-000000000005',
    '44444444-4444-4444-8444-444444444441',
    'search_runbooks',
    'runbook',
    'Matched payments-rollback-error-rate.',
    NULL,
    'bbbbbbbb-0000-4000-8000-000000000001',
    '{"slug":"payments-rollback-error-rate"}'::jsonb,
    '2026-09-14 04:23:00+00'
  ),
  (
    '77777777-0000-4000-8000-000000000006',
    '44444444-4444-4444-8444-444444444441',
    'search_previous_incidents',
    'previous_incident',
    'INC-2026-0002 also correlated a deploy with resource saturation.',
    NULL,
    '33333333-3333-4333-8333-333333333332',
    '{"reference":"INC-2026-0002"}'::jsonb,
    '2026-09-14 04:23:20+00'
  ),
  (
    '77777777-0000-4000-8000-000000000007',
    '44444444-4444-4444-8444-444444444432',
    'get_recent_logs',
    'logs',
    'oom-killed: batch_size=500 reservations',
    NULL,
    'ccccccc1-0000-4000-8000-000000000004',
    '{}'::jsonb,
    '2026-09-07 14:46:00+00'
  );

INSERT INTO recommendations (
  id, incident_id, investigation_id, action_type, title, rationale, target_service_id,
  parameters, confidence, risk_level, required_approval_role, status, created_at, updated_at
)
VALUES
  (
    '55555555-5555-4555-8555-555555555551',
    '33333333-3333-4333-8333-333333333334',
    '44444444-4444-4444-8444-444444444441',
    'rollback_deployment',
    'Roll back payments-api to 1.17.4',
    '1.18.0 introduced a 150ms synchronous reserve that inventory cannot meet. Rolling back restores the async path. Risk is lost in-flight captures during the brief deploy.',
    '22222222-2222-4222-8222-222222222221',
    '{"to_version":"1.17.4","to_deployment_id":"aaaaaaa1-0000-4000-8000-000000000004","from_version":"1.18.0"}'::jsonb,
    0.880,
    'high',
    'approver',
    'proposed',
    '2026-09-14 04:27:00+00',
    '2026-09-14 04:27:00+00'
  ),
  (
    '55555555-5555-4555-8555-555555555552',
    '33333333-3333-4333-8333-333333333332',
    '44444444-4444-4444-8444-444444444432',
    'rollback_deployment',
    'Ship inventory-worker 2.4.2 with batch cap',
    'Forward fix preferred over memory-only bump; matches runbook.',
    '22222222-2222-4222-8222-222222222223',
    '{"to_version":"2.4.2"}'::jsonb,
    0.900,
    'medium',
    'approver',
    'accepted',
    '2026-09-07 14:48:00+00',
    '2026-09-07 15:10:00+00'
  );
