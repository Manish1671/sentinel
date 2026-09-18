package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/packages/telemetry"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/approvals"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/config"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/database"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/events"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/executor"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/kafka"
)

func TestMain(m *testing.M) {
	shutdown, err := telemetry.Init(context.Background(), telemetry.Config{ServiceName: "sentinel-remediation", Environment: "test"})
	if err != nil {
		panic(err)
	}
	code := m.Run()
	_ = shutdown(context.Background())
	os.Exit(code)
}

type memPub struct {
	mu   sync.Mutex
	msgs []kafka.Message
}

func (m *memPub) Publish(_ context.Context, msg kafka.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return nil
}

func testEnv(t *testing.T) (*Processor, *database.Store, *database.DB, context.Context) {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	db, err := database.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("postgres: %v", err)
	}
	t.Cleanup(db.Close)
	if err := database.Migrate(ctx, db, database.DefaultMigrationsPath()); err != nil {
		t.Skip(err)
	}
	_ = database.SeedIfEmpty(ctx, db, database.DefaultSeedsPath())
	store := database.NewStore(db)
	cfg := config.Config{Environment: "test", LogLevel: "error", KafkaBrokers: []string{"localhost:9092"}}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	sim := executor.NewSimulator(db)
	pub := &memPub{}
	proc := NewProcessor(cfg, log, store, sim, pub)
	return proc, store, db, ctx
}

func insertHarness(t *testing.T, ctx context.Context, db *database.DB) (incidentID, recID, serviceID uuid.UUID) {
	t.Helper()
	serviceID = uuid.New()
	slug := "t" + strings.ReplaceAll(serviceID.String(), "-", "")[:12]
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO services (id, slug, name, environment, health_status)
		VALUES ($1,$2,$2,'production','unhealthy')
	`, serviceID, slug)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO deployments (id, service_id, version, status, started_at, completed_at)
		VALUES (gen_random_uuid(), $1, '1.17.4', 'succeeded', now() - interval '2 days', now() - interval '2 days'),
		       (gen_random_uuid(), $1, '1.18.0', 'succeeded', now() - interval '1 hour', now() - interval '1 hour')
	`, serviceID)
	if err != nil {
		t.Fatal(err)
	}
	incidentID = uuid.New()
	n := int(incidentID[14])<<8 | int(incidentID[15])
	ref := fmt.Sprintf("INC-2099-%04d", n%10000)
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO incidents (id, reference, service_id, title, severity, status, detected_at, version)
		VALUES ($1,$2,$3,'test','high','investigating', now(), 1)
	`, incidentID, ref, serviceID)
	if err != nil {
		t.Fatal(err)
	}
	recID = uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO recommendations (
			id, incident_id, action_type, title, rationale, target_service_id, parameters,
			confidence, risk_level, required_approval_role, status
		) VALUES (
			$1,$2,'rollback_deployment','rollback','because',$3,
			'{"to_version":"1.17.4"}'::jsonb, 0.8, 'high', 'approver', 'proposed'
		)
	`, recID, incidentID, serviceID)
	if err != nil {
		t.Fatal(err)
	}
	return incidentID, recID, serviceID
}

