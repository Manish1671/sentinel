# apps/web

Next.js operator console for Sentinel (Phase 8D — final operator experience on live control-plane data).

This application is the human interface for incident operations. It does not own incident state, detection, or remediation execution.

## Responsibility

- Operator shell, navigation, and design system
- Presentation of incidents, investigations, recommendations, and remediations
- Typed HTTP client toward `apps/api` through a same-origin proxy

It does **not**:

- Consume Kafka
- Execute remediations
- Call model providers
- Invent production metrics
- Talk to internal service ports (ingestion, detection, incident, AI, remediation)

## Stack

- Next.js 15 (App Router) + React 19
- TypeScript (strict)
- Tailwind CSS v4
- shadcn/ui (Base UI primitives)
- Lucide icons
- Recharts
- Framer Motion (subtle transitions only)

## Local development

From the repository root, start PostgreSQL, Redis, Kafka, internal services, and `apps/api` first. Then:

```bash
cd apps/web
cp .env.example .env.local
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). Unauthenticated visits redirect to `/login`. `/` redirects to `/overview`.

Server-only `API_URL` (default `http://127.0.0.1:8080`) is used by the Next.js proxy at `/api/v1/*`. The browser never calls `apps/api` or internal services directly.

```bash
npm run lint
npm run test
npm run build
```

### Sign-in (local seed users)

Password for all seed users: `sentinel-dev`

| Email | Role |
| --- | --- |
| `maya.chen@sentinel.dev` | admin |
| `jordan.hale@sentinel.dev` | approver |
| `sam.okonkwo@sentinel.dev` | responder |
| `riley.park@sentinel.dev` | viewer |

Use an approver/admin to approve pending remediations.

## Authentication

1. Browser `POST /api/v1/auth/login` (same origin).
2. Next.js proxies to `apps/api`, receives the JWT, stores it in the HttpOnly `sentinel_session` cookie, and strips `data.token` from the JSON returned to JavaScript.
3. Later requests send the cookie. The proxy attaches `Authorization: Bearer`.
4. Logout revokes the session in PostgreSQL and clears the cookie.

`apps/api` still returns `data.token` to non-browser clients (curl). Role checks in the UI only hide controls; the Go API remains authoritative.

## Data flow

```
Browser
  → Next.js (apps/web)  /api/v1/* proxy + HttpOnly cookie
  → Go control plane (apps/api)
  → PostgreSQL reads / remediation service for approve|reject
```

## Polling

Live WebSocket/SSE is not in this phase. `src/lib/refresh.ts` centralizes intervals so SSE can replace polling later:

| Surface | Interval |
| --- | --- |
| Overview | 20s |
| Incident workspace (active) | 8s |
| Incident workspace (resolved/closed) | 20s |
| Remediation running/verifying | 3s |
| Catalog lists | 30s |

## Mock-data boundary

`src/lib/mock/**` remains for isolated component development. Production pages and `src/lib/api` do **not** import it. Fields the backend does not expose (historical telemetry series, fabricated latency/error charts) render as explicit unavailable states.

## Routes

| Path | Data source |
| --- | --- |
| `/login` | `POST /api/v1/auth/login` |
| `/overview` | services, incidents, pending remediations, featured incident |
| `/services` | `GET /api/v1/services` |
| `/services/[id]` | `GET /api/v1/services/{id}` plus incidents and attached alerts |
| `/incidents` | `GET /api/v1/incidents` |
| `/incidents/[id]` | Flagship workspace (incident, timeline, alerts, investigation, recommendation, remediation) |
| `/investigations` | `GET /api/v1/investigations` |
| `/remediations` | `GET /api/v1/remediations` + approval mutations |
| `/deployments` | `GET /api/v1/deployments` |
| `/settings` | operator chrome; system status from `GET /api/ready` → `apps/api` `GET /ready` |

## Incident workspace data flow

```
Browser
  → Next.js /api/v1/* (cookie session) and GET /api/ready
  → Go apps/api
  → PostgreSQL incident / investigation / recommendation / remediation rows
  → services/remediation only for approve | reject
```

The live pages do not import `src/lib/mock`. Partial subsection failures render in place. Polling uses `src/lib/refresh.ts`. SSE is not implemented: the control plane has no event stream.

Phase 8D completes the operator product surface. Historical telemetry APIs remain unavailable.

## Environment

| Name | Public? | Purpose |
| --- | --- | --- |
| `API_URL` | no | Go control-plane base URL for the server-side proxy |
| `API_INTERNAL_URL` | no | Alias for `API_URL` |

Do not put database credentials, AI keys, or JWT secrets in `NEXT_PUBLIC_*` variables.

## Status

Phase 8D completes the operator console. Historical telemetry APIs remain unavailable. Observability and Kubernetes/AWS are later phases.
