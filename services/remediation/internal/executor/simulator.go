package executor

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sentinel-dev/sentinel/services/remediation/internal/config"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/database"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/recommendations"
)

// Simulator mutates remediation_simulator_state and services.health_status only.
// It never shells out, never calls kubectl, and never touches the host.
type Simulator struct {
	DB *database.DB
}

func NewSimulator(db *database.DB) *Simulator { return &Simulator{DB: db} }

func (s *Simulator) Validate(_ context.Context, action Action) error {
	_, err := recommendations.CanonicalAction(action.Type)
	if err != nil {
		return err
	}
	if action.ServiceID == uuid.Nil {
		return fmt.Errorf("target service is required")
	}
	return nil
}

func (s *Simulator) State(ctx context.Context, serviceID uuid.UUID) (State, error) {
	st, err := s.loadOrInit(ctx, serviceID)
	if err != nil {
		return State{}, err
	}
	return st, nil
}

func (s *Simulator) Execute(ctx context.Context, action Action) (res Result, err error) {
	if faultExecuteFail() {
		return Result{}, fmt.Errorf("sentinel fault injection: remediation execute failed (test-only)")
	}
	if err := s.Validate(ctx, action); err != nil {
		return Result{}, err
	}
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	st, err := loadTx(ctx, tx, action.ServiceID)
	if err != nil {
		return Result{}, err
	}
	canonical, _ := recommendations.CanonicalAction(action.Type)
	switch canonical {
	case config.ActionRollback:
		to := stringParam(action.Parameters, "to_version")
		if to == "" {
			to = st.PreviousVersion
		}
		if to == "" && stringParam(action.Parameters, "version_hint") == "previous" {
			to = st.PreviousVersion
		}
		if to == "" {
			return Result{}, fmt.Errorf("rollback target version is unknown")
		}
		st.PreviousVersion = st.CurrentVersion
		st.CurrentVersion = to
		st.HealthStatus = "recovering"
		st.ErrorRate = 0.01
		st.LatencyMS = 180
		st.DBUtilization = 0.35
	case config.ActionRestart:
		st.HealthStatus = "recovering"
		st.ErrorRate = 0.02
		st.LatencyMS = 220
		st.DBUtilization = 0.40
	case config.ActionScale:
		n, err := recommendations.NormalizeParams(config.ActionScale, action.Parameters)
		if err != nil {
			return Result{}, err
		}
		st.Replicas = n["replicas"].(int)
		st.HealthStatus = "recovering"
		st.ErrorRate = 0.03
		st.LatencyMS = 250
		st.DBUtilization = 0.45
	default:
		return Result{}, fmt.Errorf("unknown action")
	}
	st.LastAction = canonical
	st.LastRemediationID = action.RemediationID.String()
	if faultVerifyFail() {
		// Leave signals above verification criteria so recovery must not be claimed.
		st.HealthStatus = "unhealthy"
		st.ErrorRate = 0.20
		st.LatencyMS = 2200
		st.DBUtilization = 0.97
	}
	if err := saveTx(ctx, tx, st); err != nil {
		return Result{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE services SET health_status = $2::service_health_status WHERE id = $1`, action.ServiceID, catalogHealth(st.HealthStatus)); err != nil {
		return Result{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, err
	}
	return Result{State: st, Summary: fmt.Sprintf("simulated %s on service %s", canonical, action.ServiceID)}, nil
}

func (s *Simulator) loadOrInit(ctx context.Context, serviceID uuid.UUID) (State, error) {
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		return State{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	st, err := loadTx(ctx, tx, serviceID)
	if err != nil {
		return State{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return State{}, err
	}
	return st, nil
}

func loadTx(ctx context.Context, tx pgx.Tx, serviceID uuid.UUID) (State, error) {
	var exists int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM services WHERE id = $1`, serviceID).Scan(&exists); err != nil {
		return State{}, err
	}
	if exists == 0 {
		return State{}, fmt.Errorf("target service does not exist")
	}
	var st State
	err := tx.QueryRow(ctx, `
		SELECT service_id, current_version, COALESCE(previous_version,''), health_status, replicas,
		       error_rate, latency_ms, db_utilization, COALESCE(last_action,''), COALESCE(last_remediation_id::text,'')
		FROM remediation_simulator_state WHERE service_id = $1
	`, serviceID).Scan(
		&st.ServiceID, &st.CurrentVersion, &st.PreviousVersion, &st.HealthStatus, &st.Replicas,
		&st.ErrorRate, &st.LatencyMS, &st.DBUtilization, &st.LastAction, &st.LastRemediationID,
	)
	if err == nil {
		return st, nil
	}
	if err != pgx.ErrNoRows {
		return State{}, err
	}
	var version, health string
	_ = tx.QueryRow(ctx, `
		SELECT COALESCE((
			SELECT version FROM deployments WHERE service_id = $1 ORDER BY started_at DESC LIMIT 1
		), 'unknown'), health_status::text
		FROM services WHERE id = $1
	`, serviceID).Scan(&version, &health)
	prev := ""
	_ = tx.QueryRow(ctx, `
		SELECT version FROM deployments WHERE service_id = $1
		ORDER BY started_at DESC OFFSET 1 LIMIT 1
	`, serviceID).Scan(&prev)
	st = State{
		ServiceID:       serviceID,
		CurrentVersion:  version,
		PreviousVersion: prev,
		HealthStatus:    health,
		Replicas:        2,
		ErrorRate:       0.11,
		LatencyMS:       1500,
		DBUtilization:   0.96,
	}
	if health == "healthy" {
		st.ErrorRate, st.LatencyMS, st.DBUtilization = 0.002, 24, 0.35
	}
	if err := saveTx(ctx, tx, st); err != nil {
		return State{}, err
	}
	return st, nil
}

func saveTx(ctx context.Context, tx pgx.Tx, st State) error {
	var lastID *uuid.UUID
	if st.LastRemediationID != "" {
		id, err := uuid.Parse(st.LastRemediationID)
		if err == nil {
			lastID = &id
		}
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO remediation_simulator_state (
			service_id, current_version, previous_version, health_status, replicas,
			error_rate, latency_ms, db_utilization, last_action, last_remediation_id, updated_at
		) VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,NULLIF($9,''),$10,now())
		ON CONFLICT (service_id) DO UPDATE SET
			current_version = EXCLUDED.current_version,
			previous_version = EXCLUDED.previous_version,
			health_status = EXCLUDED.health_status,
			replicas = EXCLUDED.replicas,
			error_rate = EXCLUDED.error_rate,
			latency_ms = EXCLUDED.latency_ms,
			db_utilization = EXCLUDED.db_utilization,
			last_action = EXCLUDED.last_action,
			last_remediation_id = EXCLUDED.last_remediation_id,
			updated_at = now()
	`, st.ServiceID, st.CurrentVersion, st.PreviousVersion, st.HealthStatus, st.Replicas,
		st.ErrorRate, st.LatencyMS, st.DBUtilization, st.LastAction, lastID)
	return err
}

func catalogHealth(sim string) string {
	switch sim {
	case "recovering":
		return "degraded"
	case "healthy", "degraded", "unhealthy", "unknown":
		return sim
	default:
		return "unknown"
	}
}

func stringParam(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	v, _ := m[k].(string)
	return v
}

// Failer wraps an executor and fails once when Fail is true.
type Failer struct {
	Inner Executor
	Fail  bool
	Msg   string
}

func (f *Failer) Validate(ctx context.Context, action Action) error {
	return f.Inner.Validate(ctx, action)
}

func (f *Failer) State(ctx context.Context, serviceID uuid.UUID) (State, error) {
	return f.Inner.State(ctx, serviceID)
}

func (f *Failer) Execute(ctx context.Context, action Action) (Result, error) {
	if f.Fail {
		msg := f.Msg
		if msg == "" {
			msg = "simulated executor failure"
		}
		return Result{}, fmt.Errorf("%s", msg)
	}
	return f.Inner.Execute(ctx, action)
}
