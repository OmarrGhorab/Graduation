# Architecture and Technology References for Graduation Report

## Introduction

This document provides formal reference material to justify the architectural and technology decisions used in the graduation project. The project is an education platform that combines user authentication, course management, attendance tracking, payment processing, notifications, real-time chat, AI-supported recommendations, and a cross-platform mobile application.

Because the platform includes multiple domains with different runtime characteristics, the implementation was designed as a microservices-based backend supported by a mobile client built with Expo and React Native. The references below are intended to support the architectural discussion in the graduation report and presentation.

## System Overview

The system consists of:

- a backend composed of multiple services, each responsible for a specific domain
- an API gateway that acts as the main client entry point
- shared infrastructure for persistence, caching, event streaming, and observability
- a mobile application that consumes backend APIs and real-time services

This design was selected to support maintainability, separation of concerns, scalability, and the ability to integrate both transactional and real-time features in the same platform.

## 1. References Supporting the Use of Microservices

Microservices were selected because the platform includes several distinct responsibilities, including authentication, attendance, payments, chat, notifications, and AI-based recommendation features. These concerns differ in terms of scaling requirements, deployment frequency, and technical dependencies.

### Reference 1

- Source: Martin Fowler, "Microservices"
- Link: https://www.martinfowler.com/microservices/
- Relevance: This reference is widely used to define microservices and explain their main characteristics, especially independent deployability, bounded service responsibilities, and separation of concerns.

### Reference 2

- Source: Microsoft Learn, "Microservices architecture"
- Link: https://learn.microsoft.com/en-us/dotnet/architecture/microservices/architect-microservice-container-applications/microservices-architecture
- Relevance: Microsoft explains that microservices are appropriate when application components need to evolve independently and scale according to different workload patterns.

### Reference 3

- Source: Microservices.io, "Pattern Language"
- Link: https://microservices.io/patterns/index.html
- Relevance: This source documents common microservice patterns, including API gateways, communication styles, and service decomposition strategies that are directly relevant to this project.

### Formal justification

Based on these references, a microservices architecture is appropriate for this project because the system includes both transactional and real-time services, along with AI-related features that require different operational and development approaches. It also supports independent scaling and clearer service ownership.

## 2. References Supporting the Use of an API Gateway

An API gateway was used as the main public-facing backend entry point. Its role is to centralize request routing and common cross-cutting concerns such as security policies and request management.

### Reference 1

- Source: Microservices.io, "API Gateway"
- Link: https://microservices.io/patterns/apigateway.html
- Relevance: This reference defines the API gateway pattern as a standard way to provide clients with a single entry point in a microservices architecture.

### Reference 2

- Source: Microsoft Learn, "Direct client-to-microservice communication versus the API Gateway pattern"
- Link: https://learn.microsoft.com/en-ca/dotnet/architecture/microservices/architect-microservice-container-applications/direct-client-to-microservice-communication-versus-the-api-gateway-pattern
- Relevance: This reference explains why an API gateway helps reduce client complexity and centralizes policies such as routing, protocol handling, and security enforcement.

### Reference 3

- Source: Apache APISIX, "API Gateway for Microservices"
- Link: https://apisix.apache.org/learning-center/api-gateway-for-microservices/
- Relevance: This source provides a practical explanation of how API gateways improve manageability in distributed systems.

### Formal justification

The use of an API gateway in this project is justified because the mobile and web clients should not communicate separately with every internal service. A gateway provides one stable interface while allowing routing and policy decisions to remain centralized.

## 3. References Supporting the Use of PostgreSQL

PostgreSQL was selected as the primary persistent data store for users, courses, enrollments, attendance records, payment data, and other relational entities.

### Reference 1

- Source: PostgreSQL Global Development Group, "About PostgreSQL"
- Link: https://www.postgresql.org/about/
- Relevance: The PostgreSQL project describes the system as reliable, standards-compliant, and suitable for complex relational applications.

