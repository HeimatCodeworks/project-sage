# RequestService Technical Documentation

Status: Active Development

This document provides a detailed technical overview of the `RequestService`, the central orchestrator for the Project Sage application.

## 1. Overview

The `RequestService` is the most critical microservice in the backend. It is a Go-based service that acts as the "brain" for the entire expert handoff flow. Its sole responsibility is to manage the complete lifecycle of an `assistance_request` from its creation by a user to its resolution by an expert.

Its primary functions are:

* **Orchestration:** Following the "LLM-Human Handoff" logic in  **TRD Section 9** .
* **Creating Requests:** Responding to user requests by calling the `BillingService` to debit a token, the `LLMGatewayService` to generate a summary, and the `ChatGatewayService` to manage participants ( **TRD 5.3.4** ).
* **Managing the Queue:** Providing a list of "pending" requests for the Expert App ( **TRD 5.4.6** ).
* **Handling Acceptance:** Assigning an expert to a request and adding them to the chat ( **TRD 5.4.7** ).
* **Resolving Requests:** Marking requests as "resolved" and triggering the rating prompt ( **TRD 5.4.11** ).

This service **owns** the `assistance_requests` and `expert_ratings` tables ( **TRD 8.1** ).

---

## 2. Architecture & Design

The `RequestService` uses a layered architecture but extends it to include  **service-to-service clients** , as it is an orchestrator.

### Handler (`handler.go`)

* **C# Bridge:** This is your **Controller** (`RequestsController.cs`).
* **Responsibility:**
  * Defines all API endpoints for both the User and Expert apps.
  * Parses request DTOs (e.g., `CreateRequestPayload`).
  * Reads `UserID` or `ExpertID` from the auth context.
  * Calls the `Service` layer.
  * Returns formatted JSON responses or errors (e.g., `402 Payment Required` if the `BillingService` call fails).

### Service (`service.go`)

* **C# Bridge:** This is the **Service/Manager** layer (`IRequestService.cs`).
* **Responsibility:**
  * This is the **orchestrator** and the core brain of the service.
  * It contains all the business logic for the handoff flow ( **TRD 9** ).
  * It **calls** the client interfaces (`BillingClient`, `LLMClient`, `ChatClient`) in the correct sequence.
  * It **calls** its own `Repository` to persist state changes (`pending` -> `active` -> `resolved`).

### Clients (`clients.go`)

* **C# Bridge:** These are your typed `HttpClient` wrappers (e.g., `IBillingApiClient`).
* **Responsibility:**
  * Defines interfaces for each *external* service this orchestrator depends on.
  * Includes `httpBillingClient`, `stubLLMClient`, etc.
  * These clients are responsible for the "how" of making the HTTP call (marshalling JSON, setting headers, handling HTTP errors).

### Repository (`repository.go`)

* **C# Bridge:** This is the **Repository** (`IRequestRepository.cs`).
* **Responsibility:**
  * Handles all direct database communication for the `assistance_requests` and `expert_ratings` tables.
  * Contains all SQL queries (`INSERT`, `UPDATE`, `SELECT`).
  * Includes concurrency-safe `UPDATE` queries (e.g., `...WHERE status = 'pending'`) to prevent race conditions.

---

## 3. API Endpoints

This service exposes endpoints for both the User and Expert applications.

### User App Endpoints

#### `POST /request/create`

* **Description:** Initiates a new expert help request. This is the start of the orchestration flow.
* **Fulfills:**  **TRD 5.3.4** ,  **TRD 9 (Step 4)** .
* **Request Body:**
  **JSON**

  ```
  {
    "twilio_conversation_sid": "CH...SID"
  }
  ```
* **Success Response (201 Created):**

  * Returns the newly created `assistance_request` object.

  **JSON**

  ```
  {
    "request_id": "a1b2c3d4-...",
    "user_id": "e5f6g7h8-...",
    "status": "pending",
    "llm_summary": "User needs help with their Wi-Fi.",
    "twilio_conversation_sid": "CH...SID",
    "created_at": "2025-11-13T17:39:40Z",
    ...
  }
  ```
* **Error Responses:**

  * `400 Bad Request`: Invalid payload.
  * `401 Unauthorized`: No valid user auth.
  * `402 Payment Required`: The `BillingService` call failed due to insufficient tokens.
  * `500 Internal Server Error`: `LLMGateway` failed or database error.

#### `POST /request/rate`

* **Description:** Submits a 1-5 star rating for a completed request.
* **Fulfills:**  **TRD 5.4.4** .
* **Request Body:**
  **JSON**

  ```
  {
    "request_id": "a1b2c3d4-...",
    "expert_id": "e5f6g7h8-...",
    "score": 5
  }
  ```
* **Success Response (200 OK):** `{"status": "rating received"}`

---

### Expert App Endpoints

#### `GET /request/pending`

* **Description:** Fetches the list of all pending requests for the expert queue, sorted by wait time.
* **Fulfills:**  **TRD 5.4.6** .
* **Success Response (200 OK):**

  * Returns an array of `assistance_request` objects.

  **JSON**

  ```
  [
    {
      "request_id": "a1b2c3d4-...",
      "user_id": "e5f6g7h8-...",
      "status": "pending",
      ...
    },
    ...
  ]
  ```

