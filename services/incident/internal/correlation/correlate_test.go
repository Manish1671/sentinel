package correlation

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMatchSameServiceWithinWindow(t *testing.T) {
	svc := uuid.MustParse("22222222-2222-4222-8222-222222222221")
	now := time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC)
	alert := Alert{ID: uuid.New(), ServiceID: svc, Environment: "production", DetectorID: "high_error_rate", Severity: "high", StartedAt: now, Labels: map[string]string{}}
	c := Candidate{ID: uuid.New(), ServiceID: svc, Environment: "production", Status: "open", Severity: "high", DetectedAt: now.Add(-2 * time.Minute), LastActivityAt: now.Add(-2 * time.Minute), DetectorIDs: []string{"high_latency"}}
	ok, exp := Match(alert, c, 5*time.Minute)
	if !ok || !exp.SameService || !exp.WithinWindow || !exp.CompatibleCategory {
		t.Fatalf("ok=%v exp=%+v", ok, exp)
	}
}

func TestMatchRejectsDifferentService(t *testing.T) {
	now := time.Now().UTC()
	alert := Alert{ServiceID: uuid.New(), Environment: "production", DetectorID: "high_latency", StartedAt: now}
	c := Candidate{ServiceID: uuid.New(), Environment: "production", Status: "open", LastActivityAt: now, DetectorIDs: []string{"high_latency"}}
	ok, exp := Match(alert, c, time.Minute)
	if ok || exp.SameService {
		t.Fatalf("expected rejection %+v", exp)
	}
}

func TestMatchRejectsDifferentEnvironment(t *testing.T) {
	svc := uuid.New()
	now := time.Now().UTC()
	alert := Alert{ServiceID: svc, Environment: "production", DetectorID: "high_latency", StartedAt: now}
	c := Candidate{ServiceID: svc, Environment: "staging", Status: "open", LastActivityAt: now, DetectorIDs: []string{"high_latency"}}
	ok, _ := Match(alert, c, time.Minute)
	if ok {
		t.Fatal("staging should not merge with production")
	}
}

func TestMatchRejectsOutsideWindow(t *testing.T) {
	svc := uuid.New()
	now := time.Now().UTC()
	alert := Alert{ServiceID: svc, Environment: "production", DetectorID: "high_error_rate", StartedAt: now}
	c := Candidate{ServiceID: svc, Environment: "production", Status: "open", LastActivityAt: now.Add(-10 * time.Minute), DetectorIDs: []string{"high_latency"}}
	ok, exp := Match(alert, c, 5*time.Minute)
	if ok || exp.WithinWindow {
		t.Fatalf("expected outside window %+v", exp)
	}
}

func TestMatchRejectsClosedIncident(t *testing.T) {
	svc := uuid.New()
	now := time.Now().UTC()
	alert := Alert{ServiceID: svc, Environment: "production", DetectorID: "high_latency", StartedAt: now}
	c := Candidate{ServiceID: svc, Environment: "production", Status: "closed", LastActivityAt: now, DetectorIDs: []string{"high_latency"}}
	ok, exp := Match(alert, c, time.Minute)
	if ok || exp.CompatibleStatus {
		t.Fatal("closed incidents are not active")
	}
}

func TestMatchRejectsIncompatibleDetectorWithoutDeployment(t *testing.T) {
	svc := uuid.New()
	now := time.Now().UTC()
	alert := Alert{ServiceID: svc, Environment: "production", DetectorID: "tls_expiry", StartedAt: now, Labels: map[string]string{}}
	c := Candidate{ServiceID: svc, Environment: "production", Status: "open", LastActivityAt: now, DetectorIDs: []string{"high_latency"}}
	ok, exp := Match(alert, c, time.Minute)
	if ok || exp.CompatibleCategory {
		t.Fatalf("unrelated detector should not merge %+v", exp)
	}
}

func TestMatchDeploymentContext(t *testing.T) {
	svc := uuid.New()
	now := time.Now().UTC()
	dep := "dep-1"
	alert := Alert{ServiceID: svc, Environment: "production", DetectorID: "tls_expiry", StartedAt: now, Labels: map[string]string{"deployment_id": dep}}
	c := Candidate{ServiceID: svc, Environment: "production", Status: "open", LastActivityAt: now, DetectorIDs: []string{"high_latency"}, DeploymentIDs: []string{dep}}
	ok, exp := Match(alert, c, time.Minute)
	if !ok || !exp.DeploymentContext {
		t.Fatalf("shared deployment should match %+v", exp)
	}
}

func TestSelectPrefersNewestActivity(t *testing.T) {
	svc := uuid.New()
	now := time.Now().UTC()
	older := Candidate{ID: uuid.New(), ServiceID: svc, Environment: "production", Status: "open", LastActivityAt: now.Add(-2 * time.Minute), DetectorIDs: []string{"high_latency"}}
	newer := Candidate{ID: uuid.New(), ServiceID: svc, Environment: "production", Status: "open", LastActivityAt: now.Add(-30 * time.Second), DetectorIDs: []string{"high_error_rate"}}
	alert := Alert{ServiceID: svc, Environment: "production", DetectorID: "db_connection_saturation", StartedAt: now}
	got, _ := Select(alert, []Candidate{older, newer}, 5*time.Minute)
	if got == nil || got.ID != newer.ID {
		t.Fatalf("got %+v", got)
	}
}

func TestSelectCreatesNoMatch(t *testing.T) {
	alert := Alert{ServiceID: uuid.New(), Environment: "production", DetectorID: "high_latency", StartedAt: time.Now().UTC()}
	got, _ := Select(alert, nil, time.Minute)
	if got != nil {
		t.Fatal("expected new incident")
	}
}

func TestMaxSeverityNeverDecreases(t *testing.T) {
	if MaxSeverity("critical", "low") != "critical" {
		t.Fatal("critical must win")
	}
	if MaxSeverity("medium", "high") != "high" {
		t.Fatal("high must win")
	}
	if MaxSeverity("high", "low") != "high" {
		t.Fatal("must not decrease")
	}
}

func TestTitleDeterministic(t *testing.T) {
	if Title("payments-api", "production", []string{"high_latency"}) != "payments-api production performance incident" {
		t.Fatal(Title("payments-api", "production", []string{"high_latency"}))
	}
	if Title("payments-api", "production", []string{"high_latency", "high_error_rate"}) != "payments-api production degradation" {
		t.Fatal(Title("payments-api", "production", []string{"high_latency", "high_error_rate"}))
	}
}
