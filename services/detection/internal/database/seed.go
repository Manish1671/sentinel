package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func SeedIfEmpty(ctx context.Context, db *DB, seedsDir string) error {
	if strings.TrimSpace(seedsDir) == "" {
		return nil
	}
	var n int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM services`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	entries, err := os.ReadDir(seedsDir)
	if err != nil {
		return fmt.Errorf("seeds path %q: %w", seedsDir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, name := range files {
		body, err := os.ReadFile(filepath.Join(seedsDir, name))
		if err != nil {
			return err
		}
		if _, err := db.Pool.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("seed %s: %w", name, err)
		}
	}
	return nil
}

func DefaultSeedsPath() string {
	candidates := []string{
		os.Getenv("SEEDS_PATH"),
		filepath.Join("database", "seeds"),
		filepath.Join("..", "..", "database", "seeds"),
		filepath.Join("..", "..", "..", "database", "seeds"),
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
	return ""
}
