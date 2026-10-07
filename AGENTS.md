## Context pointers

- **Issue tracker**: GitHub Issues for klawisha012/agentsync. See `docs/agents/issue-tracker.md`.
- **Triage labels**: five canonical roles matching role names. See `docs/agents/triage-labels.md`.
- **Domain docs**: `CONTEXT.md` and `docs/adr/` at repo root. See `docs/agents/domain.md`.
- **UI & Design**: before visual work, read `docs/DESIGN.md`. Colors, typography, spacing, component roles from it; state and behavior from this repo.

## Graphify (Knowledge graph)

Knowledge graph lives at `graphify-out/` (skill: `~/.agents/skills/graphify/SKILL.md`). Trigger: `/graphify`.
- **Query & Navigation**: when `graphify-out/graph.json` exists, run `graphify query "<question>"`, `graphify path "<A>" "<B>"`, or `graphify explain "<concept>"` before reading raw files or grepping. Use `graphify-out/wiki/index.md` for navigation; read `GRAPH_REPORT.md` only for broad architecture overview.
- **Cold start**: if no `graphify-out/` exists, run `/graphify .` for non-trivial architecture work. Skip for trivial single-file tasks.
- **Invalidate & Update**: after modifying code, run `graphify update .` (add `--force` if files were deleted or renamed) to keep `graphify-out/` current. For doc or asset changes, run `/graphify --update`.
- **Resilience**: if graphify fails due to environment/permissions, record blocker and proceed with standard navigation.

## Workflow & Git lifecycle

- **Post-task completion**: после каждой выполненной задачи делай git commit и `git push origin main`.

## Behavioral guidelines (Karpathy)

Bias toward caution over speed.

### 1. Think Before Coding
- State assumptions explicitly; ask if uncertain.
- Surface tradeoffs and alternative interpretations — never pick silently.
- Push back if a simpler approach exists. Stop and clarify if confusing.

### 2. Simplicity First
- Minimum code that solves the problem. Nothing speculative.
- No unrequested abstractions, configurability, or handling for impossible scenarios.
- Rewrite if 50 lines can replace 200.

### 3. Surgical Changes
- Touch only what the task requires.
- Match existing style; do not refactor working adjacent code or reformat untouched lines.
- Clean up only own orphans (unused imports/functions). Leave pre-existing dead code alone.
- Every changed line must trace directly to the request.

### 4. Goal-Driven Execution
- Turn tasks into verifiable goals (reproduce bug with test → make test pass; check before and after).
- State a brief 1-2-3 step plan with explicit verification criteria for each step.
- Loop until verified.
