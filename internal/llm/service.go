package llm

//go:generate mockgen -destination=./service_mock_test.go -package=llm -source=service.go Service

import (
	"context"
	"fmt"
)

// Service defines the business logic for the llm Gateway.
type Service interface {
	// SocialChat sends a list of messages to the llm for response
	SocialChat(ctx context.Context, history []*ChatMessage) (*ChatMessage, error)

	// SummarizeChatHistory fetches history from a Twilio SID and summarizes it.
	SummarizeChatHistory(ctx context.Context, twilioSID string) (string, error)
}

// service is the concrete implementation of the Service interface.
type service struct {
	gemini GeminiClient      // client for the external Gemini API
	chat   ChatGatewayClient // Client for the internal ChatGatewayService
}

// NewService is the constructor for the LLMGatewayService.
func NewService(gemini GeminiClient, chat ChatGatewayClient) Service {
	return &service{
		gemini: gemini,
		chat:   chat,
	}
}

// SocialChat implements the Service interface.
func (s *service) SocialChat(ctx context.Context, history []*ChatMessage) (*ChatMessage, error) {
	// For social chat we pass the history directly to the gemini client.
	response, err := s.gemini.GenerateContent(ctx, history)
	if err != nil {
		return nil, fmt.Errorf("gemini client failed: %w", err)
	}
	return response, nil
}

// SummarizeChatHistory implements the Service interface.
func (s *service) SummarizeChatHistory(ctx context.Context, twilioSID string) (string, error) {
	// This is the key orchestration flow for summarization.

	// Fetch the chat history using Twilio SID.
	history, err := s.chat.GetChatHistory(ctx, twilioSID)
	if err != nil {
		return "", fmt.Errorf("could not fetch chat history from ChatGateway: %w", err)
	}

	// Pass that history to the Gemini client to summarize.
	summary, err := s.gemini.Summarize(ctx, history)
	if err != nil {
		return "", fmt.Errorf("gemini client failed to summarize: %w", err)
	}

	return summary, nil
}
