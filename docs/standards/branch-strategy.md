# Branch Strategy

This repository follows a long-lived branch model designed for multi-team collaboration and controlled releases.

## Branch Roles

- `main` - production-ready code only.
- `staging` - release candidate branch used for final validation before production.
- `develop` - integration branch for completed feature work.
- `feature/*` - new capabilities, experiments, or scoped improvements.
- `fix/*` - non-urgent bug fixes that are not production incidents.
- `hotfix/*` - urgent fixes branched from `main` to address production issues.
- `release/*` - stabilization branch used to prepare a release cut.

## Merge Flow

1. Create feature work from `develop`.
2. Merge reviewed feature PRs back into `develop`.
3. Promote `develop` into `staging` when the release set is ready.
4. Promote `staging` into `main` after release validation completes.
5. Merge `main` back into `develop` after production release so the integration branch stays aligned.

## Release Flow

1. Cut a `release/*` branch from `develop` when the release scope is frozen.
2. Apply only stabilization changes on the release branch.
3. Merge the release branch into `staging` for final verification.
4. Merge the validated release into `main` and tag the release if the repository uses version tags.
5. Merge `main` back into `develop` to keep release fixes synchronized.

## Hotfix Flow

1. Create `hotfix/*` from `main` when a production incident requires an immediate fix.
2. Keep the fix minimal and focused on the production issue.
3. Merge the hotfix into `main` after validation.
4. Back-merge the hotfix into `develop` and `staging` so all long-lived branches receive the fix.

## Pull Request Expectations

- Use the pull request template.
- Link the issue or change request that motivated the change.
- Keep PRs small enough for fast review whenever possible.
- Include validation evidence for code, configuration, or documentation changes.
- Call out security, deployment, or schema impact explicitly.

## Protection Guidance

- Protect `main`, `staging`, and `develop` with required review and status checks.
- Require conventional commits for commit history hygiene.
- Disallow direct pushes to protected branches except for controlled automation or maintainers.

