<!--
Sync Impact Report
- Version change: 0.0.0 → 1.0.0 (MAJOR — initial ratification, all sections new)
- Added principles:
  - I. Service-Scoped Code Quality
  - II. Enforced Testing Standards
  - III. User Experience Consistency
  - IV. Performance & Scalability
  - V. Observability as a First-Class Concern
  - VI. Security-First Design
  - VII. API Contract Governance
- Added sections:
  - Technology & Architecture Constraints
  - Development Workflow & Quality Gates
  - Governance
- Templates requiring updates:
  - .specify/templates/plan-template.md — ✅ Constitution Check gates align with principles
  - .specify/templates/spec-template.md — ✅ User stories / requirements already support UX & perf criteria
  - .specify/templates/tasks-template.md — ✅ Phase structure supports test-first, observability, and security tasks
- Follow-up TODOs: none
-->

# Graduation Platform Constitution

## Core Principles

### I. Service-Scoped Code Quality

Every change MUST be scoped to a single service boundary. Cross-service
changes require explicit justification and coordinated review.

- TypeScript services (api-gateway, auth-service, notification-service) MUST
  use ES module syntax, PascalCase for classes/types, camelCase for variables,
  and maintain `src/main.ts` entrypoints.
- Go services (courses-attendance-service, payment-service, chat-service,
  ws-gateway) MUST pass `gofmt` and `go vet` with zero warnings before merge.
- Python code (recommendation-service) MUST follow FastAPI conventions with
  snake_case naming, Pydantic schemas in `schemas/`, and SQLAlchemy models in
  `models/`.
- No hardcoded secrets, service URLs, or credentials. All configuration MUST
  flow through `.env`, `.env.docker`, or service-local environment variables.
- Linting and formatting tools MUST be configured per service and enforced in
  CI. Code that fails lint checks MUST NOT be merged.

**Rationale**: A polyglot multi-service repository demands strict per-language
conventions to prevent style drift and reduce cognitive load when switching
between services.

### II. Enforced Testing Standards

Tests are the primary defense against regressions across service boundaries.
Every service MUST maintain a minimum viable test suite proportional to its
complexity.

- Node/TypeScript services MUST use Jest and Supertest. Tests live under
  `tests/` and follow the `*.test.ts` naming convention. `npm test` MUST pass
  before any PR is merged.
- Go services MUST keep tests beside their packages as `*_test.go` files.
  `go test ./...` MUST pass before any PR is merged.
- Python services MUST include focused tests for any new or modified endpoint,
  service function, or model. pytest is the standard runner.
- Integration tests MUST cover inter-service communication paths (HTTP calls
  between services, Kafka event flows, Redis cache interactions).
- Test coverage MUST NOT decrease on any PR. New features MUST include tests
  that exercise the happy path and at least one error/edge case.
- Database migrations MUST be tested against a clean schema to verify they
  apply and roll back cleanly.

**Rationale**: With eight services communicating over HTTP, Kafka, and Redis,
untested changes in one service can cascade failures across the entire
platform.

### III. User Experience Consistency

The platform serves students, teachers, and administrators. Every
user-facing behavior MUST be predictable, accessible, and consistent
across all service boundaries.

- API responses MUST follow a uniform envelope format: `{ success, data,
  error, message }` across all services.
- Error responses MUST include machine-readable error codes and
  human-readable messages. HTTP status codes MUST follow RFC 9110 semantics.
- Pagination MUST use a consistent contract (`page`, `limit`, `total`,
  `data`) across all list endpoints in every service.
- Real-time features (chat, notifications, WebSocket events) MUST degrade
  gracefully — clients MUST receive meaningful feedback when a connection
  drops or a service is temporarily unavailable.
- All user-facing timestamps MUST be returned in ISO 8601 / UTC format. The
  client is responsible for locale-specific display.

**Rationale**: Students and teachers interact with functionality spanning
multiple backend services; inconsistent API contracts create confusing
frontend behavior and increase mobile/web client complexity.

### IV. Performance & Scalability

The platform MUST handle concurrent classroom sessions, real-time chat, and
payment processing without perceptible degradation.

- API endpoints MUST respond within **200ms at p95** under normal load. Any
  endpoint exceeding this threshold MUST be profiled and optimized or marked
  with an explicit exception and documented justification.
- Database queries MUST use indexes for all WHERE, JOIN, and ORDER BY clauses
  on tables exceeding 10,000 rows. Full table scans on production data are
  prohibited.
- Redis MUST be used for caching frequently accessed, read-heavy data (user
  sessions, course metadata, QR rotation state). Cache TTLs MUST be
  explicitly set — no unbounded caches.
- Services MUST be stateless and horizontally scalable. The docker-compose
  multi-instance pattern (chat-service × 3, ws-gateway × 3,
  courses-service × 2) MUST remain viable without code changes.
- Kafka consumers MUST process messages within **5 seconds** of publication
  under normal load. Consumer lag MUST be monitored and alerted.

**Rationale**: Classroom attendance windows are time-sensitive (30-second QR
rotation), and payment callbacks are latency-critical; performance failures
directly impact core educational workflows.

### V. Observability as a First-Class Concern

Every service MUST be observable in production. Debugging production issues
without telemetry is unacceptable.

