# Recommendation Service AI Guide

This document explains the `recommendation-service` in detail: what it does, what is actually AI-driven, what is deterministic, how data flows through the service, which components own which responsibilities, and how the major endpoints work.

It is written as a technical reference for this repository, not as product copy.

## 1. Service Purpose

The `recommendation-service` is a Python FastAPI service that currently owns four major AI-adjacent capabilities:

1. Personalized course recommendations
2. Chatbot / conversational assistant
3. Semantic course search and autocomplete
4. User clustering for collaborative recommendation signals
5. Search feedback persistence and search analytics

The service is not a single monolithic "LLM feature." It mixes:

- deterministic retrieval and ranking
- embeddings and vector search
- clustering and collaborative filtering
- persistent search-feedback learning
- Redis caching
- an LLM-backed chat / legacy recommendation path

In the current setup, the service is increasingly retrieval-driven and deterministic for search and recommendation ranking, while the chatbot remains the main generative AI feature.

## 2. High-Level Architecture

Core runtime dependencies:

- FastAPI application
- PostgreSQL for persistence
- Redis for caching
- Qdrant for vector storage
- `courses-attendance-service` for course catalog and analytics
- `auth-service` for JWT validation
- FreeModel-compatible OpenAI Responses API endpoint for generation

Main code entrypoints:

- [app/main.py](D:/Graduation/recommendation-service/app/main.py)
- [app/config.py](D:/Graduation/recommendation-service/app/config.py)

Main subsystems:

- Recommendation orchestration:
  - [app/services/recommendation_engine.py](D:/Graduation/recommendation-service/app/services/recommendation_engine.py)
- Chatbot orchestration:
  - [app/services/chat_engine.py](D:/Graduation/recommendation-service/app/services/chat_engine.py)
- Model client:
  - [app/services/gemma_client.py](D:/Graduation/recommendation-service/app/services/gemma_client.py)
- Retrieval:
  - [app/retrieval/hybrid_search.py](D:/Graduation/recommendation-service/app/retrieval/hybrid_search.py)
  - [app/retrieval/vector_store.py](D:/Graduation/recommendation-service/app/retrieval/vector_store.py)
  - [app/retrieval/embedding_service.py](D:/Graduation/recommendation-service/app/retrieval/embedding_service.py)
- Semantic search / autocomplete:
  - [app/services/course_search_service.py](D:/Graduation/recommendation-service/app/services/course_search_service.py)
- Clustering:
  - [app/clustering/cluster_service.py](D:/Graduation/recommendation-service/app/clustering/cluster_service.py)

## 3. What Is Actually "AI" Here

There are three different meanings of "AI" in this service:

### 3.1 Generative AI

Used for:

- chatbot responses
- legacy recommendation generation
- optional agentic recommendation planning if enabled

This goes through:

- [app/services/gemma_client.py](D:/Graduation/recommendation-service/app/services/gemma_client.py)

Despite the file name `gemma_client`, the current configuration is not tied to Gemma specifically. The service is configured to call a FreeModel/OpenAI-compatible Responses API endpoint.

Current defaults in config:

- model: `gpt-5.4-mini`
- wire API: `responses`
- reasoning effort: `medium`
- response storage: disabled

### 3.2 Embedding-Based AI

Used for:

- semantic retrieval
- semantic search
- autocomplete candidate retrieval
- clustering feature construction

This is not generative text output. It is embedding generation plus vector similarity search.

Current embedding model:

- `BAAI/bge-small-en-v1.5`

Configured in:

- [app/config.py](D:/Graduation/recommendation-service/app/config.py)

### 3.3 Behavioral / Statistical AI

Used for:

- clustering similar users
- collaborative recommendation signals
- heuristic ranking based on engagement and watch behavior

This is mostly deterministic ML/statistical logic rather than LLM reasoning.

## 4. External Data Dependencies

The recommendation service does not own the source of truth for courses or user learning behavior.

It depends on `courses-attendance-service` for:

