package verification

import (
	"fmt"

	"github.com/sentinel-dev/sentinel/services/remediation/internal/config"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/executor"
)

type Criteria struct {
	MaxErrorRate     float64
	MaxLatencyMS     float64
	MaxDBUtilization float64
	AllowedHealth    []string
}

func ForAction(action string) Criteria {
	base := Criteria{
		MaxErrorRate:     0.05,
		MaxLatencyMS:     1000,
		MaxDBUtilization: 0.90,
		AllowedHealth:    []string{"healthy", "recovering"},
	}
	switch action {
	case config.ActionRollback:
		base.MaxErrorRate = 0.05
		base.MaxLatencyMS = 500
		base.MaxDBUtilization = 0.80
		base.AllowedHealth = []string{"healthy", "recovering"}
	case config.ActionRestart:
		base.MaxErrorRate = 0.05
		base.MaxLatencyMS = 800
		base.AllowedHealth = []string{"healthy", "recovering"}
	case config.ActionScale:
		base.MaxErrorRate = 0.08
		base.MaxLatencyMS = 900
	}
	return base
}

type Outcome struct {
	Passed  bool
	Status  string
	Summary string
	Checks  map[string]any
}

func Evaluate(action string, st executor.State) Outcome {
	c := ForAction(action)
	checks := map[string]any{
		"error_rate":      st.ErrorRate,
		"latency_ms":      st.LatencyMS,
		"db_utilization":  st.DBUtilization,
		"health_status":   st.HealthStatus,
		"current_version": st.CurrentVersion,
		"replicas":        st.Replicas,
	}
	healthOK := false
	for _, h := range c.AllowedHealth {
		if st.HealthStatus == h {
			healthOK = true
			break
		}
	}
	ok := st.ErrorRate <= c.MaxErrorRate &&
		st.LatencyMS <= c.MaxLatencyMS &&
		st.DBUtilization <= c.MaxDBUtilization &&
		healthOK
	status := "failed"
	summary := "verification failed"
	if ok {
		status = "passed"
		summary = fmt.Sprintf("verification passed for %s", action)
	}
	checks["passed"] = ok
	return Outcome{Passed: ok, Status: status, Summary: summary, Checks: checks}
}
