# LLMGatewayService Technical Documentation

Status: Active Development

This document provides a detailed technical overview of the `LLMGatewayService`, a core microservice for Project Sage.

## 1. Overview

The `LLMGatewayService` is a stateless, internal-facing Go microservice that acts as a "facade" or "unified interface" for all interactions with the Google Gemini API, as defined in  **TRD Section 4.2** .

It has two primary responsibilities:

1. **Social Chat:** Providing a direct pass-through endpoint (`/chat/social`) for the User App to power the general-purpose "Social Chat" feature ( **TRD U-2.1** ).
2. **Summarization:** Providing an internal endpoint (`/chat/summarize`) to be called by the `RequestService` during the expert handoff flow. This endpoint is responsible for fetching a chat history and generating a summary ( **TRD 5.3.4** ,  **TRD 9** ).

This service does not own any database tables. It is purely a coordination and proxy layer.

---

## 2. Architecture & Design

This service follows our standard layered architecture (`Handler`, `Service`), but it  **does not have a Repository layer** . Instead, its service layer depends on *Clients* to interact with other services.

### Handler (`handler.go`)

* **Responsibility:**
  * Defines the HTTP routes (`/chat/social`, `/chat/summarize`).
  * Parses incoming JSON DTOs (e.g., `socialChatRequest`).
  * Calls the `Service` layer.
  * Serializes response objects (e.g., `summarizeResponse`) back to JSON.

### Service (`service.go`)

* **Responsibility:**
  * Contains the core orchestration logic.
  * `SocialChat`: A simple pass-through to the `GeminiClient`.
  * `SummarizeChatHistory`: The main orchestration, which first calls the `ChatGatewayClient` to fetch a chat history, and *then* passes that history to the `GeminiClient` to generate a summary.

### Clients (`clients.go`)

* **Responsibility:**
  * Defines the interfaces for all external dependencies, allowing for mocking and testing.
  * `GeminiClient`: An interface for a client that talks to the external  **Google Gemini API** .
  * `ChatGatewayClient`: An interface for an *internal* client that talks to our own `ChatGatewayService` to fetch chat histories.

---

## 3. API Endpoints

### User App Endpoint

#### `POST /chat/social`

* **Description:** Forwards a chat history to the LLM for a conversational response.
* **Fulfills:**  **TRD U-2.1** .
* Request Body:
  JSON
  **JSON**

  ```
  {
    "history": [
      { "role": "user", "content": "Hello" },
      { "role": "model", "content": "Hi there!" },
      { "role": "user", "content": "What's the weather?" }
    ]
  }
  ```
* **Success Response (200 OK):**

  * Returns the single, new message object from the model.
    JSON

  **JSON**

  ```
  {
    "role": "model",
    "content": "I'm not connected to live weather data, but I'm happy to chat!"
  }
  ```

### Internal Endpoint

#### `POST /chat/summarize`

* **Description:** An internal endpoint called by the `RequestService` to summarize a chat history from a `twilio_conversation_sid`.
* **Fulfills:**  **TRD 4.2** ,  **TRD 5.3.4** ,  **TRD 9** .
* Request Body:
  JSON
  **JSON**

  ```
  {
    "twilio_conversation_sid": "CH...SID"
  }
  ```
* **Success Response (200 OK):**

  * Returns the generated summary string.
    JSON

  **JSON**

  ```
  {
    "summary": "User needs help with their Wi-Fi."
  }
  ```
* **Error Responses:**

  * `400 Bad Request`: Invalid JSON payload.
  * `500 Internal Server Error`: The `ChatGatewayService` failed or the `GeminiClient` failed.

---

## 4. Orchestration Flows

### Summarize Chat History Flow (TRD 9)

This is the service's most critical orchestration, triggered by the `RequestService`.

1. **Handler** receives `POST /chat/summarize` with a `TwilioConversationSID`.
2. **Service** is called with the `TwilioConversationSID`.
3. **Service** calls `ChatGatewayClient.GetChatHistory(ctx, twilioSID)`.
   * *If this fails, the flow stops and returns a 500 error.*
4. **Service** receives a `[]*ChatMessage` (the history) from the client.
5. **Service** calls `GeminiClient.Summarize(ctx, history)`.
   * *If this fails, the flow stops and returns a 500 error.*
6. **Service** receives a `string` (the summary) from the client.
7. **Handler** returns the summary string in a JSON object.

---

## 5. Data Model

This service  **does not own any tables** . It is a stateless facade that proxies requests to other services (the `ChatGatewayService` and the external `Gemini API`).

---

## 6. Configuration

The service is configured using environment variables:

| **Variable**   | **Description**                             | **Example**           |
| -------------------- | ------------------------------------------------- | --------------------------- |
| `PORT`             | The port for the HTTP server.                     | `8083`                    |
| `CHAT_GATEWAY_URL` | Base URL for the internal `ChatGatewayService`. | `http://chatgateway:8084` |
| `GEMINI_API_KEY`   | API Key for the Google Gemini API.                | `AIza...`                 |

---

## 7. Running the Service

1. Ensure you have Go 1.21+ installed.
2. Set all required environment variables.
3. From the root project-sage directory, run:
   Bash
   **Bash**

   ```
   go run ./cmd/llmgatewayservice/main.go
   ```
4. The server will start (e.g., `LLMGatewayService starting on port 8083`).

---

## 8. Testing

This service includes comprehensive unit tests.

### Unit Tests (Service Layer)

These tests use `gomock` to create mocks for the `GeminiClient` and `ChatGatewayClient` interfaces. They verify that the `service.go` orchestration logic is correct, for example:

* Verifying that `SummarizeChatHistory` calls the `ChatGatewayClient`  *first* , then the `GeminiClient`.
* Verifying that `SocialChat` *only* calls the `GeminiClient`.
* Verifying that failures in either client are handled and bubbled up correctly.

This service has no repository, so it does not have repository-level integration tests.

**Bash**

**Bash**

```
# Run all tests in the package
go test ./internal/llm
```
