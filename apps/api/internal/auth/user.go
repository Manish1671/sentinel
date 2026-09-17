package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
	Role        string
	Status      string
}

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type contextKey struct{}

func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, contextKey{}, user)
}

func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(contextKey{}).(User)
	return user, ok
}

func CanWriteIncidents(role string) bool {
	switch role {
	case "responder", "approver", "admin":
		return true
	default:
		return false
	}
}

func CanApproveRemediations(role string) bool {
	switch role {
	case "approver", "admin":
		return true
	default:
		return false
	}
}

func MeetsApprovalRole(actorRole, required string) bool {
	if required == "admin" {
		return actorRole == "admin"
	}
	return CanApproveRemediations(actorRole)
}