- full course catalog
- user analytics profile
- reporting activity

That integration lives in:

- [app/services/course_client.py](D:/Graduation/recommendation-service/app/services/course_client.py)

Important methods:

- `get_all_courses()`
- `get_user_analytics_profile(user_id)`
- `get_activity(user_id, period)`

This means:

- vector search uses course IDs that are hydrated back into live course objects
- watch time, course completions, and subject preferences come from the courses service

## 5. Authentication Model

The service validates bearer tokens by calling `auth-service` instead of verifying them locally.

Implementation:

- [app/api/dependencies.py](D:/Graduation/recommendation-service/app/api/dependencies.py)

Flow:

1. check `Authorization: Bearer ...`
2. call `POST /api/v1/internal/validate-token` on auth-service
3. trust the returned `userId` and `role`

This is why user-facing endpoints in this service are protected even though some internal data comes from other services.

## 6. Startup Behavior

On startup, the service does several important things:

1. verifies/creates database tables
2. ensures Qdrant collections exist
3. fetches all courses
4. recreates and reindexes the course vector collection
5. clears stale search/recommendation cache
6. runs clustering for known users

Startup logic:

- [app/main.py](D:/Graduation/recommendation-service/app/main.py)

Important consequence:

- after service restart, course embeddings and user clustering are refreshed automatically
- persistent search feedback and query analytics remain in Postgres across restarts

## 7. Vector Database Design

Qdrant collections:

- `courses`
- `users`
- `clusters`

Configured in:

- [app/config.py](D:/Graduation/recommendation-service/app/config.py)

Managed in:

- [app/retrieval/vector_store.py](D:/Graduation/recommendation-service/app/retrieval/vector_store.py)

### 7.1 Courses Collection

Stores:

- course embedding vector
- payload metadata such as title, subject, course image, price, popularity, teacher score

Used for:

- recommendation retrieval
- chatbot course context retrieval
- semantic course search
- autocomplete candidate retrieval

### 7.2 Users Collection

Stores user behavior embeddings.

Used as a clustering artifact and future personalization signal.

### 7.3 Clusters Collection

Stores cluster centroid vectors.

This is secondary. The main collaborative filtering signal actually comes from Postgres cluster metadata, not directly from vector search.

## 8. Embedding and Indexing Pipeline

Course embeddings are built from structured course fields such as:

- title
- subject
- categories/tags
- description
- teacher
- popularity

Relevant helper:

- [app/retrieval/course_indexer.py](D:/Graduation/recommendation-service/app/retrieval/course_indexer.py)

The resulting vector is inserted into Qdrant with a payload containing enough metadata for fast retrieval without immediately calling the course service again.

However, for user-facing search results, the service still hydrates from the live course catalog so the frontend gets the full normal course card shape.

## 9. Recommendation System

Main file:

- [app/services/recommendation_engine.py](D:/Graduation/recommendation-service/app/services/recommendation_engine.py)

Top-level entry:

- `get_personalized_recommendations(user_id)`

There are two modes:

1. agentic v2 mode
2. legacy recommendation mode

Controlled by:

- `AGENT_RECOMMENDATIONS_ENABLED`

### 9.1 Current Practical Behavior

In this repo state, recommendation retrieval and ranking are primarily deterministic and retrieval-based.

The LLM is no longer required for the ranking path in the same way it once was. The service now leans more on:

- semantic retrieval
- behavioral signals
- cluster affinity
- popularity / teacher quality

### 9.2 Agentic Recommendation Mode

If enabled, the service calls:

- `recommendation_agent.recommend(user_id)`

and caches:

- recommendations
- explanation trace
- tool trace
- reasoning summary

Cache keys:

- `recommendation:v2:{user_id}`
- `recommendation:v2:explain:{user_id}`

This mode is designed to support explainability and multi-step planning/tool usage.

### 9.3 Legacy Recommendation Mode

Legacy flow:

