package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"
)

// setupHandlerTest initializes a router, mock service, and handler for testing
func setupHandlerTest(t *testing.T) (*chi.Mux, *MockService, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockService := NewMockService(ctrl)

	handler := NewHandler(mockService)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	return r, mockService, ctrl
}

func TestHandleSocialChat_Success(t *testing.T) {
	r, mockService, ctrl := setupHandlerTest(t)
	defer ctrl.Finish()

	// Define the mock request and response
	reqBody := socialChatRequest{
		History: []*ChatMessage{{Role: "user", Content: "Hello"}},
	}
	respMsg := &ChatMessage{Role: "model", Content: "Hi!"}

	// Set up the service mock
	mockService.EXPECT().
		SocialChat(gomock.Any(), gomock.Any()). // We could be more specific with the matcher
		Return(respMsg, nil).
		Times(1)

	// Create the http test request
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/chat/social", bytes.NewBuffer(bodyBytes))
	rr := httptest.NewRecorder()

	// Serve request
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var respBody ChatMessage
	if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
		t.Fatalf("Could not decode response: %v", err)
	}

	if respBody.Content != "Hi!" {
		t.Errorf("Expected response content 'Hi!', got '%s'", respBody.Content)
	}
}

func TestHandleSummarizeChat_Success(t *testing.T) {
	r, mockService, ctrl := setupHandlerTest(t)
	defer ctrl.Finish()

	// Define request and response
	reqBody := summarizeRequest{TwilioConversationSID: "CH-123"}
	expectedSummary := "User needs Wi-Fi help."

	// Set up mock
	mockService.EXPECT().
		SummarizeChatHistory(gomock.Any(), "CH-123").
		Return(expectedSummary, nil).
		Times(1)

	// Create request
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/chat/summarize", bytes.NewBuffer(bodyBytes))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var respBody summarizeResponse
	if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
		t.Fatalf("Could not decode response: %v", err)
	}

	if respBody.Summary != expectedSummary {
		t.Errorf("Expected summary '%s', got '%s'", expectedSummary, respBody.Summary)
	}
}

func TestHandleSocialChat_ServiceError(t *testing.T) {
	r, mockService, ctrl := setupHandlerTest(t)
	defer ctrl.Finish()

	reqBody := socialChatRequest{
		History: []*ChatMessage{{Role: "user", Content: "Hello"}},
	}

	// Set up mock to return an error
	mockService.EXPECT().
		SocialChat(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("model is down")).
		Times(1)

	// Create request
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/chat/social", bytes.NewBuffer(bodyBytes))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	// Check for the error message
	var errBody map[string]string
	json.NewDecoder(rr.Body).Decode(&errBody)
	if errBody["error"] != "Could not process chat" {
		t.Errorf("Expected error '%s', got '%s'", "Could not process chat", errBody["error"])
	}
}
