# Conventional Commits

NexusAI-Gateway uses Conventional Commits to keep history readable, automate change tracking, and support release governance.

## Format

```text
<type>(<scope>): <description>
```

Use `!` or `BREAKING CHANGE:` when a change is not backward compatible.

## Supported Types

- `feat` - new behavior or capability.
- `fix` - bug fix.
- `refactor` - code change without a user-visible feature or bug fix.
- `perf` - performance improvement.
- `docs` - documentation only.
- `test` - tests added or updated.
- `ci` - CI or automation changes.
- `build` - build system or dependency changes.
- `chore` - maintenance work that does not affect product behavior.
- `revert` - revert a previous commit.

## Suggested Scopes For This Repository

- `auth`
- `config`
- `router`
- `mcp`
- `privacy`
- `storage`
- `db`
- `web`
- `deploy`
- `docs`
- `tests`

## Examples

```text
feat(auth): add support for gateway key bootstrap flow
fix(router): avoid nil handler when database connection is unavailable
refactor(storage): simplify provider repository initialization
perf(mcp): reduce allocations in stream heartbeat path
docs(readme): document Docker Compose local development flow
test(chat): add quota enforcement coverage for SSE handler
ci(commitlint): enforce conventional commit types in pre-commit hooks
build(web): update embedded dashboard build pipeline
chore(deps): refresh Go and web package dependencies
revert(router): revert degraded-mode fallback for admin endpoints
```

## Commit Message Guidance

- Keep the subject line short and imperative.
- Prefer one logical change per commit.
- Do not use vague messages such as `wip`, `fix`, or `update`.
- Include a scope when it adds clarity, especially for cross-team repositories.

