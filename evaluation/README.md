# Evaluation foundation

Deterministic cases live in `evaluation/cases/`. `evaluation/runner.py` checks keyword overlap against an `InvestigationResult`. It does **not** publish numerical model quality scores.

The architecture is separate from the live Kafka path so labeled eval runs (`evaluation_runs` table) can be added later without mutating production incident state.
