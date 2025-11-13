package chat

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"
)

// setupHandlerTest initializes a router, mock service, and handler for testing.
func setupHandlerTest(t *testing.T) (*chi.Mux, *MockService, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockService := NewMockService(ctrl)

	handler := NewHandler(mockService)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	return r, mockService, ctrl
}

func TestHandleGenerateToken_UserSuccess(t *testing.T) {
	r, mockService, ctrl := setupHandlerTest(t)
	defer ctrl.Finish()

	expectedToken := "fake-user-token"

	// expect the service's GenerateUserToken to be called
	mockService.EXPECT().
		GenerateUserToken(gomock.Any(), gomock.Any()).
		Return(expectedToken, nil).
		Times(1)

	// Using the fake query param auth
	req := httptest.NewRequest("POST", "/chat/token?user_id=...-...", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var respBody tokenResponse
	json.NewDecoder(rr.Body).Decode(&respBody)
	if respBody.Token != expectedToken {
		t.Errorf("Expected token '%s', got '%s'", expectedToken, respBody.Token)
	}
}

func TestHandleAddExpert_Success(t *testing.T) {
	r, mockService, ctrl := setupHandlerTest(t)
	defer ctrl.Finish()

	reqBody := addExpertRequest{
		TwilioConversationSID: "CH123",
		ExpertID:              "a1b2c3d4-e5f6-7890-a1b2-c3d4e5f67890",
	}

	// Expect AddExpert to be called
	mockService.EXPECT().
		AddExpert(gomock.Any(), "CH123", gomock.Any()).
		Return(nil).
		Times(1)

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/chat/add-expert", bytes.NewBuffer(bodyBytes))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHandleGetChatHistory_Success(t *testing.T) {
	r, mockService, ctrl := setupHandlerTest(t)
	defer ctrl.Finish()

	sid := "CH123"
	expectedHistory := []*Message{{Content: "Hello"}}

	// Expect GetChatHistory to be called with the SID from the URL
	mockService.EXPECT().
		GetChatHistory(gomock.Any(), sid).
		Return(expectedHistory, nil).
		Times(1)

	req := httptest.NewRequest("GET", "/chat/history/"+sid, nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var respBody []*Message
	json.NewDecoder(rr.Body).Decode(&respBody)
	if len(respBody) != 1 || respBody[0].Content != "Hello" {
		t.Errorf("Unexpected history response")
	}
}
