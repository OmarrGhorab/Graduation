# Feature Specification: Agentic Recommendations

**Feature Branch**: `001-agentic-recommendations`

**Created**: 2026-05-21

**Status**: Draft

**Input**: Transform the existing recommendation-service into an agentic AI recommendation platform with MCP-style tools, RAG retrieval, Qdrant vector search, user clustering, embeddings, hybrid ranking, and LLM reasoning while preserving the working service and migrating incrementally.

## User Scenarios & Testing

### User Story 1 - Agentic Personalized Recommendations (Priority: P1)

As a student, I want recommendations that are generated from my behavior, similar learners, retrieved course matches, and platform trends so that I receive relevant course suggestions without the system sending the entire course catalog to the AI model.

**Why this priority**: This is the main user-facing value and the MVP replacement for the current prompt-based recommendation path.

**Independent Test**: Can be tested by requesting recommendations for a known user and confirming the response contains ranked courses with confidence, reasons, retrieval sources, and no already-enrolled courses.

**Acceptance Scenarios**:

1. **Given** a student with analytics history and available courses, **When** they request recommendations, **Then** the system returns a ranked list built from retrieved candidates, cluster signals, and trend signals.
2. **Given** a student has already enrolled in several courses, **When** recommendations are generated, **Then** those enrolled courses are excluded from the final list.
3. **Given** the AI ranking step fails, **When** recommendations are requested, **Then** the system returns safe fallback recommendations from deterministic retrieval and trending signals.

---

### User Story 2 - Explainable Recommendation Reasoning (Priority: P2)

As a student or support operator, I want to understand why a course was recommended so that recommendations feel trustworthy and actionable.

**Why this priority**: Explanation is required for trust, debugging, and validating the new agentic behavior.

**Independent Test**: Can be tested by calling the explanation endpoint after recommendations are generated and confirming every recommendation includes a concise reason and source attribution.

**Acceptance Scenarios**:

1. **Given** recommendations have been generated, **When** explanation details are requested, **Then** the system returns the tools, retrieval sources, and high-level reasoning summary used for the result.
2. **Given** retrieved context includes noisy or unsafe text, **When** explanations are generated, **Then** unsafe text is sanitized and not repeated verbatim.

---

### User Story 3 - Cluster-Based Discovery (Priority: P3)

As a platform operator, I want students assigned to behavioral clusters and cluster-level top courses exposed through APIs so that recommendations can use peer behavior and operators can inspect cluster quality.

**Why this priority**: Clustering improves personalization and gives operators a way to validate behavior cohorts.

**Independent Test**: Can be tested by running the clustering job and calling cluster endpoints for a user and a cluster.

**Acceptance Scenarios**:

1. **Given** users have analytics profiles, **When** the clustering job runs, **Then** each eligible user receives a persisted cluster assignment.
2. **Given** a cluster has assigned users, **When** top courses for the cluster are requested, **Then** the system returns courses ranked by cluster affinity and engagement.

---

### User Story 4 - Incremental Migration And Rollback (Priority: P4)

As an engineering team, I want the new recommendation path behind a feature flag so that the current working flow remains available during rollout.

**Why this priority**: The existing system works and must not be broken while the agentic platform is introduced.

**Independent Test**: Can be tested by toggling the feature flag and confirming the service switches between the legacy and agentic paths.

**Acceptance Scenarios**:

1. **Given** the agentic feature flag is disabled, **When** recommendations are requested, **Then** the legacy recommendation flow remains available.
2. **Given** the agentic feature flag is enabled, **When** recommendations are requested, **Then** the agentic path is used and cached separately from legacy results.

### Edge Cases

- A user has no analytics history: return trending and semantic cold-start recommendations with a clear reason.
- The vector index has no matching courses: fall back to trending courses and record the retrieval miss.
- Qdrant is unavailable: skip semantic retrieval, use cached recommendations or deterministic fallback.
- The clustering job has not assigned the user yet: continue without cluster contribution and enqueue or allow later refresh.
- The LLM returns malformed or unsafe output: validate, discard invalid items, and use deterministic fallback ranking.
- Course metadata contains prompt-injection-like instructions: sanitize retrieved text before it enters model context.
- Redis is unavailable: continue request processing without cache and log the cache failure.

