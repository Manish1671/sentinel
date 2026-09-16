package recommendations

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/services/remediation/internal/config"
)

// Canonical allowlist. Phase 1 stores scale as scale_replicas (spec "scale_service").
var Allowlist = map[string]Definition{
	config.ActionRollback: {
		Name:              config.ActionRollback,
		ApprovalRequired:  true,
		DefaultRisk:       "high",
		Verification:      "rollback",
		RequiredParamsAny: []string{"to_version", "to_deployment_id", "version_hint"},
	},
	config.ActionRestart: {
		Name:             config.ActionRestart,
		ApprovalRequired: true,
		DefaultRisk:      "medium",
		Verification:     "restart",
	},
	config.ActionScale: {
		Name:             config.ActionScale,
		ApprovalRequired: true,
		DefaultRisk:      "medium",
		Verification:     "scale",
		RequiredParamsAny: []string{"replicas"},
	},
}

type Definition struct {
	Name              string
	ApprovalRequired  bool
	DefaultRisk       string
	Verification      string
	RequiredParamsAny []string
}

func CanonicalAction(name string) (string, error) {
	n := strings.TrimSpace(name)
	if n == config.ActionScaleAlias {
		n = config.ActionScale
	}
	if _, ok := Allowlist[n]; !ok {
		return "", fmt.Errorf("unknown action %q", name)
	}
	return n, nil
}

func NormalizeParams(action string, in map[string]any) (map[string]any, error) {
	def, ok := Allowlist[action]
	if !ok {
		return nil, fmt.Errorf("unknown action %q", action)
	}
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	if len(def.RequiredParamsAny) > 0 {
		found := false
		for _, k := range def.RequiredParamsAny {
			if _, ok := out[k]; ok && out[k] != nil && fmt.Sprint(out[k]) != "" {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("missing required parameter (one of %s)", strings.Join(def.RequiredParamsAny, ", "))
		}
	}
	if action == config.ActionScale {
		n, err := asInt(out["replicas"])
		if err != nil {
			return nil, fmt.Errorf("replicas must be an integer")
		}
		if n < 1 || n > 20 {
			return nil, fmt.Errorf("replicas must be between 1 and 20")
		}
		out["replicas"] = n
	}
	return out, nil
}

func asInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int32:
		return int(n), nil
	case int64:
		return int(n), nil
	case float64:
		if n != float64(int(n)) {
			return 0, fmt.Errorf("not int")
		}
		return int(n), nil
	default:
		var i int
		_, err := fmt.Sscan(fmt.Sprint(v), &i)
		return i, err
	}
}

func EligibleIncident(status string) bool {
	switch status {
	case "open", "investigating", "remediating", "verifying":
		return true
	default:
		return false
	}
}

func EligibleRecommendation(status string) bool {
	switch status {
	case "proposed", "accepted":
		return true
	default:
		return false
	}
}

func TargetMatches(recTarget, incidentService, paramService uuid.UUID) bool {
	if recTarget != uuid.Nil && recTarget != incidentService {
		return false
	}
	if paramService != uuid.Nil && paramService != incidentService {
		return false
	}
	return true
}
