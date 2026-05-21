# Repository Guidelines

## Project Structure & Module Organization

This repository is a multi-service backend for the graduation platform. Each service owns its source, dependencies, and runtime config:

- `api-gateway/`, `auth-service/`, `notification-service/`: TypeScript/Node services with `src/`, `tests/`, and service-local `package.json`.
- `courses-attendance-service/`, `payment-service/`, `chat-service/`, `ws-gateway/`: Go services with service-local `go.mod`.
- `recommendation-service/`: Python FastAPI service under `app/`, with routes, services, models, schemas, and utilities separated by folder.
- Root docs, Postman collections, SQL helpers, Docker files, and deployment notes live at repository root.
- `observability/` contains shared telemetry infrastructure.

## Build, Test, and Development Commands

Use commands from the service directory unless noted.

- `docker compose up --build`: build and run the full local stack from the root.
- `npm run dev`: run a Node service in watch mode.
- `npm run build`: compile TypeScript services.
- `npm test` or `npm run test:coverage`: run Jest tests for Node services.
- `go test ./...`: run tests in a Go service.
- `uvicorn app.main:app --reload --port 8095`: run `recommendation-service` locally.
- `pip install -r requirements.txt`: install Python dependencies for `recommendation-service`.

## Coding Style & Naming Conventions

Keep changes service-scoped. TypeScript services use ES modules, `src/main.ts` entrypoints, camelCase variables, and PascalCase classes/types. Go code should follow `gofmt` and idiomatic package naming. Python code uses FastAPI conventions, snake_case functions/files, and Pydantic/SQLAlchemy models in `schemas/` and `models/`.

Do not hardcode secrets or service URLs. Use `.env`, `.env.docker`, or service-local environment variables.

## Testing Guidelines

Node services use Jest and Supertest; place tests under `tests/` and name them `*.test.ts`. Run `npm test` before PRs touching TypeScript services. Go services should keep tests beside packages as `*_test.go` and run `go test ./...`. Python service has no visible test suite yet; add focused tests when changing recommendation, chatbot, or report behavior.

## Commit & Pull Request Guidelines

History mostly follows conventional prefixes such as `feat:`, `fix:`, `docs:`, and `chore:`. Keep commits scoped and imperative, for example `fix: validate recommendation cache invalidation`.

PRs should include a short summary, affected services, config or migration notes, test results, and linked issue/task when available. Include screenshots or Postman examples for API behavior changes.

## Security & Configuration Tips

Internal service calls rely on `x-internal-service-secret`; preserve that contract. Keep JWT validation centralized through auth-service. Treat `.env` files as local/deployment configuration and avoid committing new secrets, tokens, API keys, or database credentials.

<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
<!-- SPECKIT END -->
