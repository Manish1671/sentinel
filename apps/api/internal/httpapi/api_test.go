package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sentinel-dev/sentinel/apps/api/internal/auth"
	"github.com/sentinel-dev/sentinel/apps/api/internal/cache"
	"github.com/sentinel-dev/sentinel/apps/api/internal/config"
	"github.com/sentinel-dev/sentinel/apps/api/internal/database"
	"github.com/sentinel-dev/sentinel/apps/api/internal/incidents"
	"github.com/sentinel-dev/sentinel/apps/api/internal/observability"
	"github.com/sentinel-dev/sentinel/apps/api/internal/services"
)

var (
	testOnce sync.Once
	testSrv  *Server
	testErr  error
)

func testServer(t *testing.T) *Server {
	t.Helper()
	testOnce.Do(func() {
		dbURL := os.Getenv("TEST_DATABASE_URL")
		if dbURL == "" {
			dbURL = os.Getenv("DATABASE_URL")
		}
		if dbURL == "" {
			dbURL = "postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable"
		}
		cfg := config.Config{
			DatabaseURL:     dbURL,
			Port:            8080,
			Environment:     "test",
			LogLevel:        "error",
			AuthTokenSecret: "local-dev-token-secret",
			AuthTokenTTL:    time.Hour,
			RequestTimeout:  5 * time.Second,
			ReadTimeout:     5 * time.Second,
			WriteTimeout:    5 * time.Second,
			IdleTimeout:     5 * time.Second,
			MigrationsPath:  migrationsPath(t),
		}
		if err := cfg.Validate(); err != nil {
			testErr = err
			return
		}
		ctx := context.Background()
		db, err := database.Connect(ctx, cfg)
		if err != nil {
			testErr = err
			return
		}
		if err := database.Migrate(ctx, db, cfg.MigrationsPath); err != nil {
			testErr = err
			return
		}
		if err := applySeeds(ctx, dbURL, filepath.Join(filepath.Dir(cfg.MigrationsPath), "seeds")); err != nil {
			testErr = err
			return
		}
		redis, err := cache.Connect("")
		if err != nil {
			testErr = err
			return
		}
		log := observability.NewLogger("error")
		authSvc := auth.NewService(auth.NewRepository(db.Pool), auth.NewTokenService(cfg.AuthTokenSecret, cfg.AuthTokenTTL))
		testSrv = New(
			cfg,
			log,
			db,
			redis,
			authSvc,
			services.NewService(services.NewRepository(db.Pool)),
			incidents.NewService(incidents.NewRepository(db.Pool)),
		)
	})
	if testErr != nil {
		t.Fatalf("postgres required for integration tests: %v", testErr)
	}
	return testSrv
}

func applySeeds(ctx context.Context, dbURL, seedsDir string) error {
	cfg, err := pgx.ParseConfig(dbURL)
	if err != nil {
		return err
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	var n int
	if err := conn.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	files, err := filepath.Glob(filepath.Join(seedsDir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if _, err := conn.Exec(ctx, string(body)); err != nil {
			return err
		}
	}
	return nil
}

func migrationsPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "database", "migrations"))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("migrations %s: %v", path, err)
	}
	return path
}

func doJSON(t *testing.T, h http.Handler, method, path, token string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func loginToken(t *testing.T, h http.Handler, email, password string) string {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email":    email,
		"password": password,
	}, nil)
	if rec.Code != 200 {
		t.Fatalf("login %s: %d %s", email, rec.Code, rec.Body.String())
	}
	var out struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Data.Token == "" {
		t.Fatal("empty token")
	}
	return out.Data.Token
}

