# database

PostgreSQL is the system of record.

## Layout

- `migrations/` — versioned schema, applied in lexicographic order
- `seeds/` — deterministic local data (see `seeds/README.md`)

Apply migrations before seeds. Detection applies `0010_detection_processed_events.sql` on startup (`MIGRATIONS_PATH`).

See `docs/architecture/data.md` for relationships and `docs/architecture/domain.md` for the model.
