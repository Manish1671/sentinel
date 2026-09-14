# packages/config

Shared configuration conventions.

## Purpose

Document and later share:

- environment variable names
- local vs production config overlays
- service port and URL defaults
- feature flags (when needed)

Runtime secrets are never stored here. See `.env.example` and `docs/architecture/security.md`.

## Status

Placeholder. Services will load their own config in implementation phases.