## Requirements

### Functional Requirements

- **FR-001**: System MUST generate recommendations through an agentic workflow that dynamically retrieves only needed context.
- **FR-002**: System MUST NOT preload the full course catalog into LLM prompts.
- **FR-003**: System MUST expose MCP-style tools with name, description, JSON input schema, JSON output schema, and validated execution.
- **FR-004**: System MUST provide tools for user profile, user cluster, similar users, relevant course search, user history, trending courses, recent cart activity, and cluster top courses.
- **FR-005**: System MUST create embeddings for courses, user behavior summaries, and clusters.
- **FR-006**: System MUST support semantic course retrieval and hybrid ranking that combines similarity, popularity, teacher authority, and cluster contribution.
- **FR-007**: System MUST assign eligible users to persisted behavior clusters.
- **FR-008**: System MUST expose APIs for recommendations, recommendation explanations, refresh, user cluster lookup, and cluster top courses.
- **FR-009**: Recommendation responses MUST include course info, confidence score, explanation, similarity source, and cluster contribution.
- **FR-010**: System MUST cache recommendations, retrieval results, and embeddings where appropriate to avoid repeated expensive work.
- **FR-011**: System MUST preserve the current recommendation path until the agentic path is explicitly enabled.
- **FR-012**: System MUST record reasoning logs, tool execution logs, retrieval timing, and fallback events for observability.
- **FR-013**: System MUST validate tool inputs, bound tool output sizes, sanitize retrieved text, and prevent model output from invoking arbitrary operations.
- **FR-014**: All recommendation and cluster endpoints MUST return the platform response envelope `{ success, data, error, message }` with machine-readable error codes on failure.

### Key Entities

- **RecommendationResult**: A ranked course suggestion with confidence, explanation, source signals, and hydrated course metadata.
- **ToolDefinition**: An allowlisted callable contract with schemas and execution metadata.
- **VectorDocument**: Embedded course, user, or cluster representation with safe metadata and tags.
- **UserCluster**: Persisted assignment between a user and a behavior cluster.
- **ClusterMetadata**: Cluster label, centroid, top subjects, top courses, user count, and version metadata.
- **ReasoningTrace**: High-level trace of tool calls, retrieval sources, ranking decisions, and fallback behavior.

## Success Criteria

### Measurable Outcomes

- **SC-001**: At least 95% of recommendation requests return a valid response without requiring the full course catalog in model context.
- **SC-002**: Recommendation responses include explanations and source attribution for 100% of returned courses.
- **SC-003**: Cached recommendation responses are returned in under 500 ms for typical users in local service conditions.
- **SC-004**: Fresh agentic recommendation generation completes in under 5 seconds for typical users with indexed courses.
- **SC-005**: The system can safely fall back to non-agentic ranking when the LLM, vector store, or cluster data is unavailable.
- **SC-006**: Every eligible user processed by the clustering job receives one active cluster assignment.
- **SC-007**: Engineering can disable the new path with a feature flag without removing deployed code.
- **SC-008**: Public recommendation and cluster APIs conform to the shared response envelope and error-code contract on every success and failure path.

## Assumptions

- Existing authentication, internal service secret validation, course service integration, analytics profile, Redis, and Gemma integration are reused.
- Qdrant is introduced as a separate vector database rather than replacing the existing Postgres image.
- The initial embedding model is `BAAI/bge-small-en-v1.5`.
- KMeans is the first clustering algorithm, with the clustering service structured so DBSCAN can be added later.
- Fresh agentic recommendation generation is an explicit latency exception to the platform-wide 200ms p95 rule because it performs bounded retrieval and LLM reasoning; cached recommendation responses and dependency health checks remain optimized and monitored.
- The current endpoint prefix under `/api/v1/recommendations` remains the public API surface.
- The initial release prioritizes service-side behavior and API compatibility over frontend changes.