1. check Redis cache
2. fetch user analytics profile
3. fetch full course catalog
4. remove already enrolled courses
5. build a recommendation prompt
6. send prompt to model
7. parse JSON response
8. hydrate with full course details
9. cache the result

This path still exists in:

- `_get_legacy_recommendations()` in [app/services/recommendation_engine.py](D:/Graduation/recommendation-service/app/services/recommendation_engine.py)

### 9.4 Recommendation Persistence

Recommendation history is stored in Postgres using:

- `RecommendationHistory`

The service attempts to skip stale course IDs when persisting historical recommendations.

## 10. Deterministic Retrieval and Ranking

Core retrieval logic:

- [app/retrieval/hybrid_search.py](D:/Graduation/recommendation-service/app/retrieval/hybrid_search.py)

Main function:

- `search_relevant_courses(user_id, query, top_k, exclude_course_ids, filter_enrolled)`

### 10.1 Retrieval Flow

1. build cache key
2. look in Redis
3. embed query text
4. search Qdrant `courses`
5. build candidate list from payloads
6. optionally remove excluded IDs
7. fetch user analytics profile
8. optionally filter already enrolled courses
9. fetch cluster affinity map
10. compute hybrid score
11. sort and cache

### 10.2 Hybrid Score

The hybrid score currently combines:

- 60% semantic similarity
- 20% popularity
- 10% teacher score
- 10% cluster score

Then an additional watched-subject boost may be added.

This is the important distinction:

- `clusterContribution` is not the whole score
- it is only the surfaced contribution of the cluster part

### 10.3 Subject Boost

If a candidate course belongs to a subject the user already watches a lot, it gets a small boost.

This is a personalization tie-breaker, not the main ranking force.

### 10.4 Cluster Affinity

The cluster system builds a `{courseId: affinity}` map for the user’s assigned cluster.

That lets the service slightly prefer courses that similar users engaged with heavily.

## 11. Clustering System

Main file:

- [app/clustering/cluster_service.py](D:/Graduation/recommendation-service/app/clustering/cluster_service.py)

Purpose:

- create a collaborative-filtering signal from user behavior

### 11.1 Feature Construction

Each user gets a feature vector made from:

1. text behavior summary embedding
2. numeric behavior features such as:
   - total courses
   - total watch time
   - average completion
   - average engagement
   - cart subject count
   - top category count

### 11.2 Clustering Algorithm

- KMeans
- features standardized first
- cluster count chosen dynamically based on user count

### 11.3 Output Artifacts

For each user:

- assigned cluster
- distance to centroid

For each cluster:

- top courses
- top subjects
- centroid vector
- user count

Stored in:

- Postgres cluster tables
- Qdrant clusters collection

### 11.4 Why Cluster Contribution Can Be Zero

Cluster contribution is zero when:

- user has no cluster assignment
- cluster has no top course affinity for that course
- the course is not one of the cluster’s preferred courses
- cluster lookup failed

This is normal and not a bug by itself.

## 12. Semantic Course Search

Main logic:

- [app/services/course_search_service.py](D:/Graduation/recommendation-service/app/services/course_search_service.py)

Exposed through:

- `GET /api/v1/courses?search=...`

via:

- [app/api/routes/courses.py](D:/Graduation/recommendation-service/app/api/routes/courses.py)

### 12.1 Why Search Is in Recommendation Service

Search uses:

- embeddings
- Qdrant vectors
- personalization tie-breakers

So it belongs in the recommendation-service rather than the SQL-only course listing service.

### 12.2 Search Flow

1. receive query and filters
2. check Redis cache
3. fetch all courses for hydration
4. fetch user analytics profile
5. embed the search text
6. query Qdrant for candidate courses
7. hydrate vector IDs back to full course objects
8. apply filters after hydration
9. apply lexical scoring and query expansion
10. apply personalization tie-breakers
11. apply repeated feedback boosts for this query
12. record query analytics
13. paginate and cache

### 12.3 Search Ranking

Search ranking uses:

