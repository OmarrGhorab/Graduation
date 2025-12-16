# Requirements Document

## Introduction

This feature adds a `/myprofile` endpoint to the authentication service that allows authenticated users to retrieve their complete profile information. The endpoint will return comprehensive user data including account status, preferences, and profile details.

## Glossary

- **Auth Service**: The authentication microservice responsible for user management and authentication
- **Access Token**: JWT token used to authenticate API requests
- **User Profile**: Complete set of user information including personal details, account status, and preferences
- **Authenticated User**: A user who has provided a valid access token

## Requirements

### Requirement 1

**User Story:** As an authenticated user, I want to retrieve my complete profile information, so that I can view and verify my account details.

#### Acceptance Criteria

1. WHEN an authenticated user sends a GET request to /myprofile THEN the Auth Service SHALL return the user's complete profile data
2. WHEN the request includes a valid access token THEN the Auth Service SHALL extract the user ID from the token and retrieve the corresponding user record
3. WHEN the user record is found THEN the Auth Service SHALL return user data including id, name, username, email, verified status, onboarding status, role, profile image, account status, and last login timestamp
4. WHEN the user record is not found THEN the Auth Service SHALL return a 404 error with message "User not found"
5. WHEN the access token is missing or invalid THEN the Auth Service SHALL return a 401 error with message "Unauthorized"

### Requirement 2

**User Story:** As a system administrator, I want the profile endpoint to exclude sensitive data, so that security is maintained.

#### Acceptance Criteria

1. WHEN the Auth Service returns user profile data THEN the system SHALL exclude the password field from the response
2. WHEN the Auth Service returns user profile data THEN the system SHALL exclude internal fields like deletedAt from the response
3. WHEN the Auth Service returns user profile data THEN the system SHALL only include fields that are safe for client consumption

### Requirement 3

**User Story:** As a developer, I want the profile endpoint to follow existing authentication patterns, so that the codebase remains consistent.

#### Acceptance Criteria

1. WHEN implementing the profile endpoint THEN the Auth Service SHALL use the existing authentication middleware
2. WHEN implementing the profile endpoint THEN the Auth Service SHALL follow the same error handling patterns as other endpoints
3. WHEN implementing the profile endpoint THEN the Auth Service SHALL use the existing Prisma client for database access
