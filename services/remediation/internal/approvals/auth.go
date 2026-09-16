package approvals

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/services/remediation/internal/database"
)

type Actor struct {
	ID   uuid.UUID
	Role string
}

func CanInspect(role string) bool {
	switch role {
	case "viewer", "responder", "approver", "admin":
		return true
	default:
		return false
	}
}

func CanApprove(actorRole, required string) bool {
	if actorRole != "approver" && actorRole != "admin" {
		return false
	}
	if required == "admin" {
		return actorRole == "admin"
	}
	return true
}

func ParseBearer(secret, header string) (uuid.UUID, error) {
	const p = "Bearer "
	if !strings.HasPrefix(header, p) {
		return uuid.Nil, fmt.Errorf("missing bearer token")
	}
	token := strings.TrimSpace(header[len(p):])
	if secret == "" {
		return uuid.Nil, fmt.Errorf("AUTH_TOKEN_SECRET is not configured")
	}
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		return uuid.Nil, fmt.Errorf("invalid token")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid token claims")
	}
	sub, _ := claims["sub"].(string)
	id, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid token subject")
	}
	return id, nil
}

func LoadActor(u database.User) (Actor, error) {
	if u.Status != "active" {
		return Actor{}, fmt.Errorf("user disabled")
	}
	return Actor{ID: u.ID, Role: u.Role}, nil
}
