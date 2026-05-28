# NexusAI-Gateway Development Guidelines

This document outlines the coding standards, repository guidelines, Git workflow, and SDLC conventions adopted by the NexusAI-Gateway development team.

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
- `docs`: Documentation changes only (e.g., `docs: update README with postgres instructions`)
- `style`: Changes that do not affect the meaning of the code (formatting, missing semi-colons, etc.)
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process, auxiliary tools, or library dependencies

## Branching & Pull Request Process

1. **Branch Naming Policy**:
   - Features: `feature/short-description` (e.g., `feature/postgres-migration`)
   - Bugfixes: `bugfix/short-description` (e.g., `bugfix/sse-disconnection`)
   - Improvements: `improvement/short-description`
2. **Review Gate**:
   - Every Pull Request requires at least one peer approval.
   - All unit tests and static linters must pass in CI before merging.
3. **Commit Cleanliness**:
   - Squash commits into logical units before merging to the `main` branch.
   - Avoid leaving generic messages like `temp commit`, `fix bug`, or `wip`.
