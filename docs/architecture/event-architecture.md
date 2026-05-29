# Gateway Event Architecture

NexusAI-Gateway uses NATS JetStream as the default event platform for runtime events. This document defines architecture alignment only and does not add W2 product features.

## Gateway Subjects

- `tenant.{id}.agent.*` for agent request lifecycle coordination.
- `tenant.{id}.model.*` for provider routing, model request, model response, and provider failure events.
- `tenant.{id}.knowledge.*` when Gateway requests retrieval or receives knowledge context.
- `tenant.{id}.skill.*` when Gateway coordinates approved skill execution through NexusAI-Skills.

## Event Envelope

Gateway events must include event ID, event type, tenant ID, source service, correlation ID, schema version, timestamp, and sanitized payload. Events must never include raw API keys, provider secrets, or unredacted sensitive prompts by default.

## Why This Exists

Gateway is the ingress and policy boundary. Publishing tenant-scoped events with consistent envelopes lets downstream services observe runtime behavior without coupling to Gateway internals.
