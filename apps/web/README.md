# apps/web

Next.js operator console for Sentinel.

## Responsibility

This application is the human interface for incident operations. It does not own incident state, detection, or remediation execution.

Planned surfaces:

- dashboard
- services
- incidents
- incident details
- investigation timeline
- AI recommendations
- remediation approval
- system health
- observability views

## Stack (planned)

Next.js, TypeScript, Tailwind CSS, TanStack Query, Recharts.

## Boundaries

- Talks to `apps/api` over HTTP.
- Does not consume Kafka directly.
- Does not execute remediation actions.
- Does not call AI model providers directly; investigation results come through the API.

## Status

Not implemented. This directory is a placeholder for Phase 6 UI work.
