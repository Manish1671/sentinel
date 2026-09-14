# services/remediation

Authorized remediation workflow service (Go).

## Responsibility

Turn an approved recommendation into a controlled, auditable action and verify health afterward.

Planned work:

- accept `remediation.requested`
- validate the proposed action against allowlists and runbooks
- enforce human approval
- execute only safe, explicitly defined actions
- verify post-remediation health
- publish `remediation.completed`
- write an audit trail for every attempt

## Boundaries

- Never grants AI or operators unrestricted production access.
- Destructive or high-impact actions require explicit approval.
- Execution targets are mediated integrations, not arbitrary shells.

## Status

Not implemented. Approval and audit requirements are documented in `docs/architecture/security.md`.