1. semantic similarity
2. lexical match quality
3. repeated search-feedback boosts for the same query
4. tiny watched-subject boost
5. tiny cluster-affinity boost
6. small popularity / teacher-rating tie-breakers

Search does not exclude enrolled courses.

### 12.4 Query Expansion

The search service expands some common intent phrases into practical alternatives before embedding and lexical matching.

Examples:

- `hands-on` -> `practical`, `workshop`, `lab`, `bootcamp`, `project based`
- `beginner` -> `intro`, `fundamentals`, `essentials`
- `advanced` -> `professional`, `masterclass`, `deep dive`
- `project based` -> `project`, `capstone`, `lab`

This helps search match the way users phrase intent, even when the exact catalog wording differs.

### 12.5 Search Feedback Learning

Search now learns from explicit result feedback.

Feedback endpoint:

- `POST /api/v1/courses/feedback`

Accepted event types:

- `click`
- `preview`
- `watch`
- `enroll`

Current event weights:

- `click` = `1.0`
- `preview` = `1.25`
- `watch` = `1.5`
- `enroll` = `2.5`

The service stores feedback in two places:

- Redis for short-term ranking memory
- Postgres for durable history and analytics

That means repeated positive interactions for the same query/course pair can increasingly boost that course for future similar searches.

### 12.6 Search Fallback

If semantic search yields no useful result:

- the service falls back to keyword-style matching over the hydrated course catalog

## 13. Autocomplete

Endpoint:

- `GET /api/v1/courses/autocomplete?search=...`

Main logic:

- [app/services/course_search_service.py](D:/Graduation/recommendation-service/app/services/course_search_service.py)

### 13.1 Why Autocomplete Needed Special Handling

Pure semantic similarity is often too loose for short queries.

Example:

- query: `pyth`

A purely semantic system can return:

- data engineering
- analytics
- generic bootcamps

because semantically they live near Python/data concepts in vector space.

That feels wrong for a search box.

### 13.2 Current Autocomplete Strategy

Autocomplete is now lexical-first hybrid ranking.

It does:

1. get semantic candidates from Qdrant
2. compute strong lexical score
3. require lexical resemblance for very short queries
4. apply query-feedback boosts when the same short query has strong positive history
5. use semantic similarity as a secondary helper
6. produce course suggestions and deduped subject suggestions

### 13.3 Lexical Signals Used

The scorer heavily rewards:

- title starts with query
- title token starts with query
- title contains query
- subject prefix / contains
- teacher prefix / contains
- fuzzy token similarity

For short queries, irrelevant semantic neighbors are filtered out more aggressively.

### 13.4 Autocomplete Output Shape

Course suggestion:

- `type`
- `courseId`
- `title`
- `subjectName`
- `courseImage`
- `score`

Subject suggestion:

- `type`
- `subjectName`
- `score`

## 14. Chatbot

Main file:

- [app/services/chat_engine.py](D:/Graduation/recommendation-service/app/services/chat_engine.py)

This is the most clearly generative AI part of the service.

### 14.1 Chatbot Responsibilities

- chat session creation/listing/deletion
- history loading
- input validation
- retrieval-augmented course context
- system prompt construction
- conversation message construction
- model invocation
- SSE streaming
- output sanitization
- persistence of user/assistant messages
- optional media upload to Cloudinary

### 14.2 Chatbot Is a RAG System

Yes, the chatbot behaves as a RAG-style system.

RAG flow:

1. user sends a message
2. service retrieves relevant courses using `search_relevant_courses(...)`
3. those courses are summarized into prompt context
4. system prompt is built from retrieved course context
5. model responds using that grounded context

If retrieval fails, it falls back to the broader course catalog context.

So the chatbot is not answering from the LLM alone. It is grounded with retrieved platform course data.

### 14.3 Chat Prompt Construction

Helpers:

- [app/utils/chat_prompt_builder.py](D:/Graduation/recommendation-service/app/utils/chat_prompt_builder.py)

The engine builds:

- a system prompt containing relevant course context
- conversation messages from previous history + current user turn

