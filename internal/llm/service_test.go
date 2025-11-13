package llm

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"
)

// setupMocks is a helper to create all mocks for our service.
func setupMocks(t *testing.T) (context.Context, *MockGeminiClient, *MockChatGatewayClient, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	return context.Background(),
		NewMockGeminiClient(ctrl),
		NewMockChatGatewayClient(ctrl),
		ctrl
}

// TestService_SocialChat_Success tests the happy path for the social chat.
func TestService_SocialChat_Success(t *testing.T) {
	ctx, mockGemini, mockChat, ctrl := setupMocks(t)
	defer ctrl.Finish()

	history := []*ChatMessage{{Role: "user", Content: "Hello"}}
	expectedResponse := &ChatMessage{Role: "model", Content: "Hi there!"}

	// GeminiClient should be called with the history
	mockGemini.EXPECT().
		GenerateContent(ctx, history).
		Return(expectedResponse, nil).
		Times(1)

	// We don't expect the ChatGatewayClient to be called
	mockChat.EXPECT().GetChatHistory(gomock.Any(), gomock.Any()).Times(0)

	// Call the service
	s := NewService(mockGemini, mockChat)
	resp, err := s.SocialChat(ctx, history)

	if err != nil {
		t.Fatalf("SocialChat() returned unexpected error: %v", err)
	}
	if resp.Content != expectedResponse.Content {
		t.Errorf("want response '%s', got '%s'", expectedResponse.Content, resp.Content)
	}
}

// TestService_SummarizeChatHistory_Success tests the happy path for summarization.
func TestService_SummarizeChatHistory_Success(t *testing.T) {
	ctx, mockGemini, mockChat, ctrl := setupMocks(t)
	defer ctrl.Finish()

	twilioSID := "CH-123"
	mockHistory := []*ChatMessage{{Role: "user", Content: "My Wi-Fi is broken."}}
	expectedSummary := "User needs help with Wi-Fi."

	// We expect a specific sequence of calls.
	gomock.InOrder(
		// The service must call the ChatGatewayClient first.
		mockChat.EXPECT().
			GetChatHistory(ctx, twilioSID).
			Return(mockHistory, nil).
			Times(1),

		//the service must then call the GeminiClient with the history.
		mockGemini.EXPECT().
			Summarize(ctx, mockHistory).
			Return(expectedSummary, nil).
			Times(1),
	)

	// We don't expect GenerateContent to be called.
	mockGemini.EXPECT().GenerateContent(gomock.Any(), gomock.Any()).Times(0)

	// Call the service
	s := NewService(mockGemini, mockChat)
	summary, err := s.SummarizeChatHistory(ctx, twilioSID)

	if err != nil {
		t.Fatalf("SummarizeChatHistory() returned unexpected error: %v", err)
	}
	if summary != expectedSummary {
		t.Errorf("want summary '%s', got '%s'", expectedSummary, summary)
	}
}

// TestService_SummarizeChatHistory_ChatGatewayError tests when the first step fails.
func TestService_SummarizeChatHistory_ChatGatewayError(t *testing.T) {
	ctx, mockGemini, mockChat, ctrl := setupMocks(t)
	defer ctrl.Finish()

	twilioSID := "CH-123"
	expectedErr := fmt.Errorf("chat gateway is down")

	// The ChatGatewayClient fails.
	mockChat.EXPECT().
		GetChatHistory(ctx, twilioSID).
		Return(nil, expectedErr).
		Times(1)

	// The GeminiClient shouldn't be called if ChatGatewayClient fails.
	mockGemini.EXPECT().Summarize(gomock.Any(), gomock.Any()).Times(0)

	// Call the service
	s := NewService(mockGemini, mockChat)
	_, err := s.SummarizeChatHistory(ctx, twilioSID)

	if err == nil {
		t.Fatal("SummarizeChatHistory() expected an error but got nil")
	}
	if err.Error() != "could not fetch chat history from ChatGateway: chat gateway is down" {
		t.Errorf("wrong error message: %v", err)
	}
}
