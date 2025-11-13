# ChatGatewayService Technical Documentation

Status: Active Development

This document provides a detailed technical overview of the `ChatGatewayService`, a core microservice for Project Sage.

## 1. Overview

The `ChatGatewayService` is a stateless, internal-facing Go microservice. It acts as the exclusive "facade" for all interactions with the third-party  **Twilio Conversations API** .

Its sole responsibility is to abstract away all Twilio-specific logic from the rest of our backend. Its primary functions are:

* **Token Generation:** Providing an endpoint for the User and Expert apps to get Twilio auth tokens.
* **Participant Management:** Exposing internal endpoints for other services (like the `RequestService`) to add or remove participants from a chat (e.g., adding an expert, removing the bot).
* **History Fetching:** Providing an internal endpoint for the `LLMGatewayService` to fetch a chat history for summarization.

This service does not own any database tables and is purely a proxy and orchestration layer.

---

## 2. Architecture & Design

This service follows the standard `Handler` and `Service` layered architecture. Like the `LLMGatewayService`, it  **does not have a Repository layer** .

### Handler (`handler.go`)

* **Responsibility:**
  * Defines all HTTP routes (e.g., `/chat/token`, `/chat/add-expert`).
  * Handles auth-related logic for token generation (i.e., identifying if the caller is a User or Expert).
  * Parses incoming JSON DTOs (e.g., `addExpertRequest`).
  * Calls the `Service` layer and returns JSON responses.

### Service (`service.go`)

* **Responsibility:**
  * Contains all business logic related to chat orchestration.
  * Depends on a `TwilioClient` interface to perform its actions.
  * Implements the logic for `GenerateUserToken`, `AddExpert`, `RemoveBot`, etc., by calling the appropriate methods on the `TwilioClient`.

### Clients (`clients.go`)

* **Responsibility:**
  * Defines the `TwilioClient` interface, which is the contract for all Twilio-specific API calls.
  * This abstraction allows the service layer to be tested in isolation by mocking this interface.

---

## 3. API Endpoints

### Client-Facing Endpoint (User/Expert Apps)

#### `POST /chat/token`

* **Description:** Generates a short-lived Twilio access token for an authenticated user or expert. The app uses this token to connect directly to the Twilio SDK.
* **Fulfills:**  **TRD 4.2** .
* **Request Body:** None. Auth is handled via middleware (or placeholder query params).
* Success Response (200 OK):
  JSON
  **JSON**

  ```
  {
    "token": "ey...[a long JWT token from Twilio]"
  }
  ```

### Internal Service-to-Service Endpoints

#### `POST /chat/add-expert`

* **Description:** Called by the `RequestService` to add a specific expert to a conversation after they accept a request.
* **Fulfills:**  **TRD 9 (Step 9.b)** .
* Request Body:
  JSON
  **JSON**

  ```
  {
    "twilio_conversation_sid": "CH...SID",
    "expert_id": "a1b2c3d4-..."
  }
  ```

#### `POST /chat/remove-bot`

* **Description:** Called by the `RequestService` during the handoff flow to remove the LLM Bot from the conversation.
* **Fulfills:**  **TRD 9 (Step 5.d)** .
* Request Body:
  JSON
  **JSON**

  ```
  {
    "twilio_conversation_sid": "CH...SID"
  }
  ```

#### `GET /chat/history/{sid}`

* **Description:** Called by the `LLMGatewayService` to fetch the message history of a specific conversation for summarization. The `{sid}` is passed in the URL.
* **Fulfills:**  **TRD 4.2** .
* **Success Response (200 OK):**

  * Returns an array of message objects.
    JSON

  **JSON**

  ```
  [
    {
      "sid": "MSG_FAKE_1",
      "author": "user-uuid",
      "content": "Hello, my Wi-Fi isn't working.",
      "timestamp": "2025-11-13T15:46:00Z"
    },
    {
      "sid": "MSG_FAKE_2",
      "author": "LLM_BOT_IDENTITY",
      "content": "I see. Have you tried turning it off and on again?",
      "timestamp": "2025-11-13T15:47:00Z"
    }
  ]
  ```

---

## 4. Data Model

This service **does not own any tables** in the Postgres database, as defined in  **TRD 8.1** . It is a stateless facade that manages state within the external Twilio platform.

---

## 5. Configuration

The service is configured using environment variables, primarily for connecting to Twilio.

| **Variable**     | **Description**         | **Example** |
| ---------------------- | ----------------------------- | ----------------- |
| `PORT`               | The port for the HTTP server. | `8084`          |
| `TWILIO_ACCOUNT_SID` | Your main Twilio Account SID. | `AC...`         |
| `TWILIO_AUTH_TOKEN`  | Your main Twilio Auth Token.  | `...`           |
| `TWILIO_API_KEY`     | Twilio API Key (Chat).        | `SK...`         |
| `TWILIO_API_SECRET`  | Twilio API Secret (Chat).     | `...`           |

---

## 6. Running the Service

1. Ensure you have Go 1.21+ installed.
2. Set all required environment variables (e.g., `PORT` and `TWILIO_` keys).
3. From the root project-sage directory, run:
   Bash
   **Bash**

   ```
   go run ./cmd/chatgatewayservice/main.go
   ```
4. The server will start (e.g., `ChatGatewayService starting on port 8084`).

---

## 8. Testing

This service includes unit tests for its service layer.

### Unit Tests (Service Layer)

These tests use `gomock` to create a mock of the `TwilioClient` interface. They test the orchestration logic in `service.go` in isolation, verifying, for example:

* That `CreateConversation` correctly calls `twilio.CreateConversation`, then `twilio.AddParticipant` for the user, and `twilio.AddParticipant` for the bot.
* That `GenerateUserToken` calls `twilio.GenerateToken` with the user's UUID.
* That `RemoveBot` calls `twilio.RemoveParticipant` with the correct bot identity.

**Bash**

**Bash**

```
# Run all tests in the package
go test ./internal/chat
```
