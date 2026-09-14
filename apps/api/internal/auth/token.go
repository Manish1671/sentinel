package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenService struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenService(secret string, ttl time.Duration) *TokenService {
	return &TokenService{secret: []byte(secret), ttl: ttl}
}

type claims struct {
	Role  string `json:"role"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (t *TokenService) Issue(user User, sessionID uuid.UUID, now time.Time) (string, time.Time, error) {
	exp := now.Add(t.ttl)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role:  user.Role,
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ID:        sessionID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	})
	signed, err := token.SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

func (t *TokenService) Parse(tokenString string) (sessionID, userID uuid.UUID, err error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return t.secret, nil
	})
	if err != nil || !parsed.Valid {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid token")
	}
	c, ok := parsed.Claims.(*claims)
	if !ok {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid token claims")
	}
	sessionID, err = uuid.Parse(c.ID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid token id")
	}
	userID, err = uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid token subject")
	}
	return sessionID, userID, nil
}
