package executor

import (
	"os"
	"strings"
)

// Fault injection is disabled unless SENTINEL_FAULT_INJECTION=true.
// These flags are for evaluation/demo only. They are read at call time.

func faultInjectionEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("SENTINEL_FAULT_INJECTION")), "true")
}

func faultVerifyFail() bool {
	return faultInjectionEnabled() && strings.EqualFold(strings.TrimSpace(os.Getenv("REMEDIATION_FAULT_VERIFY_FAIL")), "true")
}

func faultExecuteFail() bool {
	return faultInjectionEnabled() && strings.EqualFold(strings.TrimSpace(os.Getenv("REMEDIATION_FAULT_EXECUTE_FAIL")), "true")
}
