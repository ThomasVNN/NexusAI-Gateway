# NexusAI-Gateway Development Guidelines

This document outlines the coding standards, repository guidelines, Git workflow, and SDLC conventions adopted by the NexusAI-Gateway development team.

> Note: the authoritative branch, commit, and pull request standards now live in `docs/standards/`. This file keeps the historical developer guidance that remains relevant to day-to-day coding.

## Coding Standards

### Go Code Style

- **Formatting**: All Go files must conform to the standard formatting. Always run `go fmt ./...` before committing.
- **Linting**: We enforce strict linting rules using `golangci-lint`. Ensure you pass `make lint` locally before making a pull request.
- **Error Handling**: Never ignore returned errors. Always handle errors explicitly or return them decorated with meaningful context:
  ```go
  if err != nil {
      return fmt.Errorf("failed to load active key quota: %w", err)
  }
  ```
- **Concurrency**: Keep goroutine lifecycles controlled. Always use contexts to handle cancellations and timeouts gracefully.
- **Connection Pools**: Maintain single connection pool instances (e.g., PostgreSQL connection pools) and pass them as dependencies, rather than recreating them per request.

## Git Commit Standards (Conventional Commits)

We enforce the **Conventional Commits** specification for all commit messages. This helps generate clean changelogs automatically.

Format: `<type>(<scope>): <description>`

### Allowed Types

- `feat`: A new feature (e.g., `feat(mcp): add SSE handler for tool stream`)
- `fix`: A bug fix (e.g., `fix(postgres): fix query syntax error in quota count`)
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A performance improvement
- `docs`: Documentation changes only (e.g., `docs: update README with postgres instructions`)
- `test`: Adding missing tests or correcting existing tests
- `ci`: Changes to CI or automation
- `build`: Changes to build tooling or dependencies
- `chore`: Changes to the build process, auxiliary tools, or library dependencies

## Branching & Pull Request Process

1. **Branch Naming Policy**:
   - `main`: production-ready branch
   - `staging`: release candidate branch
   - `develop`: integration branch for completed work
   - Features: `feature/short-description` (e.g., `feature/postgres-migration`)
   - Fixes: `fix/short-description` (e.g., `fix/sse-disconnection`)
   - Hotfixes: `hotfix/short-description` (e.g., `hotfix/auth-token-expiry`)
   - Releases: `release/version-or-scope` (e.g., `release/2026.05.28`)
2. **Review Gate**:
   - Every Pull Request requires at least one peer approval.
   - All unit tests and static linters must pass in CI before merging.
3. **Commit Cleanliness**:
   - Squash commits into logical units before merging to the `main` branch.
   - Avoid leaving generic messages like `temp commit`, `fix bug`, or `wip`.
