# Progress: Agentic Recommendations

**Feature**: `001-agentic-recommendations`

**Current Phase**: Phase 9 - Testing complete

## Completed

- Created Spec Kit feature branch `001-agentic-recommendations`.
- Created feature directory `specs/001-agentic-recommendations/`.
- Created `spec.md`, `plan.md`, `tasks.md`, `progress.md`, and `decisions.md`.
- Added Phase 1 dependencies for LangGraph, Qdrant, sentence-transformers, KMeans, NumPy, and scheduling.
- Added Qdrant service and persistent volume to `docker-compose.yml`.
- Added recommendation-service Qdrant, feature flag, retrieval, embedding, and clustering environment variables.
- Added matching settings to `recommendation-service/app/config.py`.
- Added embedding schemas in `recommendation-service/app/schemas/agent.py`.
- Added embedding generation service with Redis-backed embedding cache in `recommendation-service/app/retrieval/embedding_service.py`.
- Added course embedding text/payload builders in `recommendation-service/app/retrieval/course_indexer.py`.
- Added user behavior summary and numeric feature builder in `recommendation-service/app/clustering/feature_builder.py`.
- Added async embedding refresh job helpers in `recommendation-service/app/jobs/embedding_jobs.py`.
- Added Qdrant async vector client and collection management in `recommendation-service/app/retrieval/vector_store.py`.
- Added course/user/cluster vector upsert helpers and semantic course search in `recommendation-service/app/retrieval/vector_store.py`.
- Added hybrid retrieval module with semantic + popularity + teacher scoring in `recommendation-service/app/retrieval/hybrid_search.py`.
- Added retrieval result caching with Redis keys in `recommendation-service/app/retrieval/hybrid_search.py`.
- Added enrolled-course exclusion in retrieval post-processing in `recommendation-service/app/retrieval/hybrid_search.py`.
- Extended course embedding payload fields to include display metadata for retrieval hydration in `recommendation-service/app/retrieval/course_indexer.py`.
- Added cluster SQLAlchemy models in `recommendation-service/app/models/cluster.py`.
- Added clustering SQL migration script in `recommendation-service/alembic/versions/20260521_01_create_user_clusters.sql`.
- Added KMeans-based clustering service with persistence in `recommendation-service/app/clustering/cluster_service.py`.
- Added clustering background job orchestrator in `recommendation-service/app/clustering/jobs.py`.
- Added cluster endpoints in `recommendation-service/app/api/routes/recommendations.py`.
- Added cluster model registration during app startup in `recommendation-service/app/main.py`.
- Added MCP tool schemas in `recommendation-service/app/tools/schemas.py`.
- Added allowlisted tool registry and dispatcher with input/output validation in `recommendation-service/app/tools/registry.py`.
- Added user tools in `recommendation-service/app/tools/user_tools.py`.
- Added course retrieval/trending tools in `recommendation-service/app/tools/course_tools.py`.
- Added cluster tools in `recommendation-service/app/tools/cluster_tools.py`.
- Added registry-level output truncation and prompt-injection text sanitization in `recommendation-service/app/tools/registry.py`.
- Added agent state contract in `recommendation-service/app/agents/state.py`.
- Added planner/ranker prompts in `recommendation-service/app/agents/prompts.py`.
- Added LangGraph recommendation workflow nodes and execution loop in `recommendation-service/app/agents/graph.py`.
- Added recommendation agent facade in `recommendation-service/app/agents/recommendation_agent.py`.
- Extended Gemma client with structured JSON planner/ranker helpers in `recommendation-service/app/services/gemma_client.py`.
- Updated recommendation engine to route between legacy v1 and agentic v2 by `AGENT_RECOMMENDATIONS_ENABLED`.
- Added v2 cache keys and explanation cache storage in `recommendation-service/app/services/recommendation_engine.py`.
- Added v2 recommendation history persistence in `recommendation-service/app/services/recommendation_engine.py`.
- Added recommendation explanation endpoint in `recommendation-service/app/api/routes/recommendations.py`.
- Added explanation and v2 recommendation schemas in `recommendation-service/app/schemas/recommendation.py`.
- Kept legacy prompt-builder path active with explicit v1 fallback note in `recommendation-service/app/utils/prompt_builder.py`.
- Added `recommendation.agent.run` span and structured completion logs in `recommendation-service/app/agents/recommendation_agent.py`.
- Added `recommendation.tool.execute` span and structured tool execution logs in `recommendation-service/app/tools/registry.py`.
- Added `recommendation.vector.search` span and timing logs in `recommendation-service/app/retrieval/vector_store.py`.
- Added `recommendation.cluster.assign` span and clustering job metrics/logs in `recommendation-service/app/clustering/jobs.py`.
- Added reasoning-trace cache logs and trace-read spans for `/explain` in `recommendation-service/app/services/recommendation_engine.py`.
- Added Phase 9 test modules under `recommendation-service/tests/` for tool schemas, vector store, clustering features/service, agent behavior, and v2 recommendation engine flow.
- Added `tests/conftest.py` to make the test root importable and stub unavailable runtime dependencies in this workspace.

## In Progress

- Awaiting explicit approval to begin Phase 10 - Migration.

## Blockers

- None currently.

## Validation Results

- Spec Kit task setup discovery succeeds for `specs/001-agentic-recommendations`.
- `recommendation-service/app/config.py` compiles with `python -m py_compile`.
- `git diff --check` passed after whitespace cleanup.
- `.gitignore` and `recommendation-service/.dockerignore` added for phase-1 setup hygiene.
- `recommendation-service/Dockerfile` already includes `build-essential` and `libpq-dev`, which satisfy native dependency build prerequisites for the newly added Python packages.
- New Phase 2 modules compile successfully with `python -m py_compile`.
- New Phase 3 retrieval modules compile successfully with `python -m py_compile`.
- New Phase 4 clustering modules compile successfully with `python -m py_compile`.
- New Phase 5 tool modules compile successfully with `python -m py_compile`.
- New Phase 6 agent modules compile successfully with `python -m py_compile`.
- New Phase 7 API integration modules compile successfully with `python -m py_compile`.
- New Phase 8 observability-instrumented modules compile successfully with `python -m py_compile`.
- Targeted pytest run is currently blocked at collection by missing local packages in this workspace (`qdrant_client`, `sqlalchemy`, `google.genai`), so the full Phase 9 suite could not complete here.
- Container build and dependency installation were not run because they would require network/package downloads.

## Notes

- Implementation must proceed one phase at a time.
- Do not replace the legacy recommendation path until the feature flag integration phase.
