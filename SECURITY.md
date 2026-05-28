# Security Policy

## Supported Versions

Security fixes are applied to the current active development line and any release branches that are explicitly maintained for NexusAI-Gateway.

## Reporting a Vulnerability

If you discover a security issue, do not open a public issue with exploit details.

Preferred reporting path:

1. Notify the repository maintainers through the private security channel used by the NexusAI project.
2. Include a concise summary, affected paths, reproduction steps, and any proof-of-concept material needed to validate the issue.
3. Avoid sharing secrets, tokens, or sensitive data in plain text unless they are required for reproduction and have been minimized.

## Security Expectations

- Never commit secrets, API keys, database credentials, or private certificates.
- Keep `.env.example` aligned with runtime requirements so environment setup stays explicit.
- Review changes to authentication, authorization, quota enforcement, logging, and request routing carefully.
- Prefer least-privilege credentials for local, staging, and production environments.
- Validate any dependency or container image updates for security impact before merging.

## Operational Notes

- Production deployments should terminate TLS before traffic reaches the gateway unless a specific deployment profile states otherwise.
- Request logging should avoid secrets, bearer tokens, and full provider credentials.
- Database and provider URLs should be treated as sensitive configuration even when they are stored in environment variables.

