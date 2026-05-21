# Tasks: Agentic Recommendations

**Input**: Design documents from `specs/001-agentic-recommendations/`

**Prerequisites**: `plan.md`, `spec.md`

**Tests**: Include focused tests for new services and migration behavior.

**Organization**: Tasks are grouped by required implementation phase and user-story value.

## Phase 1: Infrastructure

**Purpose**: Add safe runtime prerequisites without changing recommendation behavior.

- [X] T001 Add agentic recommendation dependencies to `recommendation-service/requirements.txt`
- [X] T002 Add Qdrant service and `qdrant_data` volume to `docker-compose.yml`
- [X] T003 Add Qdrant dependency and environment variables to `recommendation-service` in `docker-compose.yml`
- [X] T004 Add agent, embedding, vector, clustering, retrieval, and cache settings to `recommendation-service/app/config.py`
- [X] T005 Verify `recommendation-service/Dockerfile` can install sentence-transformers and scikit-learn dependencies
- [X] T006 Update `specs/001-agentic-recommendations/progress.md` with Phase 1 validation results

## Phase 2: Embeddings

**Purpose**: Build reusable embedding generation and refresh foundations.

- [X] T007 [P] [US1] Create embedding schemas in `recommendation-service/app/schemas/agent.py`
- [X] T008 [P] [US1] Create embedding service in `recommendation-service/app/retrieval/embedding_service.py`
- [X] T009 [US1] Add Redis-backed embedding cache in `recommendation-service/app/retrieval/embedding_service.py`
- [X] T010 [P] [US1] Create course embedding text builder in `recommendation-service/app/retrieval/course_indexer.py`
- [X] T011 [P] [US3] Create user behavior summary builder in `recommendation-service/app/clustering/feature_builder.py`
- [X] T012 [US1] Create async embedding refresh job in `recommendation-service/app/jobs/embedding_jobs.py`

## Phase 3: Retrieval

**Purpose**: Add vector search and hybrid candidate ranking.

- [X] T013 [P] [US1] Create Qdrant vector store client in `recommendation-service/app/retrieval/vector_store.py`
- [X] T014 [US1] Add Qdrant collection initialization for courses, users, and clusters in `recommendation-service/app/retrieval/vector_store.py`
- [X] T015 [US1] Implement course upsert and search methods in `recommendation-service/app/retrieval/vector_store.py`
- [X] T016 [P] [US1] Implement hybrid scoring in `recommendation-service/app/retrieval/hybrid_search.py`
- [X] T017 [US1] Add retrieval cache keys and cache handling in `recommendation-service/app/retrieval/hybrid_search.py`
- [X] T018 [US1] Add enrolled-course exclusion to retrieval candidate merging in `recommendation-service/app/retrieval/hybrid_search.py`

## Phase 4: Clustering

**Purpose**: Persist user clusters and expose cluster behavior signals.

- [X] T019 [P] [US3] Create cluster SQLAlchemy models in `recommendation-service/app/models/cluster.py`
- [X] T020 [US3] Create Alembic migration for `user_clusters` and `cluster_metadata` in `recommendation-service/alembic/versions/`
- [X] T021 [P] [US3] Implement behavioral feature extraction in `recommendation-service/app/clustering/feature_builder.py`
- [X] T022 [US3] Implement KMeans clustering pipeline in `recommendation-service/app/clustering/cluster_service.py`
- [X] T023 [US3] Persist cluster assignments and metadata in `recommendation-service/app/clustering/cluster_service.py`
- [X] T024 [US3] Upsert cluster vectors after clustering in `recommendation-service/app/clustering/jobs.py`
- [X] T025 [US3] Add cluster routes in `recommendation-service/app/api/routes/recommendations.py`

## Phase 5: MCP Tools

**Purpose**: Add validated, allowlisted tool execution for the agent.

- [X] T026 [P] [US1] Create tool input and output schemas in `recommendation-service/app/tools/schemas.py`
- [X] T027 [US1] Implement tool registry and dispatcher in `recommendation-service/app/tools/registry.py`
- [X] T028 [P] [US1] Implement user profile, history, and cart tools in `recommendation-service/app/tools/user_tools.py`
- [X] T029 [P] [US1] Implement relevant course search and trending tools in `recommendation-service/app/tools/course_tools.py`
- [X] T030 [P] [US3] Implement cluster lookup, similar users, and cluster top course tools in `recommendation-service/app/tools/cluster_tools.py`
- [X] T031 [US1] Add output truncation and sanitization in `recommendation-service/app/tools/registry.py`

## Phase 6: LangGraph Agent

**Purpose**: Add bounded reasoning loop and structured ranking.

