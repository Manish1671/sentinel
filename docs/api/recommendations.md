# Recommendations

## GET /api/v1/incidents/:id/recommendations

**Purpose.** List recommendations for an incident (operator approval queue).

**Auth.** Bearer. Roles: any authenticated.

**Query.** `status` optional.

**Response `200`.**

```json
{
  "data": [
    {
      "id": "55555555-5555-4555-8555-555555555551",
      "incident_id": "33333333-3333-4333-8333-333333333334",
      "investigation_id": "44444444-4444-4444-8444-444444444441",
      "action_type": "rollback_deployment",
      "title": "Roll back payments-api to 1.17.4",
      "rationale": "1.18.0 introduced a 150ms synchronous reserve that inventory cannot meet.",
      "target_service_id": "22222222-2222-4222-8222-222222222221",
      "parameters": {
        "to_version": "1.17.4",
        "to_deployment_id": "aaaaaaa1-0000-4000-8000-000000000004",
        "from_version": "1.18.0"
      },
      "confidence": 0.88,
      "risk_level": "high",
      "required_approval_role": "approver",
      "status": "proposed"
    }
  ]
}
```

**Status.** `200` · `401` · `404` (incident).
