package verification

import (
	"testing"

	"github.com/sentinel-dev/sentinel/services/remediation/internal/executor"
)

func TestEvaluateRollback(t *testing.T) {
	bad := Evaluate("rollback_deployment", executor.State{
		ErrorRate: 0.2, LatencyMS: 1600, DBUtilization: 0.96, HealthStatus: "unhealthy",
	})
	if bad.Passed {
		t.Fatal("degraded signals must fail")
	}
	good := Evaluate("rollback_deployment", executor.State{
		ErrorRate: 0.01, LatencyMS: 180, DBUtilization: 0.35, HealthStatus: "recovering",
	})
	if !good.Passed {
		t.Fatalf("expected pass: %+v", good)
	}
}
