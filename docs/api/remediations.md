# Remediations

Creating a remediation row from a proposed recommendation is an internal step when an investigation completes (or later, an explicit accept endpoint). Phase 1 public surface is fetch + approve/reject.

Approve/reject are executed by `services/remediation`. `apps/api` authenticates/authorizes and proxies the decision. Reads come from PostgreSQL.

## GET /api/v1/remediations

**Purpose.** List remediations.

**Auth.** Bearer. Roles: any authenticated.

**Query.** `incident_id`, `status`, `limit`, `cursor`.

## GET /api/v1/incidents/:id/remediations

**Purpose.** Remediations for one incident.

## GET /api/v1/remediations/:id

**Purpose.** Remediation record including approval.

**Auth.** Bearer. Roles: any authenticated.

**Response `200`.**

```json
{
  "data": {
    "id": "66666666-6666-4666-8666-666666666661",
    "incident_id": "33333333-3333-4333-8333-333333333334",
    "recommendation_id": "55555555-5555-4555-8555-555555555551",
    "service_id": "22222222-2222-4222-8222-222222222221",
    "status": "pending_approval",
    "action_type": "rollback_deployment",
    "parameters": {
      "to_version": "1.17.4",
      "to_deployment_id": "aaaaaaa1-0000-4000-8000-000000000004"
    },
    "requested_by_user_id": "11111111-1111-4111-8111-111111111113",
    "attempt_number": 1,
    "verification_status": "pending",
    "approval": {
      "id": "88888888-0000-4000-8000-000000000001",
      "decision": "pending",
      "actor_user_id": null,
      "decided_at": null
    }
  }
}
```

**Status.** `200` · `401` · `404`.

## POST /api/v1/remediations/:id/approve

**Purpose.** Record an approved decision. Requires `approver` or `admin` (must meet `required_approval_role` on the recommendation).

**Auth.** Bearer. Roles: `approver`, `admin`.

**Headers.** `Idempotency-Key` required.

**Request.**

```json
{
  "comment": "Rollback approved. Capture volume is acceptable to interrupt."
}
```

`comment` optional, max 2000 chars.

**Response `200`.** Remediation with `status: approved` and approval `decision: approved`. Same key + same decision: `200` without a second timeline event.

**Status.** `200` · `400` · `401` · `403` · `404` · `409` (`approval_already_decided` or `idempotency_key_conflict`).

Cannot approve if remediation is not `pending_approval`.

## POST /api/v1/remediations/:id/reject

**Purpose.** Record a rejected decision. Sets remediation `rejected` and recommendation `rejected`.

**Auth.** Bearer. Roles: `approver`, `admin`.

**Headers.** `Idempotency-Key` required.

**Request.** `{ "comment": "Need a feature-flag disable instead of rollback." }`

**Response `200`.** Remediation `status: rejected`.

**Status.** Same family as approve.
