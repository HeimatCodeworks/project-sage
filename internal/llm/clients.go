package llm

//go:generate mockgen -destination=./clients_mock_test.go -package=llm -source=clients.go

import (
	"context"
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
