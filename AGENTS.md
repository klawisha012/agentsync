## Agent skills

### Issue tracker

Issues live in GitHub Issues for klawisha012/agentsync. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical roles, label strings equal to the role names. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

### Design

Before visual UI work, read `docs/DESIGN.md`. Take colors, type, spacing, and component roles from it. Take fields, copy, states, and behavior from this repo.

### graphify

- `graphify` skill: `~/.agents/skills/graphify/SKILL.md`.
- Trigger: `/graphify`.
- When the user types `/graphify`, invoke the graphify skill before doing anything else.
- For non-trivial codebase or architecture work, if the project has no `graphify-out/`, build the graph first with `/graphify .`, then use it. Skip trivial one-file tasks.
- If graphify cannot be installed or run because of sandbox, network, credential, or approval restrictions, record the blocker and continue with normal local code navigation.

### git
После каждой задачи (когда считаешь, что выполнил задачу) делай коммит в локальный git и push на github
