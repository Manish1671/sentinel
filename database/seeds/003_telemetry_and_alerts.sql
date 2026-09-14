-- Retained samples used by seeds, dashboards, and future AI tools.

INSERT INTO telemetry_events (
  id, service_id, kind, source, occurred_at, ingested_at, correlation_id, payload
)
VALUES
  -- Auth certificate (historical)
  (
    'ccccccc1-0000-4000-8000-000000000001',
    '22222222-2222-4222-8222-222222222224',
    'metric',
    'prometheus',
    '2026-08-12 18:05:00+00',
    '2026-08-12 18:05:02+00',
    'dddddddd-0000-4000-8000-0000000000a1',
    '{"name":"tls.cert.days_remaining","value":0.4,"unit":"days","labels":{"service":"auth-service"}}'::jsonb
  ),
  (
    'ccccccc1-0000-4000-8000-000000000002',
    '22222222-2222-4222-8222-222222222224',
    'log',
    'vector',
    '2026-08-12 18:06:11+00',
    '2026-08-12 18:06:12+00',
    'dddddddd-0000-4000-8000-0000000000a1',
    '{"severity":"error","message":"tls: certificate is expired or expiring","trace_id":null}'::jsonb
  ),
  -- Inventory OOM (historical)
  (
    'ccccccc1-0000-4000-8000-000000000003',
    '22222222-2222-4222-8222-222222222223',
    'metric',
    'prometheus',
    '2026-09-07 14:40:00+00',
    '2026-09-07 14:40:02+00',
    'dddddddd-0000-4000-8000-0000000000b1',
    '{"name":"container.memory.working_set","value":536870912,"unit":"bytes","labels":{"pod":"inventory-worker-7f9"}}'::jsonb
  ),
  (
    'ccccccc1-0000-4000-8000-000000000004',
    '22222222-2222-4222-8222-222222222223',
    'log',
    'vector',
    '2026-09-07 14:41:03+00',
    '2026-09-07 14:41:04+00',
    'dddddddd-0000-4000-8000-0000000000b1',
    '{"severity":"fatal","message":"oom-killed: batch_size=500 reservations"}'::jsonb
  ),
  -- Notifications lag (open)
  (
    'ccccccc1-0000-4000-8000-000000000005',
    '22222222-2222-4222-8222-222222222225',
    'metric',
    'prometheus',
    '2026-09-13 21:05:00+00',
    '2026-09-13 21:05:02+00',
    'dddddddd-0000-4000-8000-0000000000c1',
    '{"name":"kafka.consumer.lag","value":18420,"unit":"messages","labels":{"group":"notifications-worker"}}'::jsonb
  ),
  -- Payments active incident
  (
    'ccccccc1-0000-4000-8000-000000000006',
    '22222222-2222-4222-8222-222222222221',
    'metric',
    'prometheus',
    '2026-09-14 04:18:00+00',
    '2026-09-14 04:18:01+00',
    'dddddddd-0000-4000-8000-0000000000d1',
    '{"name":"payments.capture.error_rate","value":0.083,"unit":"ratio","labels":{"route":"POST /capture"}}'::jsonb
  ),
  (
    'ccccccc1-0000-4000-8000-000000000007',
    '22222222-2222-4222-8222-222222222221',
    'metric',
    'prometheus',
    '2026-09-14 04:18:00+00',
    '2026-09-14 04:18:01+00',
    'dddddddd-0000-4000-8000-0000000000d1',
    '{"name":"payments.capture.latency_p99","value":1640,"unit":"ms","labels":{"route":"POST /capture"}}'::jsonb
  ),
  (
    'ccccccc1-0000-4000-8000-000000000008',
    '22222222-2222-4222-8222-222222222221',
    'log',
    'vector',
    '2026-09-14 04:18:22+00',
    '2026-09-14 04:18:23+00',
    'dddddddd-0000-4000-8000-0000000000d1',
    '{"severity":"error","message":"inventory reservation timeout after 150ms","trace_id":"4bf2c91a7e3d00aa","span_id":"91aa00e3"}'::jsonb
  ),
  (
    'ccccccc1-0000-4000-8000-000000000009',
    '22222222-2222-4222-8222-222222222221',
    'trace',
    'otel-collector',
    '2026-09-14 04:18:22+00',
    '2026-09-14 04:18:24+00',
    'dddddddd-0000-4000-8000-0000000000d1',
    '{"trace_id":"4bf2c91a7e3d00aa","span_id":"91aa00e3","parent_span_id":"10cfe812","name":"inventory.reserve","duration_ms":150,"status":"error"}'::jsonb
  ),
  (
    'ccccccc1-0000-4000-8000-00000000000a',
    '22222222-2222-4222-8222-222222222222',
    'log',
    'vector',
    '2026-09-14 04:19:01+00',
    '2026-09-14 04:19:02+00',
    'dddddddd-0000-4000-8000-0000000000d1',
    '{"severity":"warn","message":"checkout capture failed; showing pending retry"}'::jsonb
  );