#### `POST /request/accept`

* **Description:** Allows an expert to accept a request, assigning it to them and changing its status to "active".
* **Fulfills:**  **TRD 5.4.7** ,  **TRD 9 (Step 8)** .
* **Request Body:**
  **JSON**

  ```
  {
    "request_id": "a1b2c3d4-..."
  }
  ```
* **Success Response (200 OK):**

  * Returns the updated `assistance_request` object.
* **Error Responses:**

  * `409 Conflict`: The request was already accepted by another expert (handled by the DB).

#### `POST /request/resolve`

* **Description:** Marks an "active" request as "resolved," completing its lifecycle.
* **Fulfills:**  **TRD 5.4.11** .
* **Request Body:**
  **JSON**

  ```
  {
    "request_id": "a1b2c3d4-..."
  }
  ```
* **Success Response (200 OK):** `{"status": "resolved"}`

---

## 4. Orchestration Flows (TRD 9)

### Create Request Flow (User)

1. **Handler** receives `POST /request/create`.
2. **Service** is called with `UserID` and `TwilioSID`.
3. **Service** calls `BillingClient.DebitToken(UserID)`.
   * *If this fails (e.g., 409 Conflict), the flow stops and returns a `402 Payment Required` error.*
4. **Service** calls `LLMClient.Summarize(TwilioSID)`.
   * *If this fails, the flow stops and returns a `500` error (token is *not* refunded in MVP).*
5. **Service** calls `Repository.CreateRequest(...)` to save the "pending" request to Postgres.
6. **Service** calls `ChatClient.RemoveBot(TwilioSID)`.
7. **Service** returns the new request object to the handler.

### Accept Request Flow (Expert)

1. **Handler** receives `POST /request/accept`.
2. **Service** is called with `RequestID` and `ExpertID`.
3. **Service** calls `Repository.AcceptRequest(...)`, which atomically sets `status='active'` and `expert_id=...` *only if* `status` is currently 'pending'.
   * *If this fails (0 rows affected), the flow stops and returns a `409 Conflict` error.*
4. **Service** calls `Repository.GetRequestByID(...)` to fetch the `TwilioConversationSID`.
5. **Service** calls `ChatClient.AddExpert(TwilioSID, ExpertID)`.
   * *If this fails, the flow stops and returns a `500` error (the request is in a bad state).*
6. **Service** returns the updated request object.

---

## 5. Data Model

This service is the exclusive owner of two tables as defined in  **TRD 8.1** :

* **`assistance_requests`** : The primary table for managing the request lifecycle. It contains foreign keys to `users(user_id)` and `experts(expert_id)`.
* **`expert_ratings`** : Stores the 1-5 star ratings. It is linked via `request_id`, `user_id`, and `expert_id`.

---

## 6. Configuration

The service is configured using environment variables:

| **Variable**       | **Description**                               | **Example**                              |
| ------------------------ | --------------------------------------------------- | ---------------------------------------------- |
| `DB_CONNECTION_STRING` | Standard Postgres DSN.                              | `postgres://user:pass@host:5432/dbname`      |
| `PORT`                 | The port for the HTTP server.                       | `8082`                                       |
| `BILLING_SERVICE_URL`  | Base URL for the `BillingService`.                | `http://billingservice:8081`                 |
| `LLM_SERVICE_URL`      | Base URL for the `LLMGatewayService`.             | `http://llmgateway:8083`                     |
| `CHAT_SERVICE_URL`     | Base URL for the `ChatGatewayService`.            | `http://chatgateway:8084`                    |
| `TEST_DB_URL`          | (Testing only) The DSN for the integration test DB. | `postgres://user:pass@localhost:5433/testdb` |

---

## 7. Running the Service

1. Ensure you have Go 1.21+ installed.
2. Set all required environment variables (e.g., `DB_CONNECTION_STRING`, `BILLING_SERVICE_URL`).
3. From the root `project-sage` directory, run:
   **Bash**

   ```
   go run ./cmd/requestservice/main.go
   ```
4. The server will start (e.g., `RequestService starting on port 8082`).

---

## 8. Testing

The service has extensive unit and integration tests.

### Unit Tests (Service Layer)

These are the most important tests. They use `gomock` to create mocks for **all** dependencies: `Repository`, `BillingClient`, `LLMClient`, and `ChatClient`. They test the orchestration logic in isolation, verifying, for example, that a call to `CreateRequest` fails immediately if `BillingClient.DebitToken` returns an error.

**Bash**

```
# Run all tests in the package
go test ./internal/request

# Run only the service tests
go test ./internal/request -run TestService_
```

### Integration Tests (Repository Layer)

These tests run against a **real Postgres database** to verify the SQL queries. They are critical for validating foreign key relationships and the atomic `UPDATE` logic.

> **Warning:** These tests require prerequisite data. The `TestMain` function **must** create a test `user` and a test `expert` in the database *before* tests can run, or all `INSERT` queries on `assistance_requests` will fail due to foreign key constraints.

**Bash**

```
# Set the test DB variable
export TEST_DB_URL="postgres://user:pass@localhost:5433/testdb"

# Run all tests
go test ./internal/request
```
