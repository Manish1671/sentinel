package recommendations

import "testing"

func TestCanonicalAction(t *testing.T) {
	got, err := CanonicalAction("scale_service")
	if err != nil || got != "scale_replicas" {
		t.Fatalf("alias: %s %v", got, err)
	}
	if _, err := CanonicalAction("kubectl"); err == nil {
		t.Fatal("unknown action must be rejected")
	}
	if _, err := CanonicalAction("rollback_deployment"); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeParams(t *testing.T) {
	if _, err := NormalizeParams("rollback_deployment", map[string]any{}); err == nil {
		t.Fatal("missing rollback target")
	}
	if _, err := NormalizeParams("rollback_deployment", map[string]any{"version_hint": "previous"}); err != nil {
		t.Fatal(err)
	}
	p, err := NormalizeParams("rollback_deployment", map[string]any{"to_version": "1.17.4"})
	if err != nil || p["to_version"] != "1.17.4" {
		t.Fatalf("%v %v", p, err)
	}
	if _, err := NormalizeParams("scale_replicas", map[string]any{"replicas": 99}); err == nil {
		t.Fatal("replicas out of range")
	}
	p, err = NormalizeParams("scale_replicas", map[string]any{"replicas": 4.0})
	if err != nil || p["replicas"] != 4 {
		t.Fatalf("%v %v", p, err)
	}
}

func TestEligible(t *testing.T) {
	if EligibleIncident("closed") || EligibleIncident("resolved") {
		t.Fatal("closed/resolved not eligible")
	}
	if !EligibleIncident("investigating") {
		t.Fatal("investigating should be eligible")
	}
	if EligibleRecommendation("expired") {
		t.Fatal("expired rec")
	}
}