func TestHealthAndUnauthenticatedErrorEnvelope(t *testing.T) {
	s := testServer(t)
	rec := doJSON(t, s.Handler(), http.MethodGet, "/health", "", nil, nil)
	if rec.Code != 200 {
		t.Fatalf("health=%d", rec.Code)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("missing request id")
	}
	rec = doJSON(t, s.Handler(), http.MethodGet, "/api/v1/me", "", nil, nil)
	if rec.Code != 401 {
		t.Fatalf("me=%d %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != "unauthenticated" || env.Error.RequestID == "" {
		t.Fatalf("envelope=%+v", env)
	}
}

func TestReady(t *testing.T) {
	s := testServer(t)
	rec := doJSON(t, s.Handler(), http.MethodGet, "/ready", "", nil, nil)
	if rec.Code != 200 {
		t.Fatalf("ready=%d %s", rec.Code, rec.Body.String())
	}
}

func TestAuthMeLogout(t *testing.T) {
	s := testServer(t)
	h := s.Handler()
	token := loginToken(t, h, "sam.okonkwo@sentinel.dev", "sentinel-dev")
	rec := doJSON(t, h, http.MethodGet, "/api/v1/me", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("me=%d %s", rec.Code, rec.Body.String())
	}
	var me struct {
		Data struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &me)
	if me.Data.Email != "sam.okonkwo@sentinel.dev" || me.Data.Role != "responder" {
		t.Fatalf("me=%s", rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPost, "/api/v1/auth/logout", token, nil, nil)
	if rec.Code != 204 {
		t.Fatalf("logout=%d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/v1/me", token, nil, nil)
	if rec.Code != 401 {
		t.Fatalf("after logout me=%d", rec.Code)
	}
}

func TestLoginRejected(t *testing.T) {
	s := testServer(t)
	rec := doJSON(t, s.Handler(), http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email":    "sam.okonkwo@sentinel.dev",
		"password": "wrong",
	}, nil)
	if rec.Code != 401 {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestViewerForbiddenCreate(t *testing.T) {
	s := testServer(t)
	h := s.Handler()
	token := loginToken(t, h, "riley.park@sentinel.dev", "sentinel-dev")
	rec := doJSON(t, h, http.MethodPost, "/api/v1/incidents", token, map[string]any{
		"service_id": "22222222-2222-4222-8222-222222222225",
		"title":      "should fail",
		"severity":   "low",
	}, map[string]string{"Idempotency-Key": "viewer-cannot-create-1"})
	if rec.Code != 403 {
		t.Fatalf("code=%d %s", rec.Code, rec.Body.String())
	}
}

func TestServicesAndIncidents(t *testing.T) {
	s := testServer(t)
	h := s.Handler()
	token := loginToken(t, h, "sam.okonkwo@sentinel.dev", "sentinel-dev")

	rec := doJSON(t, h, http.MethodGet, "/api/v1/services", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("services=%d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/v1/services/22222222-2222-4222-8222-222222222221", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("service=%d %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, h, http.MethodGet, "/api/v1/incidents", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("incidents=%d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/v1/incidents/33333333-3333-4333-8333-333333333334", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("incident=%d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/v1/incidents/33333333-3333-4333-8333-333333333334/timeline", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("timeline=%d %s", rec.Code, rec.Body.String())
	}
	var tl struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &tl)
	if len(tl.Data) == 0 {
		t.Fatal("expected timeline events")
	}
}

func TestIncidentIdempotency(t *testing.T) {
	s := testServer(t)
	h := s.Handler()
	token := loginToken(t, h, "sam.okonkwo@sentinel.dev", "sentinel-dev")
	key := "inc-create-" + uuid.NewString()
	body := map[string]any{
		"service_id": "22222222-2222-4222-8222-222222222225",
		"title":      "Notifications lag impacting password-reset email",
		"summary":    "Opened from Phase 2 API test",
		"severity":   "medium",
		"alert_ids":  []string{"eeeeeeee-0000-4000-8000-000000000003"},
	}
	rec := doJSON(t, h, http.MethodPost, "/api/v1/incidents", token, body, map[string]string{"Idempotency-Key": key})
	if rec.Code != 201 {
		t.Fatalf("create=%d %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Data struct {
			ID        string `json:"id"`
			Reference string `json:"reference"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.Data.ID == "" {
		t.Fatal("missing id")
	}
	rec = doJSON(t, h, http.MethodPost, "/api/v1/incidents", token, body, map[string]string{"Idempotency-Key": key})
	if rec.Code != 200 {
		t.Fatalf("replay=%d %s", rec.Code, rec.Body.String())
	}
	var replay struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &replay)
	if replay.Data.ID != created.Data.ID {
		t.Fatalf("id mismatch %s %s", replay.Data.ID, created.Data.ID)
	}
	rec = doJSON(t, h, http.MethodGet, "/api/v1/incidents/"+created.Data.ID+"/timeline", token, nil, nil)
	var tl struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &tl)
	createdCount := 0
	for _, ev := range tl.Data {
		if ev["kind"] == "created" {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Fatalf("created events=%d", createdCount)
	}
	body["title"] = "Different title"
	rec = doJSON(t, h, http.MethodPost, "/api/v1/incidents", token, body, map[string]string{"Idempotency-Key": key})
	if rec.Code != 409 {
		t.Fatalf("conflict=%d %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Error.Code != "idempotency_key_conflict" {
		t.Fatalf("code=%s", env.Error.Code)
	}
}

func TestIncidentByReferenceAndOpsReads(t *testing.T) {
	s := testServer(t)
	h := s.Handler()
	token := loginToken(t, h, "sam.okonkwo@sentinel.dev", "sentinel-dev")

	rec := doJSON(t, h, http.MethodGet, "/api/v1/incidents/INC-2026-0004", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("by ref=%d %s", rec.Code, rec.Body.String())
	}
	var incident struct {
		Data struct {
			Reference  string `json:"reference"`
			ServiceSlug string `json:"service_slug"`
			Environment string `json:"environment"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &incident)
	if incident.Data.Reference != "INC-2026-0004" || incident.Data.ServiceSlug != "payments-api" {
		t.Fatalf("incident=%s", rec.Body.String())
	}

	rec = doJSON(t, h, http.MethodGet, "/api/v1/incidents/INC-2026-0004/alerts", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("alerts=%d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/v1/incidents/INC-2026-0004/investigations", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("investigations=%d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/v1/investigations/44444444-4444-4444-8444-444444444441", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("investigation=%d %s", rec.Code, rec.Body.String())
	}
	var inv struct {
		Data struct {
			Status   string `json:"status"`
			Evidence []any  `json:"evidence"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &inv)
	if inv.Data.Status != "completed" || len(inv.Data.Evidence) == 0 {
		t.Fatalf("investigation body=%s", rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/v1/incidents/INC-2026-0004/recommendations", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("recommendations=%d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/v1/incidents/INC-2026-0004/remediations", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("remediations=%d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/v1/deployments", token, nil, nil)
	if rec.Code != 200 {
		t.Fatalf("deployments=%d %s", rec.Code, rec.Body.String())
	}
}

func TestCookieAuth(t *testing.T) {
	s := testServer(t)
	h := s.Handler()
	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email":    "sam.okonkwo@sentinel.dev",
		"password": "sentinel-dev",
	}, nil)
	if rec.Code != 200 {
		t.Fatalf("login=%d %s", rec.Code, rec.Body.String())
	}
	cookie := rec.Result().Cookies()
	var session *http.Cookie
	for _, c := range cookie {
		if c.Name == "sentinel_session" {
			session = c
			break
		}
	}
	if session == nil || session.Value == "" || !session.HttpOnly {
		t.Fatalf("cookie=%v", cookie)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(session)
	got := httptest.NewRecorder()
	h.ServeHTTP(got, req)
	if got.Code != 200 {
		t.Fatalf("me cookie=%d %s", got.Code, got.Body.String())
	}
}

func TestViewerCannotApprove(t *testing.T) {
	s := testServer(t)
	h := s.Handler()
	token := loginToken(t, h, "riley.park@sentinel.dev", "sentinel-dev")
	rec := doJSON(t, h, http.MethodPost, "/api/v1/remediations/66666666-6666-4666-8666-666666666661/approve", token, map[string]string{
		"comment": "no",
	}, map[string]string{"Idempotency-Key": "viewer-cannot-approve-1"})
	if rec.Code != 403 {
		t.Fatalf("code=%d %s", rec.Code, rec.Body.String())
	}
}

func TestApproveWithoutRemediationService(t *testing.T) {
	s := testServer(t)
	h := s.Handler()
	token := loginToken(t, h, "jordan.hale@sentinel.dev", "sentinel-dev")
	rec := doJSON(t, h, http.MethodPost, "/api/v1/remediations/66666666-6666-4666-8666-666666666661/approve", token, map[string]string{
		"comment": "try",
	}, map[string]string{"Idempotency-Key": "approver-no-gateway-1"})
	if rec.Code != 503 {
		t.Fatalf("code=%d %s", rec.Code, rec.Body.String())
	}
}
