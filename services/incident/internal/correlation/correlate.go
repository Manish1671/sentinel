package correlation

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

var degradationDetectors = map[string]struct{}{
	"high_latency":              {},
	"high_error_rate":           {},
	"db_connection_saturation":  {},
	"error_log_burst":           {},
}

type Alert struct {
	ID          uuid.UUID
	ServiceID   uuid.UUID
	Environment string
	DetectorID  string
	Severity    string
	StartedAt   time.Time
	Labels      map[string]string
}

type Candidate struct {
	ID              uuid.UUID
	ServiceID       uuid.UUID
	Environment     string
	Status          string
	Severity        string
	DetectedAt      time.Time
	LastActivityAt  time.Time
	DetectorIDs     []string
	DeploymentIDs   []string
}

type Explanation struct {
	SameService        bool `json:"same_service"`
	SameEnvironment    bool `json:"same_environment"`
	WithinWindow       bool `json:"within_window"`
	CompatibleCategory bool `json:"compatible_category"`
	DeploymentContext  bool `json:"deployment_context"`
	CompatibleStatus   bool `json:"compatible_status"`
}

func IsActiveStatus(status string) bool {
	switch status {
	case "open", "investigating", "remediating", "verifying":
		return true
	default:
		return false
	}
}

func IsDegradation(detector string) bool {
	_, ok := degradationDetectors[detector]
	return ok
}

func Match(alert Alert, c Candidate, window time.Duration) (bool, Explanation) {
	exp := Explanation{
		SameService:      alert.ServiceID == c.ServiceID,
		SameEnvironment:  alert.Environment != "" && alert.Environment == c.Environment,
		CompatibleStatus: IsActiveStatus(c.Status),
	}
	anchor := c.LastActivityAt
	if anchor.IsZero() {
		anchor = c.DetectedAt
	}
	delta := alert.StartedAt.Sub(anchor)
	if delta < 0 {
		delta = -delta
	}
	exp.WithinWindow = delta <= window

	exp.CompatibleCategory = compatibleCategory(alert.DetectorID, c.DetectorIDs)
	alertDep := alert.Labels["deployment_id"]
	if alertDep != "" {
		for _, d := range c.DeploymentIDs {
			if d == alertDep {
				exp.DeploymentContext = true
				break
			}
		}
	}
	if !exp.DeploymentContext && strings.EqualFold(alert.Labels["deployment_associated"], "true") {
		for _, d := range c.DeploymentIDs {
			if d != "" {
				exp.DeploymentContext = true
				break
			}
		}
	}

	ok := exp.SameService && exp.SameEnvironment && exp.CompatibleStatus && exp.WithinWindow &&
		(exp.CompatibleCategory || exp.DeploymentContext)
	return ok, exp
}

func Select(alert Alert, candidates []Candidate, window time.Duration) (*Candidate, Explanation) {
	var best *Candidate
	var bestExp Explanation
	for i := range candidates {
		c := candidates[i]
		ok, exp := Match(alert, c, window)
		if !ok {
			continue
		}
		if best == nil || c.LastActivityAt.After(best.LastActivityAt) || (c.LastActivityAt.Equal(best.LastActivityAt) && c.DetectedAt.After(best.DetectedAt)) {
			cp := c
			best = &cp
			bestExp = exp
		}
	}
	if best == nil {
		return nil, Explanation{
			SameService:     true,
			SameEnvironment: true,
		}
	}
	return best, bestExp
}

func compatibleCategory(detector string, existing []string) bool {
	if IsDegradation(detector) {
		if len(existing) == 0 {
			return true
		}
		for _, d := range existing {
			if IsDegradation(d) {
				return true
			}
		}
		return false
	}
	for _, d := range existing {
		if d == detector {
			return true
		}
	}
	return false
}

func MaxSeverity(a, b string) string {
	if rank(a) >= rank(b) {
		return a
	}
	return b
}

func rank(s string) int {
	switch s {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func Title(slug, env string, detectors []string) string {
	if slug == "" {
		slug = "service"
	}
	if env == "" {
		env = "production"
	}
	onlyLatency := true
	hasDegrade := false
	for _, d := range detectors {
		if d != "high_latency" {
			onlyLatency = false
		}
		if IsDegradation(d) {
			hasDegrade = true
		}
	}
	if onlyLatency && hasDegrade {
		return slug + " " + env + " performance incident"
	}
	if hasDegrade {
		return slug + " " + env + " degradation"
	}
	return slug + " " + env + " incident"
}

func Summary(slug string, detectors []string, deploymentVersion string) string {
	parts := strings.Join(unique(detectors), ", ")
	s := "Correlated alerts for " + slug + ": " + parts + "."
	if deploymentVersion != "" {
		s += " Deployment-associated context recorded (version " + deploymentVersion + "). Not treated as root cause."
	}
	return s
}

func unique(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
