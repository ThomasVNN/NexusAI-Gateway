# Contributing to NexusAI-Gateway

Thank you for helping improve NexusAI-Gateway. This repository is part of a larger microservices ecosystem, so changes should stay focused, observable, and production-oriented.

## Before You Start

- Read `README.md` for the project overview.
- Review `docs/standards/branch-strategy.md` for branch flow expectations.
- Review `docs/standards/conventional-commits.md` before writing commit messages.
- Review `docs/standards/pull-request-guidelines.md` before opening a PR.

## Local Validation

Run the repository checks that apply to your change before opening a pull request:

```bash
go test ./...
make lint
cd web && npm run build
```

If your change affects the embedded dashboard or static assets, rebuild the frontend bundle so the Go binary can serve the updated output.

## Branching

- Create work from `develop` unless you are fixing an urgent production issue.
- Use `feature/*`, `fix/*`, `hotfix/*`, or `release/*` branch names as documented in the branch strategy guide.
- Keep PRs small and focused whenever possible.

## Commit Messages

- Use Conventional Commits for every commit.
- Prefer a meaningful scope such as `auth`, `router`, `mcp`, `storage`, or `web`.
- Include `!` or `BREAKING CHANGE:` when a change is not backward compatible.

## Pull Requests

- Link the issue or change request that the PR addresses.
- Include testing evidence in the PR description.
- Call out any security, deployment, or migration implications explicitly.
- Use the repository PR template so reviewers receive consistent context.

## Security Expectations

- Do not commit secrets, private keys, or production credentials.
- Use `.env.example` as the source of truth for required environment variables.
- Treat changes to authentication, quota enforcement, logging, and request routing as security-sensitive.

## Review Expectations

- Maintain the current runtime behavior unless the change is explicitly intended to alter it.
- Prefer incremental refactors over broad structural rewrites.
- Add or update tests when behavior changes.

If you are unsure how a change should be handled, open a draft PR early and ask for review before expanding the scope.

