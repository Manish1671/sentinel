# Frontend architecture

Phase 8B connects the Phase 8A operator console to the Go control-plane API. Phase 8C polishes `/incidents/[id]` as the flagship workspace. The visual system from 8A is unchanged: dark-first, technical, information-dense.

## Visual direction

Sentinel is an AI-native production reliability platform. The console is:

- dark-first
- technical and information-dense
- calm, high-contrast, restrained color
- original (inspired by Linear / Vercel / Datadog *discipline*, not copied)

Geist Sans is the UI face. Geist Mono is used for technical identifiers.

## Application shell

Desktop layout:

```
┌──────────────────────────────────────────────┐
│ top bar                                      │
├────────────┬─────────────────────────────────┤
│ sidebar    │ main content                    │
└────────────┴─────────────────────────────────┘
```

Unauthenticated visitors hit `/login`. The console layout loads `GET /api/v1/me` and redirects on `401`.

## Data flow

```
Browser
  → Next.js (apps/web)
       same-origin /api/v1/* proxy
       HttpOnly sentinel_session cookie
  → Go control plane (apps/api)
  → PostgreSQL
  → services/remediation (approve / reject only)
```

The browser does **not** call ingestion, detection, incident, AI, or remediation ports.

## Authentication

`apps/api` issues a JWT bound to `auth_sessions`. The Next.js proxy stores it in `sentinel_session` (HttpOnly, SameSite=Lax) and strips the token from the login response body. Subsequent proxy calls attach `Authorization: Bearer`. Logout revokes the session and clears the cookie.

UI role checks hide approval controls for `viewer` / `responder`. Backend authorization is authoritative.

## Refresh

Polling lives in `src/lib/refresh.ts` so WebSocket/SSE can replace it later. Incident workspace polls faster while the incident is active; remediations poll faster while `approved` / `running` / `verifying`.

## Incident Workspace

`/incidents/[id]` is the flagship operator screen. It tells the incident story from detection through recovery using control-plane data only.

```
Browser
  → Next.js (apps/web)  /api/v1/*
  → Go control plane (apps/api)
  → incident / investigation / recommendation / remediation records
```

Reads:

- `GET /api/v1/incidents/{id}`
- `GET /api/v1/incidents/{id}/timeline`
- `GET /api/v1/incidents/{id}/alerts`
- `GET /api/v1/incidents/{id}/investigations` then `GET /api/v1/investigations/{id}`
- `GET /api/v1/incidents/{id}/recommendations`
- `GET /api/v1/incidents/{id}/remediations`
- `GET /api/v1/services/{id}` for deployment context when present

Mutations (approver/admin only; backend authorizes):

- `POST /api/v1/remediations/{id}/approve`
- `POST /api/v1/remediations/{id}/reject`

Correlation signals come from timeline `payload.correlation` produced by `services/incident`. They are explainability flags, not a score.

### Telemetry

No historical telemetry endpoint was added. `telemetry_events` holds sparse seed samples, not a coherent latency / error-rate / DB-utilization series. The workspace renders an explicit unavailable state instead of fabricating charts. Ingestion was not changed.

## Mock-data boundary

`src/lib/mock` is isolated fixture data for component development. Live routes use `src/lib/api` only. Missing backend fields (historical telemetry series, before/after recovery pairs) render unavailable — they are never filled with fake production numbers.

## Component organization

| Area | Role |
| --- | --- |
| `components/ui` | shadcn/Base UI primitives |
| `components/core` | Product wrappers |
| `components/incident` | Flagship workspace presentation |
| `components/ai` | Evidence / hypothesis / recommendation language |
| `components/charts` | Optional series; empty when none |
| `lib/api` | Typed clients, errors, proxy-facing fetch |

## Accessibility and responsive

Semantic landmarks, skip link, labelled icon buttons, visible focus, status/severity text + marker, collapsing nav.
