package executor

import (
	"context"

	"github.com/google/uuid"
)

type Action struct {
	Type           string
	ServiceID      uuid.UUID
	RemediationID  uuid.UUID
	Parameters     map[string]any
}

type State struct {
	ServiceID         uuid.UUID `json:"service_id"`
	CurrentVersion    string    `json:"current_version"`
	PreviousVersion   string    `json:"previous_version,omitempty"`
	HealthStatus      string    `json:"health_status"`
	Replicas          int       `json:"replicas"`
	ErrorRate         float64   `json:"error_rate"`
	LatencyMS         float64   `json:"latency_ms"`
	DBUtilization     float64   `json:"db_utilization"`
	LastAction        string    `json:"last_action,omitempty"`
	LastRemediationID string    `json:"last_remediation_id,omitempty"`
}

type Result struct {
	State   State
	Summary string
}

// Executor is the seam for a future Kubernetes implementation.
// The remediation workflow calls only these methods.
type Executor interface {
	Validate(ctx context.Context, action Action) error
	Execute(ctx context.Context, action Action) (Result, error)
	State(ctx context.Context, serviceID uuid.UUID) (State, error)
}