INSERT INTO alerts (
  id, service_id, detector_id, severity, status, title, summary, fingerprint,
  labels, triggering_event_id, started_at, ended_at, created_at, updated_at
)
VALUES
  (
    'eeeeeeee-0000-4000-8000-000000000001',
    '22222222-2222-4222-8222-222222222224',
    'tls.days_remaining.low',
    'high',
    'resolved',
    'Auth TLS certificate near expiry',
    'Serving certificate had less than 12 hours remaining.',
    'auth-service:tls.days_remaining.low',
    '{"detector":"threshold"}'::jsonb,
    'ccccccc1-0000-4000-8000-000000000001',
    '2026-08-12 18:06:00+00',
    '2026-08-12 19:10:00+00',
    '2026-08-12 18:06:00+00',
    '2026-08-12 19:10:00+00'
  ),
  (
    'eeeeeeee-0000-4000-8000-000000000002',
    '22222222-2222-4222-8222-222222222223',
    'kube.oom_killed',
    'critical',
    'resolved',
    'Inventory worker OOM killed',
    'inventory-worker-7f9 restarted after exceeding 512Mi on batch_size=500.',
    'inventory-worker:kube.oom_killed',
    '{"pod":"inventory-worker-7f9"}'::jsonb,
    'ccccccc1-0000-4000-8000-000000000004',
    '2026-09-07 14:41:00+00',
    '2026-09-07 16:20:00+00',
    '2026-09-07 14:41:00+00',
    '2026-09-07 16:20:00+00'
  ),
  (
    'eeeeeeee-0000-4000-8000-000000000003',
    '22222222-2222-4222-8222-222222222225',
    'kafka.consumer.lag.high',
    'medium',
    'open',
    'Notifications consumer lag',
    'Group notifications-worker lag crossed 10k messages.',
    'notifications-worker:kafka.consumer.lag.high',
    '{"group":"notifications-worker"}'::jsonb,
    'ccccccc1-0000-4000-8000-000000000005',
    '2026-09-13 21:06:00+00',
    NULL,
    '2026-09-13 21:06:00+00',
    '2026-09-13 21:06:00+00'
  ),
  (
    'eeeeeeee-0000-4000-8000-000000000004',
    '22222222-2222-4222-8222-222222222221',
    'payments.capture.error_rate.high',
    'critical',
    'open',
    'Payments capture error rate',
    'Capture error ratio 8.3% after payments-api 1.18.0.',
    'payments-api:payments.capture.error_rate.high',
    '{"route":"POST /capture"}'::jsonb,
    'ccccccc1-0000-4000-8000-000000000006',
    '2026-09-14 04:19:00+00',
    NULL,
    '2026-09-14 04:19:00+00',
    '2026-09-14 04:19:00+00'
  ),
  (
    'eeeeeeee-0000-4000-8000-000000000005',
    '22222222-2222-4222-8222-222222222221',
    'payments.capture.latency.p99',
    'high',
    'open',
    'Payments capture p99 latency',
    'p99 capture latency 1640ms versus 180ms baseline.',
    'payments-api:payments.capture.latency.p99',
    '{"route":"POST /capture"}'::jsonb,
    'ccccccc1-0000-4000-8000-000000000007',
    '2026-09-14 04:19:30+00',
    NULL,
    '2026-09-14 04:19:30+00',
    '2026-09-14 04:19:30+00'
  );
