# Decisions: Agentic Recommendations

## ADR-001: Use Qdrant For Vector Search

**Decision**: Add Qdrant as a separate vector database service.

**Reasoning**: The existing Docker stack uses `postgres:15-alpine`, which does not provide pgvector out of the box. Qdrant avoids changing the shared database image and isolates vector workload from existing services.

**Tradeoff**: Adds one new service to operate, but keeps the migration lower risk.

## ADR-002: Use LangGraph For Agent Orchestration

**Decision**: Use LangGraph for the recommendation agent workflow.

**Reasoning**: The target workflow needs explicit nodes, bounded tool loops, fallback paths, and stateful reasoning. LangGraph fits this better than ad hoc orchestration.

**Tradeoff**: Adds framework dependency and learning curve.

## ADR-003: Use MCP-Style Internal Tool Registry

**Decision**: Implement MCP-style tools as internal allowlisted function contracts with Pydantic schemas.

**Reasoning**: The system needs tool selection and validation without exposing arbitrary execution to the LLM.

**Tradeoff**: This is MCP-style rather than a full external MCP server in the initial phase.

## ADR-004: Use BAAI/bge-small-en-v1.5 For Initial Embeddings

**Decision**: Use `BAAI/bge-small-en-v1.5` through sentence-transformers.

**Reasoning**: It is open-source, small enough for service-side use, and appropriate for semantic retrieval.

**Tradeoff**: English-first embedding quality may be weaker for multilingual course metadata; future models can be swapped behind the embedding service.

## ADR-005: Use KMeans For Initial Clustering

**Decision**: Use KMeans for the first user clustering implementation.

**Reasoning**: KMeans is simple, deterministic enough for initial production rollout, and easy to inspect.

**Tradeoff**: Requires selecting cluster count. The clustering module should allow DBSCAN or another algorithm later.

## ADR-006: Preserve Legacy Recommendation Flow Behind Feature Flag

**Decision**: Add `AGENT_RECOMMENDATIONS_ENABLED` and keep the current prompt-based path available.

**Reasoning**: The current system works and must remain a rollback path.

**Tradeoff**: Temporary duplication until the v2 path is validated and adopted.

## ADR-007: Separate v1 And v2 Cache Keys

**Decision**: Use `recommendation:v1:{user_id}` for legacy and `recommendation:v2:{user_id}` for agentic recommendations.

**Reasoning**: Prevents cache shape collisions and enables safe rollout and rollback.

**Tradeoff**: Duplicate cache entries during migration.

## ADR-008: Rollout v2 Behind Feature Flag With Explicit Rollback Path

**Decision**: Keep `AGENT_RECOMMENDATIONS_ENABLED` disabled by default and roll out v2 gradually while preserving v1 endpoints and cache keys.

**Reasoning**: The existing recommendation path works and provides a safe rollback path if the agentic flow needs to be disabled.

**Tradeoff**: Operates both paths during migration, but avoids user-visible regression risk.

## ADR-009: Fresh Agentic Generation Is A Documented Latency Exception

**Decision**: Treat fresh agentic recommendation generation as an explicit exception to the general 200ms p95 API guidance because it performs bounded retrieval and LLM reasoning.

**Reasoning**: The workflow is intentionally more expensive than cached reads and is measured separately from the cached path.

**Tradeoff**: Adds one documented exception, but keeps the performance model honest and aligned with the constitution.

## ADR-010: Standardize Recommendation-Family Response Envelopes

**Decision**: Use the shared `{ success, data, error, message }` envelope for recommendation and cluster endpoints, with machine-readable error codes.

**Reasoning**: This aligns the service with the platform API contract and avoids client-side branching on inconsistent error shapes.

**Tradeoff**: Requires shared helper functions and explicit tests, but simplifies integration and debugging.