### Reference 2

- Source: PostgreSQL Documentation, "Reliability and the Write-Ahead Log"
- Link: https://www.postgresql.org/docs/14/wal-reliability.html
- Relevance: This documentation supports the use of PostgreSQL in systems that require strong durability and transactional consistency.

### Formal justification

PostgreSQL is a suitable choice because this project includes multiple business-critical workflows, especially authentication data, course enrollment, and payment-related records, all of which require durable storage and transactional consistency.

## 4. References Supporting the Use of Redis

Redis was used for caching, ephemeral state, and real-time coordination features such as session-like data, short-lived tokens, pub/sub behavior, and performance optimization.

### Reference 1

- Source: Redis, "What is Redis?"
- Link: https://redis.io/tutorials/what-is-redis/
- Relevance: The official Redis explanation presents Redis as an in-memory system suitable for high-speed data access, caching, and message-oriented use cases.

### Reference 2

- Source: Redis Documentation, "Use Cases"
- Link: https://redis.io/docs/latest/develop/use-cases/
- Relevance: This source explicitly documents Redis use cases such as caching, session storage, and pub/sub messaging, which match this project's needs.

### Reference 3

- Source: Redis, "Caching"
- Link: https://redis.io/solutions/use-cases/caching/
- Relevance: This resource explains how Redis can reduce database load and improve application responsiveness.

### Formal justification

Redis is appropriate because not all application data should be stored and retrieved from the main relational database. High-frequency, short-lived, or rapidly changing state benefits from an in-memory system with low-latency access.

## 5. References Supporting the Use of Kafka

Kafka was used as the event-streaming backbone to support asynchronous communication between services.

### Reference 1

- Source: Apache Kafka, "Use Cases"
- Link: https://kafka.apache.org/22/getting-started/uses/
- Relevance: This official source explains Kafka's role in publish-subscribe pipelines, streaming, and event distribution across systems.

### Reference 2

- Source: IBM, "Apache Kafka use cases"
- Link: https://www.ibm.com/think/topics/apache-kafka-use-cases
- Relevance: This source is useful for describing Kafka in broader system-design terms, especially in event-driven architectures.

### Formal justification

Kafka is suitable because several features in the platform should be decoupled from the main request-response cycle. Examples include notifications, event-driven updates, and inter-service reactions that do not need to block user-facing requests.

## 6. References Supporting the Use of Go in Core Backend Services

Go was used for services such as chat, websocket handling, attendance, and payments.

### Reference 1

- Source: Go Documentation
- Link: https://go.dev/doc/docs.html
- Relevance: The official documentation provides the language foundation and supports Go's use in systems that rely on concurrency and efficient execution.

### Reference 2

- Source: Tu et al., "Understanding Real-World Concurrency Bugs in Go"
- Link: https://songlh.github.io/paper/go-study.pdf
- Relevance: Although this is a research paper about concurrency bugs, it also reflects the central role concurrency plays in Go applications and why the language is frequently used for concurrent systems.

### Formal justification

Go is appropriate for these services because they involve concurrent workloads, low-latency behavior, and continuous network activity. This includes websocket traffic, chat presence, attendance processing, and payment operations.

## 7. References Supporting the Use of TypeScript and Node.js

TypeScript and Node.js were used in the API gateway, authentication service, and notification service.

### Reference 1

- Source: Node.js Documentation
- Link: https://nodejs.org/en/docs
- Relevance: The official Node.js documentation provides the foundation for an event-driven runtime widely used in API and middleware layers.

### Reference 2

- Source: TypeScript Documentation
- Link: https://www.typescriptlang.org/docs/
- Relevance: The TypeScript documentation supports the use of static typing in large JavaScript codebases, improving maintainability and reducing integration errors.

### Formal justification

This stack is suitable for services that coordinate requests, enforce policies, and integrate with multiple downstream dependencies. TypeScript also improves reliability in multi-service environments by making data contracts easier to maintain.

