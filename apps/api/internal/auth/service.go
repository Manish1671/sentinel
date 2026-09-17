package auth

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/sentinel-dev/sentinel/apps/api/internal/apierr"
)

type Service struct {
	repo   *Repository
	tokens *TokenService
	now    func() time.Time
}

func NewService(repo *Repository, tokens *TokenService) *Service {
	return &Service{repo: repo, tokens: tokens, now: time.Now}
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      User
}

func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var fields []map[string]string
	if email == "" {
		fields = append(fields, apierr.Field("email", "required", "email is required"))
	}
	if password == "" {
		fields = append(fields, apierr.Field("password", "required", "password is required"))
	} else if utf8.RuneCountInString(password) > 256 {
		fields = append(fields, apierr.Field("password", "max_length", "password must be at most 256 characters"))
	}
	if len(fields) > 0 {
		return LoginResult{}, apierr.Validation(fields)
	}

	user, hash, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if IsNotFound(err) {
			return LoginResult{}, apierr.InvalidCredentials()
		}
		return LoginResult{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return LoginResult{}, apierr.InvalidCredentials()
	}
	if user.Status != "active" {
		return LoginResult{}, apierr.UserDisabled()
	}

	sessionID := uuid.New()
	now := s.now().UTC()
	token, exp, err := s.tokens.Issue(user, sessionID, now)
	if err != nil {
		return LoginResult{}, err
	}
	if err := s.repo.CreateSession(ctx, Session{ID: sessionID, UserID: user.ID, ExpiresAt: exp}); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, ExpiresAt: exp, User: user}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (User, uuid.UUID, error) {
	if token == "" {
		return User{}, uuid.Nil, apierr.Unauthenticated()
	}
	sessionID, userID, err := s.tokens.Parse(token)
	if err != nil {
		return User{}, uuid.Nil, apierr.Unauthenticated()
	}
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return User{}, uuid.Nil, apierr.Unauthenticated()
	}
	now := s.now().UTC()
	if session.UserID != userID || session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return User{}, uuid.Nil, apierr.Unauthenticated()
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return User{}, uuid.Nil, apierr.Unauthenticated()
	}
	if user.Status != "active" {
		return User{}, uuid.Nil, apierr.UserDisabled()
	}
	return user, sessionID, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	_, sessionID, err := s.Authenticate(ctx, token)
	if err != nil {
		return err
	}
	if err := s.repo.RevokeSession(ctx, sessionID, s.now().UTC()); err != nil && !IsNotFound(err) {
		return err
	}
	return nil
}
