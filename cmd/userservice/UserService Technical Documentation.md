# UserService Technical Documentation

Status: Active Development

This document provides a detailed technical overview of the `UserService`, one of the core microservices for Project Sage.

## 1. Overview

The `UserService` is a Go-based microservice responsible for all operations related to user and expert profiles. Its primary functions are:

* **User Onboarding:** Creating a new user profile in the database after they have successfully authenticated with Firebase ( **TRD U-1.1** ).
* **Profile Management:** Providing endpoints to read and update user profile information, such as display name and photo ( **TRD U-1.2** ).
* **Token & Tier Visibility:** Serving as the source of truth for a user's `membership_tier` and `assistance_token_balance` ( **TRD U-1.3** ).

This service **owns** the `users` and `experts` tables in the Postgres database ( **TRD 8.1** ). It does **not** handle authentication logic (e.g., password checks, token generation); that is fully delegated to **Firebase Authentication** ( **TRD 7** ).

---

## 2. Architecture & Design

The `UserService` is built using a layered architecture, which you'll find very similar to a modern ASP.NET Core Web API project. This separation of concerns is key for maintainability and, most importantly, for testability.

The code is organized into three distinct layers within the `/internal/user` package:

### Handler (`handler.go`)

* **Responsibility:**
  * Defines the HTTP routes using the `chi` router.
  * Parses incoming JSON request bodies (DTOs).
  * Validates input (e.g., checks for missing fields).
  * Calls the `Service` layer.
  * Serializes response objects (or errors) back to JSON.
  * Interacts directly with `net/http` types (`http.ResponseWriter`, `*http.Request`).

### Service (`service.go`)

* **Responsibility:**
  * Contains all business logic. **This is the most important layer.**
  * Orchestrates data operations by calling the `Repository`.
  * Example logic: When registering a new user, this layer sets the default `membership_tier` and `assistance_token_balance` ( **TRD 8.1** ).
  * It is completely "database-agnostic" and "HTTP-agnostic." It operates purely on Go types.

### Repository (`repository.go`)

* **Responsibility:**
  * Handles all database communication for the `users` table.
  * Contains all SQL queries.
  * Maps database rows to and from our `domain.User` struct.
  * In our case, it uses the standard `database/sql` package with the `pgx` driver.

These layers are wired together in `/cmd/userservice/main.go` using constructor-based dependency injection.

---

## 3. API Endpoints

All endpoints are implicitly prefixed by the API gateway (e.g., `/api/v1`). Authentication is handled by middleware (not shown here) that validates a Firebase JWT and makes the `firebase_auth_id` available to the handler.

### `POST /users/register`

* **Description:** Creates a new user profile after successful Firebase signup. This endpoint is intended to be called by the client app once, immediately after Firebase confirms a new user.
* **Fulfills:**  **TRD U-1.1** .
* **Request Body:**
  **JSON**

  ```
  {
    "display_name": "Jane Doe",
    "profile_image_url": "https://example.com/images/jane.png"
  }
  ```
* **Success Response (201 Created):**

  * Returns the complete user object, including the new server-generated `user_id` (UUID) and default token/tier.

  **JSON**

  ```
  {
    "user_id": "a1b2c3d4-e5f6-7890-a1b2-c3d4e5f67890",
    "display_name": "Jane Doe",
    "profile_image_url": "https://example.com/images/jane.png",
    "membership_tier": "free",
    "assistance_token_balance": 3
  }
  ```
* **Error Responses:**

  * `400 Bad Request`: Invalid or missing JSON payload.
  * `401 Unauthorized`: No valid Firebase token was provided.
  * `500 Internal Server Error`: Database error or other server logic failure.

### `GET /users/profile`

* **Description:** Fetches the profile for the currently authenticated user.
* **Fulfills:**  **TRD U-1.2** ,  **U-1.3** .
* **Request Body:** None.
* **Success Response (200 OK):**

  * Returns the full user profile object.

  **JSON**

  ```
  {
    "user_id": "a1b2c3d4-e5f6-7890-a1b2-c3d4e5f67890",
    "display_name": "Jane Doe",
    "profile_image_url": "https://example.com/images/jane.png",
    "membership_tier": "free",
    "assistance_token_balance": 3
  }
  ```
* **Error Responses:**

  * `401 Unauthorized`: No valid Firebase token was provided.
  * `404 Not Found`: A valid token was provided, but no corresponding profile exists in the `users` table (e.g., the `register` step was missed).
  * `500 Internal Server Error`: Database error.

---

## 4. Data Model

This service is the exclusive owner of the `users` and `experts` tables, as defined in  **TRD Section 8.1** .

* **`users` Table:** Stores standard user information.
* **`experts` Table:** Stores internal support staff information ( **TRD 3. User Roles** ).

**Key Design Point:** The `firebase_auth_id` (a string from Firebase) is the immutable foreign key linking our system to the auth provider. The `user_id` (a `UUID` generated by our service) is the **primary key** used for all *internal* database relations (e.g., linking a `user` to an `assistance_request`).

---

## 5. Configuration

The service is configured using environment variables:

| **Variable**       | **Description**                               | **Example**                              |
| ------------------------ | --------------------------------------------------- | ---------------------------------------------- |
| `DB_CONNECTION_STRING` | Standard Postgres DSN (Data Source Name).           | `postgres://user:pass@host:5432/dbname`      |
| `PORT`                 | The port for the HTTP server to listen on.          | `8080`                                       |
| `TEST_DB_URL`          | (Testing only) The DSN for the integration test DB. | `postgres://user:pass@localhost:5433/testdb` |

---

## 6. Running the Service

1. Ensure you have Go 1.21+ installed.
2. Set the required environment variables (e.g., `DB_CONNECTION_STRING`).
3. From the root `project-sage` directory, run:
   **Bash**

   ```
   go run ./cmd/userservice/main.go
   ```
4. The server will start (e.g., `UserService starting on port 8080`).

---

## 7. Testing

The service has two types of tests. You can run them all from the `/internal/user` directory.

### Unit Tests (Service Layer)

These tests check the business logic of the `service.go` file  *in isolation* . They use `gomock` to mock the `Repository` interface and **do not** require a database.

**Bash**

```
# Run all tests in the package
go test ./internal/user

# Run only the service tests
go test ./internal/user -run TestService_
```

### Integration Tests (Repository Layer)

These tests run against a **real Postgres database** to verify the SQL queries in `repository.go` are correct.

> **Warning:** These tests are destructive. They will **DELETE all data** from the `users` table before each run. **NEVER** run these against a development or production database.

1. Ensure a test database is running (e.g., a local Postgres instance or a Docker container).
2. Set the `TEST_DB_URL` environment variable.
   **Bash**

   ```
   # Example for macOS/Linux
   export TEST_DB_URL="postgres://user:pass@localhost:5433/testdb"

   # Run only the repository tests
   go test ./internal/user -run TestCreateUser
   ```
