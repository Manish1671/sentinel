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
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, name := range files {
		body, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return err
		}
		sum := checksum(body)
		if prev, ok := applied[name]; ok {
			if prev != sum {
				return fmt.Errorf("migration %s was already applied with a different checksum", name)
			}
			continue
		}
		tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)`, name, sum); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func loadApplied(ctx context.Context, db *DB) (map[string]string, error) {
	rows, err := db.Pool.Query(ctx, `SELECT filename, checksum FROM schema_migrations`)
	if err != nil {
		return nil, err
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

func checksum(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func DefaultMigrationsPath() string {
	if p := firstExistingDir(os.Getenv("MIGRATIONS_PATH")); p != "" {
		return p
	}
	if p := findUp("database/migrations"); p != "" {
		return p
	}
	return filepath.Join("database", "migrations")
}

func firstExistingDir(p string) string {
	if strings.TrimSpace(p) == "" {
		return ""
	}
	st, err := os.Stat(p)
	if err != nil || !st.IsDir() {
		return ""
	}
	abs, err := filepath.Abs(p)
	if err == nil {
		return abs
	}
	return p
}

func findUp(rel string) string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 8; i++ {
		p := filepath.Join(wd, filepath.FromSlash(rel))
		if found := firstExistingDir(p); found != "" {
			return found
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return ""
}
