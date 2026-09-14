INSERT INTO services (
  id, slug, name, environment, description, owner_user_id, health_status, created_at, updated_at
)
VALUES
  (
    '22222222-2222-4222-8222-222222222221',
    'payments-api',
    'Payments API',
    'production',
    'Authorization, capture, and refunds for checkout.',
    '11111111-1111-4111-8111-111111111113',
    'unhealthy',
    '2025-11-02 00:00:00+00',
    '2026-09-14 04:22:00+00'
  ),
  (
    '22222222-2222-4222-8222-222222222222',
    'checkout-web',
    'Checkout Web',
    'production',
    'Customer-facing checkout SPA and BFF.',
    '11111111-1111-4111-8111-111111111113',
    'degraded',
    '2025-11-02 00:00:00+00',
    '2026-09-14 04:22:00+00'
  ),
  (
    '22222222-2222-4222-8222-222222222223',
    'inventory-worker',
    'Inventory Worker',
    'production',
    'Stock reservation consumer.',
    '11111111-1111-4111-8111-111111111112',
    'healthy',
    '2025-08-14 00:00:00+00',
    '2026-09-07 16:40:00+00'
  ),
  (
    '22222222-2222-4222-8222-222222222224',
    'auth-service',
    'Auth Service',
    'production',
    'Session issuance and JWT verification.',
    '11111111-1111-4111-8111-111111111111',
    'healthy',
    '2025-06-01 00:00:00+00',
    '2026-08-12 19:15:00+00'
  ),
  (
    '22222222-2222-4222-8222-222222222225',
    'notifications-worker',
    'Notifications Worker',
    'production',
    'Email and push delivery from the outbox.',
    '11111111-1111-4111-8111-111111111112',
    'degraded',
    '2025-09-20 00:00:00+00',
    '2026-09-13 21:10:00+00'
  );

INSERT INTO deployments (
  id, service_id, version, git_sha, status, changelog, deployed_by_user_id,
  started_at, completed_at, metadata
)
VALUES
  (
    'aaaaaaa1-0000-4000-8000-000000000001',
    '22222222-2222-4222-8222-222222222224',
    '3.1.0',
    'b7e12c0',
    'succeeded',
    'Rotate TLS certificates 30 days early; extend OCSP cache.',
    '11111111-1111-4111-8111-111111111111',
    '2026-08-12 18:40:00+00',
    '2026-08-12 18:48:00+00',
    '{"ticket":"SEC-418"}'::jsonb
  ),
  (
    'aaaaaaa1-0000-4000-8000-000000000002',
    '22222222-2222-4222-8222-222222222223',
    '2.4.1',
    '9c44aa1',
    'succeeded',
    'Batch inventory reservations; raise worker memory to 512Mi.',
    '11111111-1111-4111-8111-111111111112',
    '2026-09-07 14:05:00+00',
    '2026-09-07 14:12:00+00',
    '{"ticket":"INV-882"}'::jsonb
  ),
  (
    'aaaaaaa1-0000-4000-8000-000000000003',
    '22222222-2222-4222-8222-222222222223',
    '2.4.2',
    'c1d90e4',
    'succeeded',
    'Cap batch size at 50 and add memory watermark; rollback of 2.4.1 behavior.',
    '11111111-1111-4111-8111-111111111112',
    '2026-09-07 16:05:00+00',
    '2026-09-07 16:11:00+00',
    '{"ticket":"INV-882","fixes":"2.4.1 OOM"}'::jsonb
  ),
  (
    'aaaaaaa1-0000-4000-8000-000000000004',
    '22222222-2222-4222-8222-222222222221',
    '1.17.4',
    '18af33d',
    'succeeded',
    'Stable payments capture path.',
    '11111111-1111-4111-8111-111111111113',
    '2026-09-13 09:00:00+00',
    '2026-09-13 09:08:00+00',
    '{}'::jsonb
  ),
  (
    'aaaaaaa1-0000-4000-8000-000000000005',
    '22222222-2222-4222-8222-222222222221',
    '1.18.0',
    'e4b21aa',
    'succeeded',
    'Synchronous inventory reservation before capture; 150ms timeout.',
    '11111111-1111-4111-8111-111111111113',
    '2026-09-14 04:12:00+00',
    '2026-09-14 04:17:00+00',
    '{"ticket":"PAY-1204"}'::jsonb
  ),
  (
    'aaaaaaa1-0000-4000-8000-000000000006',
    '22222222-2222-4222-8222-222222222222',
    '8.2.0',
    '55c0d12',
    'succeeded',
    'Show pending state while payments-api capture waits.',
    '11111111-1111-4111-8111-111111111113',
    '2026-09-14 03:50:00+00',
    '2026-09-14 03:58:00+00',
    '{}'::jsonb
  ),
  (
    'aaaaaaa1-0000-4000-8000-000000000007',
    '22222222-2222-4222-8222-222222222225',
    '1.9.3',
    '0aa19ce',
    'succeeded',
    'Increase outbox poll interval to 2s.',
    '11111111-1111-4111-8111-111111111112',
    '2026-09-13 18:00:00+00',
    '2026-09-13 18:06:00+00',
    '{"ticket":"NTF-77"}'::jsonb
  );

