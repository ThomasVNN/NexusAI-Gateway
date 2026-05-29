# Pull Request Guidelines

Pull requests are the primary governance point for NexusAI-Gateway changes.

## Expectations

- Keep PRs small and focused.
- Link the related issue, incident, or change request.
- Include tests or validation steps for all behavior changes.
- Call out security implications for authentication, authorization, quota, logging, and routing changes.
- Call out migration impact when schemas, configuration, or persistent data shape changes are involved.
- Call out breaking changes explicitly and describe the compatibility path.

## Recommended Review Checklist

- [ ] The change is scoped to a single concern.
- [ ] The linked issue or request is included.
- [ ] Tests were added or updated where appropriate.
- [ ] Documentation was updated where needed.
- [ ] Security implications were reviewed.
- [ ] Deployment impact was reviewed.
- [ ] Rollback or mitigation steps are clear.

## Review Hygiene

- Prefer draft PRs for larger changes so reviewers can see direction early.
- Avoid bundling unrelated refactors with feature work.
- Keep generated assets and dependency updates separate when practical.
- Add screenshots or example requests when the change affects the embedded dashboard or public API behavior.

