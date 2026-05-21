# Implementation Plan: Agentic Recommendations

## Overview

Evolve `recommendation-service` incrementally from the current `Analytics -> Prompt Builder -> Gemma -> Recommendations` flow into `RecommendationAgent -> MCP Tool Registry -> RAG Retrieval + Clustering + Analytics -> LLM Reasoning + Ranking -> Recommendations`.

The existing FastAPI app, Redis cache, `course_client`, `recommendation_engine.py`, and `gemma_client.py` stay in place. The new path is introduced behind `AGENT_RECOMMENDATIONS_ENABLED` and uses separate `v2` cache keys.

## Architecture

Runtime request path:

1. `GET /api/v1/recommendations` enters the existing route.
2. `recommendation_engine.get_personalized_recommendations()` checks feature flag and cache.
3. If enabled, it delegates to `RecommendationAgent`.
4. LangGraph runs a bounded reasoning loop using allowlisted tools.
5. Tools retrieve analytics, cluster data, similar users, semantic course matches, cart activity, and trending courses.
6. Retrieved candidates are hybrid-ranked before LLM reranking.
7. The LLM receives only compact retrieved context, not the full catalog.
8. Output is validated, hydrated, cached, traced, and returned.
9. Public recommendation and cluster endpoints use a shared `{ success, data, error, message }` envelope with machine-readable error codes.

Fallback behavior:

- If the feature flag is off, use the legacy path.
- If Qdrant retrieval fails, use trending and analytics-derived candidates.
- If clustering is missing, continue without cluster contribution.
- If LLM ranking fails, use deterministic hybrid ranking.

## New Module Layout

```text
recommendation-service/app/
  agents/
    state.py
    prompts.py
    graph.py
    recommendation_agent.py
  tools/
    schemas.py
    registry.py
    user_tools.py
    course_tools.py
    cluster_tools.py
  retrieval/
    embedding_service.py
    vector_store.py
    course_indexer.py
    hybrid_search.py
  clustering/
    feature_builder.py
    cluster_service.py
    jobs.py
  jobs/
    scheduler.py
    embedding_jobs.py
    clustering_jobs.py
```

## Technology Choices

- Agent orchestration: LangGraph
- Tool contracts: Pydantic JSON-schema models
- Vector DB: Qdrant
- Embeddings: sentence-transformers with `BAAI/bge-small-en-v1.5`
- Clustering: scikit-learn KMeans
- Cache: Redis
- Tracing: existing OpenTelemetry setup extended with custom spans
- LLM: existing Gemma client extended for structured planning and ranking

## Qdrant Design

Collections:

- `courses`: course embeddings with safe metadata and popularity signals
- `users`: behavior-summary embeddings
- `clusters`: cluster centroid embeddings and metadata

Payload fields:

- `entity_type`
- `entity_id`
- `title`
- `subject`
- `categories`
- `popularity_score`
- `teacher_score`
- `tags`
- `updated_at`

Indexing strategy:

- Course embeddings refresh after catalog changes or scheduled refresh.
- User embeddings refresh after analytics/cart/history changes.
- Cluster embeddings refresh after clustering jobs.
- Retrieval uses top-k semantic search plus metadata filters and hybrid score blending.

## Embedding Pipeline

Embedding service responsibilities:

- Load `BAAI/bge-small-en-v1.5` once per process.
- Normalize input text.
- Cache text-hash embeddings in Redis.
- Batch encode course and user summaries.
- Bound max input length before model execution.

Course embedding text includes title, subject, categories, description, teacher, and popularity hints.

User embedding text includes interests, watched subjects, preview interests, cart subjects, completion tendency, and engagement summary.

## Clustering Flow

1. Fetch eligible user analytics profiles.
2. Build normalized behavioral features:
   - viewed/previewed courses
   - purchased/enrolled courses
   - categories
   - cart subjects
   - watch time
   - completion rate
   - engagement score
3. Combine numeric features with behavior embeddings.
4. Fit KMeans using configured cluster count.
5. Persist assignments in `user_clusters`.
6. Persist metadata in `cluster_metadata`.
7. Upsert cluster vectors into Qdrant.

## Tool Registry Design

Each tool has:

- `name`
- `description`
- Pydantic input model
- Pydantic output model
- async callable

Required tools:

- `get_user_profile`
- `get_user_cluster`
- `get_similar_users`
- `search_relevant_courses`
- `get_user_history`
- `get_trending_courses`
- `get_recent_cart_activity`
- `get_cluster_top_courses`

Only registered tools can execute. The LLM can request tools by name and JSON args, but code validates and dispatches all calls.

## LangGraph Orchestration

State:

- `user_id`
- `user_profile`
- `cluster`
- `similar_users`
- `retrieved_courses`
- `trending_courses`
- `candidate_courses`
- `tool_trace`
- `reasoning_summary`
- `recommendations`
- `errors`

Nodes:

- `plan_next_tool`
- `execute_tool`
- `merge_tool_result`
- `should_continue`
- `rank_candidates`
- `validate_output`
- `fallback_ranker`

Limits:

- Max 8 tool calls.
- Top 20 candidate courses into ranking.
- Top 6 final recommendations.
- Tool output truncation before model context.

## API Integration

Existing:

- `GET /api/v1/recommendations`
- `POST /api/v1/recommendations/refresh`
- `GET /api/v1/recommendations/trending`

Add:

- `GET /api/v1/recommendations/explain`
- `GET /api/v1/recommendations/clusters/{user_id}`
- `GET /api/v1/recommendations/clusters/{cluster_id}/top-courses`

## Caching Strategy

- Legacy recommendations: keep `recommendation:v1:{user_id}`.
- Agentic recommendations: use `recommendation:v2:{user_id}`.
- Retrieval cache: `retrieval:v1:{user_id}:{query_hash}`.
- Embedding cache: `embedding:v1:{model}:{text_hash}`.
- Cluster cache: `cluster:v1:{user_id}`.

## Observability Strategy

Add spans:

- `recommendation.agent.run`
- `recommendation.tool.execute`
- `recommendation.vector.search`
- `recommendation.embedding.generate`
- `recommendation.cluster.assign`
- `recommendation.llm.rank`

Structured logs include user ID, request ID, tool name, duration, result counts, cache hit, cluster ID, and fallback reason.

Health behavior:

- `/health` verifies database, Redis, and Qdrant connectivity and returns degraded status if any dependency fails.
- `/debug-sentry` remains available for error-tracing validation in non-production environments.

## Migration Strategy

1. Add infrastructure and configuration.
2. Add embeddings and vector indexing without changing public behavior.
3. Add retrieval and clustering APIs.
4. Add MCP tools and tests.
5. Add LangGraph agent behind `AGENT_RECOMMENDATIONS_ENABLED`.
6. Switch recommendation engine to route by feature flag.
7. Validate v1/v2 coexistence.
8. Gradually enable and monitor.
9. Keep rollback by disabling the flag.
10. Document the 200ms p95 exception for fresh agentic generation and preserve the shared response envelope across all recommendation-family APIs.

## Phase 1 Scope

Phase 1 only adds infrastructure:

- Dependencies in `requirements.txt`
- Qdrant service in `docker-compose.yml`
- Environment/config settings in `app/config.py`
- Feature flag defaults
- No agent logic, retrieval logic, clustering logic, or route changes yet
