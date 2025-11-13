# BillingService Technical Documentation

Status: Active Development

This document provides a detailed technical overview of the `BillingService`, a core microservice for Project Sage.

## 1. Overview

The `BillingService` is a highly specialized, internal-facing Go microservice. It has a single responsibility, as defined in  **TRD Section 4.2** : to manage the debiting of "Assistance Tokens" from a user's account.

Its primary function is to:

* Provide a secure, atomic API endpoint to decrement a user's `assistance_token_balance`.
* Enforce the business rule that a user's balance cannot go below zero.

This service is **internal** and is designed to be called by other backend services (specifically, the `RequestService` during the handoff flow), not directly by the end-user's mobile app.

---

## 2. Architecture & Design

The `BillingService` follows the same robust, layered architecture as the `UserService` to ensure separation of concerns and high testability.

### Handler (`handler.go`)

* **C# Bridge:** This is your **Controller** class (e.g., `BillingController.cs`).
* **Responsibility:**
  * Defines the `POST /token/debit` API endpoint.
  * Parses the incoming JSON request (`debitRequest`).
  * Validates the `user_id` format.
  * Calls the `Service` layer.
  * Returns a JSON response (`debitResponse`) or a formatted error (e.g., `409 Conflict` for insufficient funds).

### Service (`service.go`)

* **Responsibility:**
  * Contains the business logic for billing.
  * For the MVP, this layer is a simple pass-through to the repository.
  * **Future Enhancement:** This layer is where more complex logic would live (e.g., "debit 2 tokens if user is on 'premium-plus' tier" or "allow a negative balance of -1 for 'trusted' users").

### Repository (`repository.go`)

* **Responsibility:**
  * Executes the raw SQL query to manage token balances.
  * The core of this service is its  **atomic `UPDATE` query** , which prevents race conditions and ensures a balance never drops below zero.

---

## 3. API Endpoints

This service exposes a single internal endpoint.

### `POST /token/debit`

* **Description:** Atomically decrements the token balance for a specified `user_id` by 1.
* **Fulfills:** **TRD 4.2** (`BillingService` to manage user tokens) and **TRD 5.3.4** (BillingService to debit one token).
* **Request Body:**
  **JSON**

  ```
  {
    "user_id": "a1b2c3d4-e5f6-7890-a1b2-c3d4e5f67890"
  }
  ```
* **Success Response (200 OK):**

  * Returns the new, remaining token balance for the user.

  **JSON**

  ```
  {
    "new_balance": 2
  }
  ```
* **Error Responses:**

  * `400 Bad Request`: Invalid JSON payload or malformed `user_id` (not a UUID).
  * `409 Conflict`: The debit failed because the user's balance was 0, or the `user_id` does not exist. The service returns this specific code so the calling service (like `RequestService`) can handle this business rule failure gracefully.
  * `500 Internal Server Error`: A database connection error or other unexpected panic.

---

## 4. Data Model

This is a key architectural point. The `BillingService`  **does not own any tables** .

Instead, it has explicit, limited permission to perform `UPDATE` operations on the `assistance_token_balance` column of the  **`users` table** , which is owned by the `UserService`. This adheres to the microservice principle of "single responsibility" — the `BillingService` is responsible for the *logic* of debiting, not the *storage* of the user's entire profile.

The core of this logic is the atomic SQL query:

**SQL**

```
UPDATE users
SET assistance_token_balance = assistance_token_balance - 1
WHERE user_id = $1 AND assistance_token_balance > 0
RETURNING assistance_token_balance
```

This query elegantly handles both finding the user and checking their balance in a single, thread-safe operation.

---

## 5. Configuration

The service is configured using environment variables:

| **Variable**       | **Description**                               | **Example**                              |
| ------------------------ | --------------------------------------------------- | ---------------------------------------------- |
| `DB_CONNECTION_STRING` | Standard Postgres DSN (Data Source Name).           | `postgres://user:pass@host:5432/dbname`      |
| `PORT`                 | The port for the HTTP server to listen on.          | `8081`                                       |
| `TEST_DB_URL`          | (Testing only) The DSN for the integration test DB. | `postgres://user:pass@localhost:5433/testdb` |

---

## 6. Running the Service

1. Ensure you have Go 1.21+ installed.
2. Set the required environment variables (e.g., `DB_CONNECTION_STRING`).
3. From the root `project-sage` directory, run:
   **Bash**

   ```
   go run ./cmd/billingservice/main.go
   ```
4. The server will start (e.g., `BillingService starting on port 8081`).

---

## 7. Testing

The service includes comprehensive unit and integration tests.

### Unit Tests (Service Layer)

These tests use `gomock` to mock the `Repository` and verify that the `Service` layer correctly handles both success and error cases (like `insufficient funds`) from the repository. They **do not** require a database.

**Bash**

```
# Run all tests in the package
go test ./internal/billing

# Run only the service tests
go test ./internal/billing -run TestService_
```

### Integration Tests (Repository Layer)

These tests run against a **real Postgres database** to verify the core atomic SQL query. They are critical for ensuring the billing logic is correct.

> **Warning:** These tests are destructive. They will insert and delete test data. **NEVER** run these against a development or production database.

1. Ensure a test database is running.
2. Set the `TEST_DB_URL` environment variable.
   **Bash**

   ```
   # Example for macOS/Linux
   export TEST_DB_URL="postgres://user:pass@localhost:5433/testdb"

   # Run all billing tests
   go test ./internal/billing
   ```
3. The tests will:

   * Create a test user with 3 tokens.
   * Successfully debit the balance from 3 -> 2 -> 1 -> 0.
   * Assert that attempting to debit from 0 returns the correct `insufficient funds` error.
   * Assert that attempting to debit from a non-existent user returns the same error.
