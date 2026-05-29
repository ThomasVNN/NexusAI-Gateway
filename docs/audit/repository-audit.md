# Repository Audit: NexusAI-Gateway

Audit date: 2026-05-28

This audit captures the repository state that informed the standardization work in this task. Some findings were remediated in the same change set after the audit was written.

## Tech Stack Summary

- Go 1.22.0 gateway service using the standard `net/http` stack.
- PostgreSQL persistence via `database/sql` and `github.com/lib/pq`.
- Embedded React 18 dashboard built with Vite and TypeScript.
- Docker multi-stage build and Docker Compose-based local environment.
- Lightweight shell-based developer scripts and `golangci-lint` for Go linting.

## Architecture Summary

NexusAI-Gateway is a layered gateway service with a Go entrypoint in `cmd/gateway`, domain contracts in `internal/domain`, HTTP ingress and MCP handling in `internal/gateway`, persistence in `internal/storage`, and PII handling in `internal/privacy`.

The service supports:

- API key authentication and hashing.
- Quota-aware chat completion streaming.
- MCP SSE and JSON-RPC endpoints.
- Admin APIs for keys, providers, logs, usage, and model catalog compatibility.
- Embedded static dashboard delivery from `web/dist` via `embed.FS`.
- In-memory fallback stores when database connectivity is unavailable.

## Existing CI/CD Summary

- No GitHub Actions workflows are present in the repository.
- No release automation, dependency update automation, or security scanning workflows are defined.
- Build and test automation exists only as local commands in `Makefile` and `tests/run_e2e_tests.sh`.
- Docker and Docker Compose assets are present for local execution and image building.

## Existing Standards Summary

- The repository has a development guide in `DEVELOPMENT.md`, but it does not fully align with the required NexusAI branch, commit, and PR standards.
- `golangci-lint` is configured for Go linting.
- The code base already uses layered packages and a DDD-lite split between domain, storage, and gateway layers.
- A conventional commit policy is mentioned, but it is not enforced by hooks or commitlint.

## Risks And Inconsistencies

1. Sensitive values are present in defaults and deployment examples, including database credentials and bootstrap passwords.
2. `cmd/gateway/main.go` logs the full database URL, which can expose credentials in logs.
3. The repository has no enforced commit validation, so branch and commit conventions are advisory only.
4. The top-level `.gitignore` is incomplete for this stack and does not fully exclude `web/node_modules/`, `web/dist/`, or other local build artifacts.
5. No top-level `CODEOWNERS`, issue templates, or pull request template exist to support multi-team review.
6. The current `DEVELOPMENT.md` branch naming examples differ from the required `main`, `staging`, `develop`, `feature/*`, `fix/*`, `hotfix/*`, and `release/*` flow.
7. The current worktree includes generated frontend assets and local dependency directories, which increases the risk of accidental commits if ignore rules are not tightened.
8. The embedded dashboard and backend have minimal observability scaffolding beyond standard logging.

## Missing Governance Files

- `.github/PULL_REQUEST_TEMPLATE.md`
- `.github/ISSUE_TEMPLATE/bug_report.md`
- `.github/ISSUE_TEMPLATE/feature_request.md`
- `.github/ISSUE_TEMPLATE/question.md`
- `CODEOWNERS`
- `CONTRIBUTING.md`
- `CODE_OF_CONDUCT.md`
- `SECURITY.md`
- `CHANGELOG.md`
- `.editorconfig`
- `.gitattributes`
- `.env.example`
- commit enforcement configuration for commitlint, husky, and lint-staged

## Missing Documentation

- `docs/architecture/overview.md`
- `docs/standards/branch-strategy.md`
- `docs/standards/conventional-commits.md`
- `docs/standards/pull-request-guidelines.md`
- A more complete README covering architecture, setup, testing, deployment, and troubleshooting

## Developer Experience Gaps

- No automated pre-commit formatting or commit validation.
- No root-level formatting or editor standardization files.
- No explicit environment template for contributors.
- No issue or PR templates to keep contribution intake consistent.
- No ownership routing for code reviews across gateway, storage, privacy, and web domains.

## Suggested Improvements

1. Add governance docs and templates so contribution and review expectations are explicit.
2. Enforce conventional commits through husky and commitlint.
3. Tighten ignore rules to keep generated artifacts and local dependencies out of the repository.
4. Add `.env.example` and security guidance to make environment setup explicit and safer.
5. Align branch strategy documentation with the required `main`, `staging`, `develop`, and work branch flow.
6. Introduce a baseline GitHub Actions pipeline for linting, tests, and dependency checks in a later change.
7. Add structured observability support in a future service improvement phase, including request IDs, metrics, and tracing propagation.
