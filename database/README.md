# database

PostgreSQL is the system of record.

## Layout

- `migrations/` — versioned schema, applied in lexicographic order
- `seeds/` — deterministic local data (see `seeds/README.md`)

Apply migrations before seeds. There is no application migrator in Phase 1.

See `docs/architecture/data.md` for relationships and `docs/architecture/domain.md` for the model.