## 8. References Supporting the Use of Python and FastAPI for AI Features

Python and FastAPI were used in the recommendation and AI assistant service.

### Reference 1

- Source: FastAPI Documentation, "Features"
- Link: https://fastapi.tiangolo.com/features/
- Relevance: FastAPI is documented as a modern framework that offers automatic API documentation and efficient API development patterns.

### Reference 2

- Source: FastAPI Documentation, "Concurrency and async / await"
- Link: https://fastapi.tiangolo.com/async/
- Relevance: This reference supports using FastAPI in services that combine external calls, I/O-bound tasks, and asynchronous workflows.

### Formal justification

Python is widely used in AI and machine learning workflows, and FastAPI provides a practical method for exposing those capabilities through APIs. This makes it suitable for recommendation, reporting, and AI chat functionality.

## 9. References Supporting Observability and Monitoring

Observability is important in distributed systems because service interactions, failures, and performance bottlenecks are harder to detect than in a monolithic system.

### Reference 1

- Source: OpenTelemetry Documentation
- Link: https://opentelemetry.io/docs/
- Relevance: OpenTelemetry is described as a vendor-neutral observability framework for traces, metrics, and logs across multiple languages and services.

### Reference 2

- Source: Prometheus Documentation, "Overview"
- Link: https://prometheus.io/docs/introduction/overview/
- Relevance: Prometheus is a recognized monitoring solution for collecting and querying service metrics.

### Reference 3

- Source: Prometheus Home Page
- Link: https://prometheus.io/
- Relevance: Useful general reference for monitoring and metrics-based observability in service-oriented systems.

### Formal justification

Because the project contains multiple services and asynchronous communication paths, observability tooling is necessary to monitor system behavior, trace requests across services, and diagnose failures effectively.

## 10. References Supporting the Mobile Application Stack

The mobile application source at `C:\Users\Raven_dev\Downloads\Graduation-RN-Source` shows the use of Expo, React Native, Expo Router, TypeScript, React Query, notifications, location services, and websocket-based real-time communication.

### Reference 1

- Source: Expo Documentation
- Link: https://docs.expo.dev/
- Relevance: Expo provides a development and deployment workflow for React Native applications and simplifies access to native device capabilities.

### Reference 2

- Source: Expo Router Documentation
- Link: https://docs.expo.dev/router/introduction/
- Relevance: Supports the file-based routing approach used by the mobile application.

### Reference 3

- Source: React Native Documentation
- Link: https://reactnative.dev/docs/getting-started
- Relevance: Official reference for cross-platform native mobile application development using React Native.

### Reference 4

- Source: TypeScript Documentation
- Link: https://www.typescriptlang.org/docs/
- Relevance: Supports type-safe frontend and mobile development in a large application with many screens and services.

### Reference 5

- Source: TanStack Query Documentation, "Overview"
- Link: https://tanstack.com/query/latest/docs/framework/react/overview
- Relevance: This reference supports the use of React Query for server-state management, caching, synchronization, and background updates.

### Reference 6

- Source: Expo Notifications Documentation
- Link: https://docs.expo.dev/versions/latest/sdk/notifications/
- Relevance: Supports the push-notification capability used by the mobile application.

### Reference 7

- Source: Expo Location Documentation
- Link: https://docs.expo.dev/versions/latest/sdk/location/
- Relevance: Supports the use of device location services for attendance and parent-monitoring related features.

### Reference 8

- Source: MDN Web Docs, "WebSocket"
- Link: https://developer.mozilla.org/en-US/docs/Web/API/WebSocket
- Relevance: Provides a recognized reference for real-time bidirectional communication, which is relevant to the chat and presence features used by the app.

### Formal justification

The mobile stack is appropriate because the project requires a cross-platform application with access to native capabilities such as push notifications, location, media, and real-time communication. Expo and React Native reduce development overhead while preserving mobile functionality.

## 11. References Supporting Scalability Claims