### 14.4 Streaming

The chatbot streams response chunks as Server-Sent Events.

Event types include:

- `chunk`
- `correction`
- `done`
- `error`
- `retrieval`

This gives the frontend real-time incremental output.

### 14.5 Safety / Guardrails

Input validation:

- [app/utils/content_guard.py](D:/Graduation/recommendation-service/app/utils/content_guard.py)

Output sanitization:

- also applied before final persistence/return

### 14.6 Media Handling

If the user sends media:

- media is attached to the final turn
- model request can include image input
- uploaded media is persisted to Cloudinary

## 15. Model Client

Main file:

- [app/services/gemma_client.py](D:/Graduation/recommendation-service/app/services/gemma_client.py)

Despite the name, this file is now a general model client for an OpenAI-compatible Responses API.

### 15.1 Key Behaviors

- adds bearer auth header
- talks to `/v1/responses`
- converts internal message format to Responses API input format
- supports retry on:
  - 429
  - 500
  - 502
  - 503
  - 504
  - connection errors
  - read timeouts

### 15.2 Structured JSON Extraction

The client tries hard to extract JSON from model outputs, including:

- fenced code blocks
- object/array slices

This is why recommendation generation can ask the model for JSON arrays and still recover if the model wraps them awkwardly.

### 15.3 Chat Streaming Note

The current `stream_chat` implementation calls the model and yields the full extracted text as a single chunk from the Responses API result. The chat engine still uses an SSE streaming interface, but the upstream model call itself is not token-streamed in the current implementation.

So:

- frontend gets SSE events
- but model output is effectively batched before being yielded

## 16. Caching Strategy

Redis is used heavily.

### 16.1 Recommendation Cache

- `recommendation:v1:{user_id}`
- `recommendation:v2:{user_id}`
- `recommendation:v2:explain:{user_id}`

### 16.2 Retrieval Cache

- `retrieval:v1:{hash}`

### 16.3 Search Cache

- `course-search:v1:{hash}`

### 16.4 Autocomplete Cache

- `course-autocomplete:v1:{hash}`

### 16.5 Search Feedback Cache

- `course-search:feedback:{query_hash}`
- `course-search:recent:{user_id}`

### 16.6 Chat Course Context Cache

- `chatbot:course_context`

Caching keeps:

- search fast
- recommendation refresh cheaper
- course retrieval reusable
- chat course catalog fetches lighter
- recent feedback and recent searches available as low-latency ranking signals

## 17. Exposed API Surface

Main routers:

- recommendations router
- chatbot router
- reports router
- courses router

Mounted in:

- [app/main.py](D:/Graduation/recommendation-service/app/main.py)

Important public endpoints:

### Recommendations

- `GET /api/v1/recommendations`
- `GET /api/v1/recommendations/explain`
- `GET /api/v1/recommendations/debug`
- `POST /api/v1/recommendations/refresh`
- `POST /api/v1/recommendations/clusters/rebuild`
- `GET /api/v1/recommendations/trending`

### Cluster inspection

- `GET /api/v1/recommendations/clusters/{user_id}`
- `GET /api/v1/recommendations/clusters/{cluster_id}/top-courses`

### Chatbot

- `POST /api/v1/chatbot`
- `GET /api/v1/chatbot`
- `PATCH /api/v1/chatbot/{chat_id}`
- `DELETE /api/v1/chatbot/{chat_id}`
- `POST /api/v1/chatbot/{chat_id}/messages`
- `GET /api/v1/chatbot/{chat_id}/messages`
- `POST /api/v1/chatbot/{chat_id}/messages/binary`

### Search

- `GET /api/v1/courses?search=...`
- `GET /api/v1/courses/autocomplete?search=...`
- `POST /api/v1/courses/feedback`
- `GET /api/v1/courses/analytics/top-clicked`
- `GET /api/v1/courses/analytics/zero-results`
- `GET /api/v1/courses/analytics/top-query-courses`