func TestHappyPathAndIdempotency(t *testing.T) {
	proc, store, db, ctx := testEnv(t)
	incidentID, recID, _ := insertHarness(t, ctx, db)
	rec, err := store.GetRecommendation(ctx, recID)
	if err != nil {
		t.Fatal(err)
	}
	row, err := proc.CreateFromRecommendation(ctx, rec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if row.Status != "pending_approval" {
		t.Fatalf("status %s", row.Status)
	}
	again, err := proc.CreateFromRecommendation(ctx, rec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != row.ID {
		t.Fatal("duplicate create must reuse remediation")
	}
	approver := approvals.Actor{ID: uuid.MustParse("11111111-1111-4111-8111-111111111112"), Role: "approver"}
	viewer := approvals.Actor{ID: uuid.MustParse("11111111-1111-4111-8111-111111111114"), Role: "viewer"}
	viewerKey := "idem-viewer-" + uuid.NewString()[:8] + "-xxxxxx"
	approveKey := "idem-approve-" + uuid.NewString()[:8] + "-xx"
	rejectKey := "idem-reject-" + uuid.NewString()[:8] + "-xxx"
	if _, err := proc.Approve(ctx, row.ID, viewer, "", viewerKey, "h1"); !IsForbidden(err) {
		t.Fatalf("viewer must be forbidden: %v", err)
	}
	approved, err := proc.Approve(ctx, row.ID, approver, "ok", approveKey, "hash-a")
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != "approved" {
		t.Fatalf("status %s", approved.Status)
	}
	dup, err := proc.Approve(ctx, row.ID, approver, "ok", approveKey, "hash-a")
	if err != nil || dup.Status != "approved" {
		t.Fatalf("idempotent approve: %v %s", err, dup.Status)
	}
	if _, err := proc.Reject(ctx, row.ID, approver, "no", rejectKey, "hash-r"); !IsAlreadyDecided(err) {
		t.Fatalf("reject after approve: %v", err)
	}

	env := requestedEnvelope(approved.ID, incidentID, recID, approved.ServiceID)
	raw, _ := json.Marshal(env)
	if _, err := proc.Handle(ctx, raw, events.TopicRequested, 0, 1); err != nil {
		t.Fatal(err)
	}
	done, err := store.GetRemediation(ctx, approved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != "succeeded" {
		msg := ""
		if done.ErrorMessage != nil {
			msg = *done.ErrorMessage
		}
		t.Fatalf("expected succeeded got %s %s", done.Status, msg)
	}
	inc, _ := store.GetIncident(ctx, incidentID)
	if inc.Status != "resolved" {
		t.Fatalf("incident %s", inc.Status)
	}
	if _, err := proc.Handle(ctx, raw, events.TopicRequested, 0, 2); err != nil {
		t.Fatal(err)
	}
	againDone, _ := store.GetRemediation(ctx, approved.ID)
	if againDone.Status != "succeeded" {
		t.Fatal("replay must not change success")
	}
	st, err := proc.exec.State(ctx, approved.ServiceID)
	if err != nil {
		t.Fatal(err)
	}
	if st.CurrentVersion != "1.17.4" {
		t.Fatalf("version %s", st.CurrentVersion)
	}
}

func TestVerificationFailureKeepsIncidentActive(t *testing.T) {
	t.Setenv("SENTINEL_FAULT_INJECTION", "true")
	t.Setenv("REMEDIATION_FAULT_VERIFY_FAIL", "true")
	proc, store, db, ctx := testEnv(t)
	incidentID, recID, _ := insertHarness(t, ctx, db)
	rec, _ := store.GetRecommendation(ctx, recID)
	row, err := proc.CreateFromRecommendation(ctx, rec, nil)
	if err != nil {
		t.Fatal(err)
	}
	approver := approvals.Actor{ID: uuid.MustParse("11111111-1111-4111-8111-111111111112"), Role: "approver"}
	if _, err := proc.Approve(ctx, row.ID, approver, "", "idem-verify-"+uuid.NewString()[:8]+"-xx", "hv"); err != nil {
		t.Fatal(err)
	}
	env := requestedEnvelope(row.ID, incidentID, recID, row.ServiceID)
	raw, _ := json.Marshal(env)
	if _, err := proc.Handle(ctx, raw, events.TopicRequested, 0, 1); err != nil {
		t.Fatal(err)
	}
	failed, _ := store.GetRemediation(ctx, row.ID)
	if failed.Status != "failed" {
		t.Fatalf("status %s", failed.Status)
	}
	if failed.VerificationStatus != "failed" {
		t.Fatalf("verification %s", failed.VerificationStatus)
	}
	inc, _ := store.GetIncident(ctx, incidentID)
	if inc.Status == "resolved" || inc.Status == "closed" {
		t.Fatalf("incident must stay active, got %s", inc.Status)
	}
}

func TestExecutorFailureLeavesIncidentActive(t *testing.T) {
	proc, store, db, ctx := testEnv(t)
	incidentID, recID, _ := insertHarness(t, ctx, db)
	failer := &executor.Failer{Inner: executor.NewSimulator(db), Fail: true, Msg: "boom"}
	proc.exec = failer
	rec, _ := store.GetRecommendation(ctx, recID)
	row, err := proc.CreateFromRecommendation(ctx, rec, nil)
	if err != nil {
		t.Fatal(err)
	}
	approver := approvals.Actor{ID: uuid.MustParse("11111111-1111-4111-8111-111111111112"), Role: "approver"}
	if _, err := proc.Approve(ctx, row.ID, approver, "", "idem-fail-"+uuid.NewString()[:8]+"-xxxx", "hf"); err != nil {
		t.Fatal(err)
	}
	env := requestedEnvelope(row.ID, incidentID, recID, row.ServiceID)
	raw, _ := json.Marshal(env)
	if _, err := proc.Handle(ctx, raw, events.TopicRequested, 0, 1); err != nil {
		t.Fatal(err)
	}
	failed, _ := store.GetRemediation(ctx, row.ID)
	if failed.Status != "failed" {
		t.Fatalf("status %s", failed.Status)
	}
	inc, _ := store.GetIncident(ctx, incidentID)
	if inc.Status == "resolved" || inc.Status == "closed" {
		t.Fatalf("incident must stay active, got %s", inc.Status)
	}
}

func TestInvestigationCompletedBridge(t *testing.T) {
	proc, _, db, ctx := testEnv(t)
	incidentID, recID, _ := insertHarness(t, ctx, db)
	invID := uuid.New()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO investigations (id, incident_id, status, idempotency_key, requested_at)
		VALUES ($1,$2,'completed',$3, now())
	`, invID, incidentID, "test-inv:"+invID.String())
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Pool.Exec(ctx, `UPDATE recommendations SET investigation_id = $1 WHERE id = $2`, invID, recID)
	if err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{
		"investigation_id": invID.String(),
		"incident_id":      incidentID.String(),
		"status":           "completed",
		"result":           map[string]any{},
	}
	env := events.Build(events.TypeInvCompleted, uuid.New(), incidentID, nil, time.Now().UTC(), payload)
	raw, _ := json.Marshal(env)
	if _, err := proc.Handle(ctx, raw, events.TopicInvestigations, 0, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := proc.Handle(ctx, raw, events.TopicInvestigations, 0, 2); err != nil {
		t.Fatal(err)
	}
	items, err := proc.store.ListByIncident(ctx, incidentID)
	if err != nil || len(items) != 1 {
		t.Fatalf("expected 1 remediation, got %d %v", len(items), err)
	}
	if items[0].Status != "pending_approval" {
		t.Fatalf("status %s", items[0].Status)
	}
}

func requestedEnvelope(remID, incidentID, recID, serviceID uuid.UUID) events.Envelope {
	return events.Build(events.TypeRequested, events.RequestedEventID(remID), incidentID, nil, time.Now().UTC(), map[string]any{
		"remediation_id":    remID.String(),
		"incident_id":       incidentID.String(),
		"recommendation_id": recID.String(),
		"service_id":        serviceID.String(),
		"action_type":       "rollback_deployment",
		"parameters":        map[string]any{"to_version": "1.17.4"},
		"approval_id":       uuid.New().String(),
	})
}

func TestRemediationMetrics(t *testing.T) {
	ctx := context.Background()
	telemetry.Count(ctx, telemetry.RemRequested, "action_type", "rollback_deployment", "status", "pending_approval")
	telemetry.Count(ctx, telemetry.RemApproved, "action_type", "rollback_deployment", "status", "approved")
	telemetry.Count(ctx, telemetry.RemStarted, "action_type", "rollback_deployment", "status", "running")
	telemetry.Count(ctx, telemetry.RemSucceeded, "action_type", "rollback_deployment", "status", "succeeded")
	telemetry.Count(ctx, telemetry.RemVerifyPassed, "action_type", "rollback_deployment", "status", "passed")
	telemetry.Observe(ctx, telemetry.RemDuration, 0.05, "operation", "execute")
	rec := httptest.NewRecorder()
	telemetry.MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	for _, name := range []string{
		"sentinel_remediation_requested",
		"sentinel_remediation_approved",
		"sentinel_remediation_started",
		"sentinel_remediation_succeeded",
		"sentinel_remediation_verification_passed",
	} {
		if !strings.Contains(body, name) {
			t.Fatalf("missing %s\n%s", name, body)
		}
	}
}