The project's scalability argument is based on independent service scaling, cache-assisted performance optimization, and decoupled event-driven communication.

### Key supporting references

- Microsoft Learn, "Microservices architecture"
- Link: https://learn.microsoft.com/en-us/dotnet/architecture/microservices/architect-microservice-container-applications/microservices-architecture
- Relevance: Supports the claim that services can scale independently.

- Redis, "Caching"
- Link: https://redis.io/solutions/use-cases/caching/
- Relevance: Supports the use of caching to reduce primary database load.

- Apache Kafka, "Use Cases"
- Link: https://kafka.apache.org/22/getting-started/uses/
- Relevance: Supports asynchronous and decoupled scaling through event-driven patterns.

- Prometheus Documentation, "Overview"
- Link: https://prometheus.io/docs/introduction/overview/
- Relevance: Supports the monitoring requirements of scalable distributed systems.

### Formal justification

This architecture scales effectively because different platform features produce different kinds of load. For example, chat and websocket traffic can grow independently from authentication traffic, and AI workloads can scale independently from course or payment workflows. The chosen architecture supports this separation.

## 12. Suggested Academic Wording

The following wording can be reused in the graduation report:

"The system was implemented using a microservices architecture because the project combines multiple functional domains, including authentication, attendance, payments, real-time communication, notifications, and AI-supported recommendation features. According to recognized architectural references such as Martin Fowler, Microsoft Learn, and Microservices.io, microservices are appropriate when services require independent deployment, clear domain boundaries, and separate scaling behavior."

"An API gateway was introduced to provide a unified client entry point and to centralize routing and policy-related concerns. This follows the API gateway pattern described in Microservices.io and Microsoft architectural guidance."

"PostgreSQL was selected for durable relational storage, Redis for caching and short-lived operational state, Kafka for asynchronous event-driven communication, Go for concurrency-oriented backend services, and Python with FastAPI for AI-related features. The mobile application was built with Expo and React Native in order to support cross-platform development with access to native mobile capabilities."

## 13. Compact Reference List

1. Martin Fowler. Microservices. https://www.martinfowler.com/microservices/
2. Microsoft Learn. Microservices architecture. https://learn.microsoft.com/en-us/dotnet/architecture/microservices/architect-microservice-container-applications/microservices-architecture
3. Microservices.io. API Gateway pattern. https://microservices.io/patterns/apigateway.html
4. PostgreSQL. About PostgreSQL. https://www.postgresql.org/about/
5. PostgreSQL Documentation. Reliability and the Write-Ahead Log. https://www.postgresql.org/docs/14/wal-reliability.html
6. Redis. What is Redis? https://redis.io/tutorials/what-is-redis/
7. Redis Documentation. Use Cases. https://redis.io/docs/latest/develop/use-cases/
8. Apache Kafka. Use Cases. https://kafka.apache.org/22/getting-started/uses/
9. Go Documentation. https://go.dev/doc/docs.html
10. Node.js Documentation. https://nodejs.org/en/docs
11. TypeScript Documentation. https://www.typescriptlang.org/docs/
12. FastAPI Features. https://fastapi.tiangolo.com/features/
13. OpenTelemetry Documentation. https://opentelemetry.io/docs/
14. Prometheus Overview. https://prometheus.io/docs/introduction/overview/
15. Expo Documentation. https://docs.expo.dev/
16. Expo Router Introduction. https://docs.expo.dev/router/introduction/
17. React Native Documentation. https://reactnative.dev/docs/getting-started
18. TanStack Query Overview. https://tanstack.com/query/latest/docs/framework/react/overview
19. Expo Notifications Documentation. https://docs.expo.dev/versions/latest/sdk/notifications/
20. Expo Location Documentation. https://docs.expo.dev/versions/latest/sdk/location/
21. MDN Web Docs. WebSocket API. https://developer.mozilla.org/en-US/docs/Web/API/WebSocket
