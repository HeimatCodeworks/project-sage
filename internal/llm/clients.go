package llm

//go:generate mockgen -destination=./clients_mock_test.go -package=llm -source=clients.go

import (
	"context"
	"fmt"
)

// GeminiClient defines the contract for an external client that talks to the Gemini API.
type GeminiClient interface {
	// GenerateContent takes a history and returns the next message from the model.
	GenerateContent(ctx context.Context, history []*ChatMessage) (*ChatMessage, error)
	// Sumarize takes a history and returns a single summary string.
	Summarize(ctx context.Context, history []*ChatMessage) (string, error)
}

// ChatGatewayClient defines the contract the client that talks to the ChatGatewayService.
type ChatGatewayClient interface {
	GetChatHistory(ctx context.Context, twilioSID string) ([]*ChatMessage, error)
}

// stubGeminiClient is a fake GeminiClient.
type stubGeminiClient struct{}

// NewStubGeminiClient creates a fake client.
func NewStubGeminiClient() GeminiClient {
	return &stubGeminiClient{}
}

func (s *stubGeminiClient) GenerateContent(ctx context.Context, history []*ChatMessage) (*ChatMessage, error) {
	// Return a canned response
	return &ChatMessage{
		Role:    "model",
		Content: "Hello! As an AI assistant, I'm happy to chat with you.",
	}, nil
}

func (s *stubGeminiClient) Summarize(ctx context.Context, history []*ChatMessage) (string, error) {
	// Return a fixed summary
	return "User needs help with their Wi-Fi.", nil
}

// stubChatGatewayClient is a fake ChatGatewayClient.
type stubChatGatewayClient struct{}

// NewStubChatGatewayClient creates a fake client.
func NewStubChatGatewayClient() ChatGatewayClient {
	return &stubChatGatewayClient{}
}

func (s *stubChatGatewayClient) GetChatHistory(ctx context.Context, twilioSID string) ([]*ChatMessage, error) {
	// Return a mock chat history.
	if twilioSID == "" {
		return nil, fmt.Errorf("twilioSID cannot be empty")
	}

	return []*ChatMessage{
		{Role: "user", Content: "Hello, my Wi-Fi isn't working."},
		{Role: "model", Content: "I see. Have you tried turning it off and on again?"},
		{Role: "user", Content: "Yes, I tried that, and it's still broken."},
	}, nil
}
