package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

type AppliedMigration struct {
	Filename string
	Checksum string
}

func Migrate(ctx context.Context, db *DB, migrationsDir string) error {
	if strings.TrimSpace(migrationsDir) == "" {
		return fmt.Errorf("migrations path is empty")
	}
	if _, err := os.Stat(migrationsDir); err != nil {
		return fmt.Errorf("migrations path %q: %w", migrationsDir, err)
	}

	if _, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename text PRIMARY KEY,
			checksum text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := loadApplied(ctx, db)
	if err != nil {
		return err
	}

	files, err := listSQLFiles(migrationsDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		body, err := os.ReadFile(file.path)
		if err != nil {
			return fmt.Errorf("read %s: %w", file.name, err)
		}
		sum := checksum(body)
		if prev, ok := applied[file.name]; ok {
			if prev != sum {
				return fmt.Errorf("migration %s was already applied with a different checksum", file.name)
			}
			continue
		}

		tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return fmt.Errorf("begin %s: %w", file.name, err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", file.name, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)
		`, file.name, sum); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record %s: %w", file.name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", file.name, err)
		}
	}
	return nil
}

func loadApplied(ctx context.Context, db *DB) (map[string]string, error) {
	rows, err := db.Pool.Query(ctx, `SELECT filename, checksum FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("list schema_migrations: %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var filename, sum string
		if err := rows.Scan(&filename, &sum); err != nil {
			return nil, err
		}
		out[filename] = sum
	}
	return out, rows.Err()
}

type migrationFile struct {
	name string
	path string
}

func listSQLFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []migrationFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		files = append(files, migrationFile{name: e.Name(), path: filepath.Join(dir, e.Name())})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return files, nil
}

func checksum(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func DefaultMigrationsPath() string {
	candidates := []string{
		os.Getenv("MIGRATIONS_PATH"),
		filepath.Join("database", "migrations"),
		filepath.Join("..", "..", "database", "migrations"),
		filepath.Join("..", "..", "..", "database", "migrations"),
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs
			}
			return c
		}
	}
	return filepath.Join("database", "migrations")
}