INSERT INTO runbooks (
  id, service_id, slug, title, failure_class, body, status, created_by_user_id, created_at, updated_at
)
VALUES
  (
    'bbbbbbbb-0000-4000-8000-000000000001',
    '22222222-2222-4222-8222-222222222221',
    'payments-rollback-error-rate',
    'Payments API error-rate rollback',
    'high_error_rate',
    $rb$1. Confirm error rate on `payments.capture.errors` > 2% for 5 minutes.
2. Compare current deployment to previous successful version.
3. If the regression started at deploy time, roll back payments-api to the last good version.
4. Verify capture success rate returns above 99% before closing.$rb$,
    'published',
    '11111111-1111-4111-8111-111111111112',
    '2026-04-02 10:00:00+00',
    '2026-04-02 10:00:00+00'
  ),
  (
    'bbbbbbbb-0000-4000-8000-000000000002',
    '22222222-2222-4222-8222-222222222223',
    'inventory-oom-batch-cap',
    'Inventory worker OOM',
    'oom_killed',
    $rb$1. Check pod restarts and memory watermark.
2. If a recent deploy increased batch size, cap batches at 50 and roll forward.
3. Do not raise memory limits without a batch cap.
4. Drain the reservation backlog before closing.$rb$,
    'published',
    '11111111-1111-4111-8111-111111111112',
    '2026-03-11 16:00:00+00',
    '2026-09-07 16:30:00+00'
  ),
  (
    'bbbbbbbb-0000-4000-8000-000000000003',
    '22222222-2222-4222-8222-222222222224',
    'auth-certificate-rotation',
    'Auth TLS certificate expiry',
    'tls_expiry',
    $rb$1. Confirm `notAfter` on the serving certificate.
2. Deploy the rotated secret (runbook SEC-418).
3. Reload auth-service; do not restart unrelated data stores.
4. Verify `/ready` and a successful login.$rb$,
    'published',
    '11111111-1111-4111-8111-111111111111',
    '2026-05-20 12:00:00+00',
    '2026-08-12 19:00:00+00'
  ),
  (
    'bbbbbbbb-0000-4000-8000-000000000004',
    NULL,
    'platform-consumer-lag',
    'Consumer lag playbook',
    'queue_lag',
    $rb$1. Identify the consumer group and partition lag.
2. Check downstream dependency errors before scaling.
3. Scale replicas only if CPU is saturated and error rate is flat.
4. Page the owning team if lag doubles in 15 minutes.$rb$,
    'published',
    '11111111-1111-4111-8111-111111111112',
    '2026-01-15 08:00:00+00',
    '2026-01-15 08:00:00+00'
  );