- [X] T032 [P] [US1] Create `RecommendationState` in `recommendation-service/app/agents/state.py`
- [X] T033 [P] [US1] Create agent prompts in `recommendation-service/app/agents/prompts.py`
- [X] T034 [US1] Extend `recommendation-service/app/services/gemma_client.py` with structured JSON planning and ranking helpers
- [X] T035 [US1] Implement LangGraph nodes in `recommendation-service/app/agents/graph.py`
- [X] T036 [US1] Implement tool planning and execution loop in `recommendation-service/app/agents/graph.py`
- [X] T037 [US1] Implement LLM ranking node in `recommendation-service/app/agents/graph.py`
- [X] T038 [US1] Implement validation and fallback nodes in `recommendation-service/app/agents/graph.py`
- [X] T039 [US1] Create `RecommendationAgent` facade in `recommendation-service/app/agents/recommendation_agent.py`

## Phase 7: API Integration

**Purpose**: Route public recommendation APIs through the new path without breaking legacy behavior.

- [X] T040 [US4] Update `recommendation-service/app/services/recommendation_engine.py` to choose legacy or agentic path by `AGENT_RECOMMENDATIONS_ENABLED`
- [X] T041 [US4] Add `recommendation:v2:{user_id}` cache usage in `recommendation-service/app/services/recommendation_engine.py`
- [X] T042 [US2] Add explanation response schemas in `recommendation-service/app/schemas/recommendation.py`
- [X] T043 [US2] Add `GET /explain` route in `recommendation-service/app/api/routes/recommendations.py`
- [X] T044 [US1] Persist v2 recommendation history and source metadata in `recommendation-service/app/services/recommendation_engine.py`
- [X] T045 [US4] Keep legacy prompt builder available until rollout is complete in `recommendation-service/app/utils/prompt_builder.py`

## Phase 8: Observability

**Purpose**: Make agent behavior debuggable and measurable.

- [ ] T046 [P] [US1] Add agent run spans in `recommendation-service/app/agents/recommendation_agent.py`
- [ ] T047 [P] [US1] Add tool execution spans and structured logs in `recommendation-service/app/tools/registry.py`
- [ ] T048 [P] [US1] Add vector search timing logs in `recommendation-service/app/retrieval/vector_store.py`
- [ ] T049 [P] [US3] Add clustering job metrics and logs in `recommendation-service/app/clustering/jobs.py`
- [ ] T050 [US2] Add reasoning trace storage for explanation endpoint in `recommendation-service/app/services/recommendation_engine.py`

## Phase 9: Testing

**Purpose**: Validate tools, retrieval, clustering, agent flow, cache behavior, and migration safety.

- [ ] T051 [P] [US1] Add tool schema validation tests in `recommendation-service/tests/test_tool_schemas.py`
- [ ] T052 [P] [US1] Add vector store tests with mocked Qdrant in `recommendation-service/tests/test_vector_store.py`
- [ ] T053 [P] [US3] Add clustering feature builder tests in `recommendation-service/tests/test_cluster_features.py`
- [ ] T054 [P] [US3] Add cluster persistence tests in `recommendation-service/tests/test_cluster_service.py`
- [ ] T055 [P] [US1] Add agent loop tests with mocked tools and LLM in `recommendation-service/tests/test_recommendation_agent.py`
- [ ] T056 [P] [US4] Add cache and feature-flag tests in `recommendation-service/tests/test_recommendation_engine_v2.py`
- [ ] T057 [US1] Run recommendation-service test suite with `pytest` from `recommendation-service/`

## Phase 10: Migration

**Purpose**: Roll out safely with v1/v2 coexistence and rollback.

- [ ] T058 [US4] Add deployment notes for `AGENT_RECOMMENDATIONS_ENABLED` in `specs/001-agentic-recommendations/decisions.md`
- [ ] T059 [US4] Validate v1 path with feature flag disabled using existing recommendation endpoint
- [ ] T060 [US4] Validate v2 path with feature flag enabled using indexed course data
- [ ] T061 [US4] Validate rollback by disabling feature flag and confirming v1 cache path still works
- [ ] T062 [US4] Update `specs/001-agentic-recommendations/progress.md` with rollout validation results

## Dependencies & Execution Order

- Phase 1 blocks all later phases.
- Phase 2 blocks semantic retrieval and clustering quality.
- Phase 3 depends on Phase 2.
- Phase 4 depends on Phase 2 and partially on Phase 3 for cluster vector upserts.
- Phase 5 depends on Phases 3 and 4.
- Phase 6 depends on Phase 5.
- Phase 7 depends on Phase 6.
- Phase 8 can begin after each subsystem exists.
- Phase 9 should be added alongside each phase, with final full validation before migration.
- Phase 10 depends on Phases 1-9.

## Parallel Opportunities

- Phase 2 embedding service and text builders can run in parallel.
- Phase 4 models and feature extraction can run in parallel.
- Phase 5 user, course, and cluster tools can run in parallel after registry schemas are stable.
- Phase 8 observability tasks can run in parallel by subsystem.
- Phase 9 tests can run in parallel by test file.

## Implementation Strategy

1. Complete Phase 1 and verify configuration only.
2. Build embeddings and retrieval without changing public recommendation behavior.
3. Add clustering and APIs.
4. Add MCP tools and LangGraph agent.
5. Integrate through feature flag.
6. Validate v1/v2 coexistence before enabling in deployment.
