# Frontend architecture

Phase 8B connects the Phase 8A operator console to the Go control-plane API. Phase 8C is the flagship incident workspace. Phase 8D finishes the operator product surface. The visual system from 8A is unchanged: dark-first, technical, information-dense.

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

Live updates use polling in `src/lib/refresh.ts`. SSE was assessed and deferred: `apps/api` has no incident/investigation/remediation event stream, and adding one would be new infrastructure. Pages already share `useApiResource`; swapping the transport later does not require rewriting screens.

| Surface | Interval |
| --- | --- |
| Overview | 20s |
| Control-plane readiness | 20s |
| Incident workspace (active) | 8s |
| Incident workspace (resolved/closed) | 20s |
| Remediation pending / running / verifying | 20s / 3s |
| Catalog lists | 30s |

## Routes

| Path | Data |
| --- | --- |
| `/overview` | Production snapshot: health, featured incident, active incidents, attention services, activity |
| `/services` | Catalog |
| `/services/[id]` | Service header, health, current incident, recent alerts (from incidents), deployments |
| `/incidents` | Filterable incident list |
| `/incidents/[id]` | Flagship workspace |
| `/investigations` | Investigation queue with incident links |
| `/remediations` | Lifecycle-grouped queue; approve/reject via control plane |
| `/deployments` | Catalog deployments; incident link when the same service has a real incident whose title includes the version or whose detection time falls in a 36h window after the deploy |
| `/settings` | Preferences + `GET /ready` system status |

`GET /api/ready` (Next.js) proxies unauthenticated `GET /ready` on `apps/api`. The top bar never claims Operational if that probe fails.

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

Semantic landmarks, skip link, labelled icon buttons, visible focus, status/severity text + marker, collapsing nav. Desktop is primary; sidebar hides below `md`, lists stack, command search remains available from the top bar.
