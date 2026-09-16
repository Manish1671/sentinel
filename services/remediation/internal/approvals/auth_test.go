package approvals

import "testing"

func TestCanApprove(t *testing.T) {
	if CanApprove("viewer", "approver") || CanApprove("responder", "approver") {
		t.Fatal("viewer/responder cannot approve")
	}
	if !CanApprove("approver", "approver") || !CanApprove("admin", "approver") {
		t.Fatal("approver/admin can approve")
	}
	if CanApprove("approver", "admin") {
		t.Fatal("approver cannot satisfy admin-required")
	}
	if !CanApprove("admin", "admin") {
		t.Fatal("admin can satisfy admin-required")
	}
	if !CanInspect("viewer") {
		t.Fatal("viewer can inspect")
	}
}
