package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (User, string, error) {
	var user User
	var hash string
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, display_name, role::text, status::text, password_hash
		FROM users
		WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.Status, &hash)
	if err != nil {
		return User{}, "", err
	}
	return user, hash, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, display_name, role::text, status::text
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.Status)
	return user, err
}

func (r *Repository) CreateSession(ctx context.Context, session Session) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO auth_sessions (id, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, session.ID, session.UserID, session.ExpiresAt)
	return err
}

func (r *Repository) GetSession(ctx context.Context, id uuid.UUID) (Session, error) {
	var s Session
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, expires_at, revoked_at
		FROM auth_sessions
		WHERE id = $1
	`, id).Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.RevokedAt)
	return s, err
}

func (r *Repository) RevokeSession(ctx context.Context, id uuid.UUID, at time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`, id, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