- All services MUST emit OpenTelemetry traces to the shared otel-collector.
  The `OTEL_SERVICE_NAME` and `OTEL_EXPORTER_OTLP_ENDPOINT` environment
  variables MUST be configured for every deployed service.
- Structured logging (JSON format) MUST be used in production. Log levels
  (debug, info, warn, error) MUST be used consistently: errors for
  actionable failures, warnings for degraded states, info for significant
  business events.
- Health check endpoints (`/health`) MUST be implemented on every service
  and MUST verify downstream dependencies (database, Redis, Kafka
  connectivity).
- Sentry integration MUST be configured for error tracking in all services.
  Unhandled exceptions MUST surface in Sentry within 60 seconds.

**Rationale**: With 10+ containers in the stack, distributed tracing and
structured logging are the only viable way to diagnose cross-service failures
in reasonable time.

### VI. Security-First Design

Security controls MUST be enforced at every layer — no service trusts
another implicitly.

- All inter-service HTTP calls MUST include the `x-internal-service-secret`
  header. Services MUST reject requests missing or presenting an invalid
  secret.
- JWT validation MUST be centralized through auth-service. No service may
  implement its own token parsing or validation logic outside of shared
  middleware.
- Secrets, tokens, API keys, and database credentials MUST NEVER appear in
  source code, commit history, or log output. All secrets MUST be injected
  via environment variables.
- Input validation MUST occur at the API boundary (gateway or service
  entrypoint). No raw user input may reach database queries or external
  API calls without sanitization.
- CORS origins MUST be explicitly enumerated in production — wildcard (`*`)
  origins are permitted only in development environments.

**Rationale**: The platform handles student PII, payment data (Paymob
integration), and authentication tokens; a single security lapse can
compromise the entire user base.

### VII. API Contract Governance

Service-to-service contracts are the glue of the platform. Breaking a
contract breaks downstream consumers.

- Every service MUST expose a versioned API. Breaking changes MUST increment
  the major version and provide a migration path documented in the PR.
- Inter-service request/response schemas MUST be documented (Postman
  collections, OpenAPI specs, or contract tests). Undocumented endpoints
  MUST NOT be called by other services.
- Kafka event schemas MUST be documented with topic name, payload structure,
  and producing/consuming services. Schema changes MUST be backward
  compatible or coordinated across all consumers before deployment.
- Database schema migrations that affect shared tables MUST be reviewed by
  owners of all services that query those tables.

**Rationale**: Eight services with dozens of cross-service calls and shared
Kafka topics create a high risk of silent contract breakage; governance
prevents cascading integration failures.

## Technology & Architecture Constraints

- **Languages**: TypeScript (Node.js) for gateway and auth/notification
  services; Go for courses, payments, chat, and WebSocket gateway; Python
  (FastAPI) for recommendation and AI services.
- **Data stores**: PostgreSQL 15 (primary), Redis 7 (caching/sessions),
  Kafka (event streaming via Confluent 7.5).
- **Observability**: OpenTelemetry Collector → Jaeger (traces), Prometheus
  (metrics), Loki + Promtail (logs), Grafana (dashboards).
- **External integrations**: Cloudinary (media), Paymob (payments), Resend
  (email), Sentry (error tracking), Arcjet (rate limiting).
- **Containerization**: All services MUST be deployable via
  `docker compose up --build` from the repository root. Service images MUST
  build successfully in CI before merge.
- New technology additions (languages, databases, message brokers) require a
  written justification in the PR description and approval from at least two
  maintainers.

## Development Workflow & Quality Gates

- **Branching**: Feature branches follow conventional naming
  (`feat/`, `fix/`, `docs/`, `chore/`). Direct commits to the main branch
  are prohibited.
- **Commit messages**: MUST use conventional commit prefixes (`feat:`,
  `fix:`, `docs:`, `chore:`, `refactor:`, `test:`). Commits MUST be scoped
  and imperative.
- **PR requirements**:
  1. Short summary of changes and affected services.
  2. Configuration or migration notes (if applicable).
  3. Test results (paste or CI link).
  4. Linked issue/task when available.
  5. Screenshots or Postman examples for API behavior changes.
- **CI gates**: Lint → Build → Test → Docker build. All four gates MUST pass
  before a PR can be merged.
- **Code review**: Every PR MUST be reviewed by at least one maintainer who
  is not the author. Reviews MUST verify principle compliance.

## Governance

This constitution supersedes all other development practices. When a
conflict exists between this document and any other guideline, this
constitution takes precedence.

- **Amendments**: Any principle change MUST be proposed as a PR to this file,
  include a rationale, and be approved by at least two maintainers. The
  version MUST be incremented per semantic versioning rules (see below).
- **Versioning**: MAJOR for principle removals or incompatible redefinitions;
  MINOR for new principles or material expansions; PATCH for wording
  clarifications and typo fixes.
- **Compliance review**: PRs MUST be checked against the applicable
  principles. Reviewers MUST cite the relevant principle number when
  requesting changes for compliance reasons.
- **Runtime guidance**: See [AGENTS.md](file:///d:/Graduation/AGENTS.md) for
  day-to-day development commands and conventions.

**Version**: 1.0.0 | **Ratified**: 2025-05-21 | **Last Amended**: 2025-05-21
