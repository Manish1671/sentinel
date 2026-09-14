-- Deterministic local development identities.
-- Local password for every seed user: sentinel-dev (development only).

INSERT INTO users (id, email, display_name, role, status, password_hash, created_at, updated_at)
VALUES
  (
    '11111111-1111-4111-8111-111111111111',
    'maya.chen@sentinel.dev',
    'Maya Chen',
    'admin',
    'active',
    '$2a$10$TTzzFZ7B32DjK3TCoaQJge9mHFwk8Q/cWXtncY1I6q2cpussNQNaK',
    '2026-01-06 15:00:00+00',
    '2026-01-06 15:00:00+00'
  ),
  (
    '11111111-1111-4111-8111-111111111112',
    'jordan.hale@sentinel.dev',
    'Jordan Hale',
    'approver',
    'active',
    '$2a$10$TTzzFZ7B32DjK3TCoaQJge9mHFwk8Q/cWXtncY1I6q2cpussNQNaK',
    '2026-01-06 15:00:00+00',
    '2026-01-06 15:00:00+00'
  ),
  (
    '11111111-1111-4111-8111-111111111113',
    'sam.okonkwo@sentinel.dev',
    'Sam Okonkwo',
    'responder',
    'active',
    '$2a$10$TTzzFZ7B32DjK3TCoaQJge9mHFwk8Q/cWXtncY1I6q2cpussNQNaK',
    '2026-02-02 09:30:00+00',
    '2026-02-02 09:30:00+00'
  ),
  (
    '11111111-1111-4111-8111-111111111114',
    'riley.park@sentinel.dev',
    'Riley Park',
    'viewer',
    'active',
    '$2a$10$TTzzFZ7B32DjK3TCoaQJge9mHFwk8Q/cWXtncY1I6q2cpussNQNaK',
    '2026-03-18 11:00:00+00',
    '2026-03-18 11:00:00+00'
  );
