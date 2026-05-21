# Recommendation Service

This service powers AI-assisted course recommendations, chatbot support, report generation, and recommendation analytics for the graduation platform.

## What it does

- Returns personalized course recommendations for an authenticated student.
- Supports two recommendation modes:
  - `legacy`: prompt-based AI recommendations using course and user profile data.
  - `agentic v2`: tool-driven recommendations using LangGraph, Redis cache, and internal tools.
- Provides trending recommendations for all users.
- Stores recommendation history and clustering data in the database.
- Exposes health checks and observability hooks for tracing, metrics, and error reporting.

## Main request flow

1. A client calls `/api/v1/recommendations/me`.
2. The route gets the authenticated user from the auth layer.
3. `app.services.recommendation_engine.get_personalized_recommendations()` decides which mode to use.
4. The service checks Redis cache first.
5. If there is no cache hit, it fetches user analytics and course catalog data from the courses service.
6. It builds the recommendation prompt or runs the agentic graph.
7. It ranks or filters the results.
8. It enriches the response with course details and writes the result back to Redis.
9. It persists recommendation history when possible.

## Legacy mode

The legacy path is the simpler pipeline:

- Fetch user analytics from the courses service.
- Fetch the full course list.
- Remove already-enrolled courses.
- Build a detailed prompt with `app.utils.prompt_builder`.
- Send the prompt to the AI model through `app.services.gemma_client`.
- Hydrate the returned course IDs with real course metadata.
- Cache the final list in Redis.

This path is the default when `AGENT_RECOMMENDATIONS_ENABLED=false`.

## Agentic v2 mode

The v2 path is enabled when `AGENT_RECOMMENDATIONS_ENABLED=true`.

It uses:

- `app.agents.graph` to orchestrate the workflow.
- `app.agents.recommendation_agent` as the entry point.
- `app.tools.registry` to expose allowlisted tools.
- `app.retrieval.vector_store` and embedding services for semantic retrieval.
- `app.clustering.cluster_service` and cluster jobs for user grouping.
- Redis to cache both recommendations and explanation traces.

The agentic flow is more structured:

- It gathers context through tools instead of relying on one large prompt.
- It can rank candidate courses.
- It stores a reasoning trace so the API can explain why a recommendation was produced.

## Key modules

- `app/main.py`: creates the FastAPI app, mounts routers, health checks, and startup table creation.
- `app/config.py`: central settings loaded from environment variables.
- `app/services/recommendation_engine.py`: main recommendation orchestration and caching.
- `app/services/gemma_client.py`: AI model client wrapper.
- `app/services/course_client.py`: internal course-service client.
- `app/retrieval/*`: embeddings, hybrid search, vector search, and course indexing.
- `app/clustering/*`: user clustering and cluster refresh jobs.
- `app/agents/*`: LangGraph state, prompts, and recommendation orchestration.
- `app/tools/*`: tool registry and tool implementations.
- `app/models/*`: SQLAlchemy models and database setup.
- `app/api/routes/*`: HTTP endpoints for recommendations, chat, and reports.

## API surface

- `GET /health`
  - Checks Postgres, Redis, and vector DB connectivity.
- `GET /api/v1/recommendations/me`
  - Returns personalized recommendations for the current user.
- `GET /api/v1/recommendations/explain`
  - Returns the last v2 reasoning summary and tool trace.
- `POST /api/v1/recommendations/refresh`
  - Clears cache and regenerates recommendations.
- `GET /api/v1/recommendations/trending`
  - Returns globally trending courses.

## Dependencies

The service expects these runtime dependencies:

- Postgres for persistence.
- Redis for caching.
- Qdrant for vector search.
- Courses service for user analytics and course catalog data.
- Auth service for authentication.
- AI model access for ranking and generation.

## Docker setup

`recommendation-service/Dockerfile` builds a Python 3.11 image, installs dependencies, copies the service code, and starts:

```bash
uvicorn app.main:app --host 0.0.0.0 --port 8095
```

In `docker-compose.yml`, the service is wired to:

- `postgres`
- `redis`
- `qdrant`
- `courses-service-1`

## Environment variables

Important variables include:

- `DATABASE_URL`
- `REDIS_URL`
- `VECTOR_DB_URL`
- `AI_API_KEY`
- `AI_MODEL`
- `INTERNAL_SERVICE_SECRET`
- `COURSES_SERVICE_URL`
- `AUTH_SERVICE_URL`
- `NOTIFICATION_SERVICE_URL`
- `AGENT_RECOMMENDATIONS_ENABLED`

## Behavior notes

- The service caches recommendation responses to reduce repeated AI calls.
- The v2 path also caches an explanation object for the UI.
- Startup creates the SQLAlchemy models if they are missing.
- Health checks can return `degraded` if one dependency is down.
- The code is designed to keep recommendation logic separate from transport logic.

## Local run

From `recommendation-service/`:

```bash
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8095
```

Or from the repo root:

```bash
docker compose up --build recommendation-service
```

## Tests

The service includes tests for:

- recommendation engine behavior
- agentic recommendation flow
- tool security and schema validation
- embedding jobs
- clustering service
- vector store integration contracts
- API response format

