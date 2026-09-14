# Security principles

These apply from Phase 0 onward. Implementation comes later; the constraints do not.

## Least privilege

Every service, user, and AI tool receives the minimum access required. Default deny for remediation actions and tool calls.

## Service authentication

Services authenticate to each other and to data stores with identities (local secrets now, IAM roles in production). Unauthenticated internal APIs are not acceptable.

## Role-based authorization

Human roles at minimum: viewer, responder, approver, admin. Opening a console does not grant remediation rights.

## Secrets through environment and configuration management

Secrets live in environment variables or a secret manager. `.env.example` lists names only. Production credentials are never in source control, container images, or docs.

## No production credentials in source code

This includes AWS keys, database passwords, model provider keys, and kubeconfigs.

## AI tool permissions are restricted

The investigator may only invoke an allowlisted tool with validated arguments. Tools return data; they do not expose shells, arbitrary HTTP, or cloud consoles.

## Remediation requires authorization

`services/remediation` checks identity, role, and approval state before any action.

## All remediation actions are auditable

Who requested, who approved, what ran, against which service, with which parameters, and what the result was. Audit records are append-only.

## Destructive actions require explicit approval

Restarts of non-prod may use a lower bar later; production-destructive or data-changing actions always require an `Approval` record before execution.