## 18. Gateway Routing

In `api-gateway`, the following are intentionally split:

- `GET /api/v1/courses` without `search` -> courses service
- `GET /api/v1/courses?search=...` -> recommendation service
- `GET /api/v1/courses/autocomplete?...` -> recommendation service
- `POST /api/v1/courses/feedback` -> recommendation service
- `GET /api/v1/courses/analytics/*` -> recommendation service
- `GET /api/v1/courses/:id` -> courses service

Gateway implementation:

- [api-gateway/src/routes/index.ts](D:/Graduation/api-gateway/src/routes/index.ts)

## 19. Current Behavioral Summary

### Recommendations

- mostly deterministic retrieval/ranking
- collaborative filtering through clusters
- optional agentic path exists
- legacy LLM path still exists

### Chatbot

- real generative AI feature
- RAG grounded on retrieved course context

### Search

- semantic retrieval
- deterministic ranking
- query expansion
- persistent behavioral feedback learning
- fallback to lexical keyword matching

### Autocomplete

- lexical-first hybrid
- semantic assistance only
- tuned for short query quality

### Search Analytics

- tracks top clicked queries
- tracks zero-result queries
- tracks strongest query-course pairs
- useful for search tuning, typo handling, synonym expansion, and course gap discovery

## 20. Known Limitations / Observations

1. `stream_chat()` currently behaves more like buffered generation than true token-level upstream streaming.
2. The file name `gemma_client.py` no longer reflects the current provider abstraction well.
3. Startup reindexing recreates the course vector collection, which is simple and clean locally but may be expensive at larger scale.
4. Recommendation generation contains both legacy and newer paths, so the service carries some historical complexity.
5. Search and autocomplete quality depend on:
   - course text quality
   - embedding quality
   - how rich the course metadata is
   - enough real feedback volume to teach the ranking layer
6. Search feedback persistence is currently stored in recommendation-service Postgres tables and updated in application code, not through a separate event pipeline.
7. Startup reindexing still rebuilds course vectors eagerly, which is correct locally but heavier than an incremental refresh approach.
8. Gateway/runtime rebuild mismatches can make it look like routes are broken when an old container is still serving stale code.

## 21. Recommended Reading Order in Code

If you want to understand the service deeply, read in this order:

1. [app/main.py](D:/Graduation/recommendation-service/app/main.py)
2. [app/config.py](D:/Graduation/recommendation-service/app/config.py)
3. [app/services/course_client.py](D:/Graduation/recommendation-service/app/services/course_client.py)
4. [app/retrieval/vector_store.py](D:/Graduation/recommendation-service/app/retrieval/vector_store.py)
5. [app/retrieval/hybrid_search.py](D:/Graduation/recommendation-service/app/retrieval/hybrid_search.py)
6. [app/clustering/cluster_service.py](D:/Graduation/recommendation-service/app/clustering/cluster_service.py)
7. [app/services/course_search_service.py](D:/Graduation/recommendation-service/app/services/course_search_service.py)
8. [app/models/search.py](D:/Graduation/recommendation-service/app/models/search.py)
9. [app/services/recommendation_engine.py](D:/Graduation/recommendation-service/app/services/recommendation_engine.py)
10. [app/services/chat_engine.py](D:/Graduation/recommendation-service/app/services/chat_engine.py)
11. [app/services/gemma_client.py](D:/Graduation/recommendation-service/app/services/gemma_client.py)

## 22. Short Answer Version

If someone asks, "How does the recommendation-service AI work?" the concise technical answer is:

The service combines embeddings, Qdrant vector search, heuristic ranking, Redis caching, persistent Postgres feedback history, and user clustering to power recommendations, search, and autocomplete. The chatbot is a RAG-style assistant that retrieves relevant course context before calling a model through a FreeModel/OpenAI-compatible Responses API. Search and autocomplete are deterministic and LLM-free. Recommendation ranking is mostly deterministic with optional agentic/LLM-assisted paths still present in the codebase.
