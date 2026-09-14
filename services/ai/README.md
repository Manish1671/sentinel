# services/ai

Python FastAPI service for bounded AI investigation.

## Responsibility

Investigate incidents using retrieval and explicitly registered tools, then produce scored recommendations.

Planned work:

- consume `investigations.requested`
- gather evidence through controlled tools
- perform root-cause analysis
- produce remediation recommendations with confidence scores
- persist investigation artifacts and evaluation data
- publish `investigations.completed`
- support later evaluation runs (`EvaluationRun`)

## Allowed tool class (planned)

Tools are named, permissioned, and implemented by Sentinel — never raw production access:

- `get_recent_logs`
- `get_metrics`
- `get_service_health`
- `get_recent_deployments`
- `get_trace`
- `search_runbooks`
- `search_previous_incidents`
- `get_deployment_diff`

## Boundaries

- No unrestricted shell.
- No direct cloud credential use beyond the service’s own scoped runtime identity.
- Cannot approve or execute remediation.
- Model provider keys stay in environment/configuration, not in source.

See `docs/ai/README.md` for the investigation flow.

## Status

Not implemented. This directory is a placeholder for Phase 4.
