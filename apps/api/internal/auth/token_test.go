package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenRoundTrip(t *testing.T) {
	ts := NewTokenService("local-dev-token-secret", time.Hour)
	user := User{ID: uuid.MustParse("11111111-1111-4111-8111-111111111113"), Email: "sam.okonkwo@sentinel.dev", Role: "responder"}
	sessionID := uuid.New()
	token, _, err := ts.Issue(user, sessionID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	gotSession, gotUser, err := ts.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if gotSession != sessionID || gotUser != user.ID {
		t.Fatalf("got %s %s", gotSession, gotUser)
	}
}

func TestCanWriteIncidents(t *testing.T) {
	if CanWriteIncidents("viewer") {
		t.Fatal("viewer must not write")
	}
	if !CanWriteIncidents("responder") || !CanWriteIncidents("admin") {
		t.Fatal("responder/admin must write")
	}
}
