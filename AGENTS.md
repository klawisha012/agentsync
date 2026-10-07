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
- Invalidate/refresh: after modifying code, run `graphify update .` (add `--force` if files were deleted or renamed) to keep `graphify-out/` current. For doc changes, run `/graphify --update`.
- If graphify cannot be installed or run because of sandbox, network, credential, or approval restrictions, record the blocker and continue with normal local code navigation.

### git
После каждой задачи (когда считаешь, что выполнил задачу) делай коммит в локальный git и push на github

## Behavioral guidelines (Karpathy)

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

### 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

### 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

### 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

### 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

