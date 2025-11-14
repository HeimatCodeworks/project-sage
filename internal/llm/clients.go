package llm

//go:generate mockgen -destination=./clients_mock_test.go -package=llm -source=clients.go

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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

// httpChatGatewayClient is the real implementation for the ChatGatewayClient.
type httpChatGatewayClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewHTTPChatGatewayClient is the constructor for the real client
func NewHTTPChatGatewayClient(baseURL string) ChatGatewayClient {
	return &httpChatGatewayClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    baseURL,
	}
}

// This DTO must match the chat.Message struct from the ChatGatewayService
type chatServiceMessage struct {
	SID       string    `json:"sid"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// GetChatHistory makes http call to the ChatGatewayService.
func (c *httpChatGatewayClient) GetChatHistory(ctx context.Context, twilioSID string) ([]*ChatMessage, error) {
	// This matches the ChatGatewayService handler: /chat/history/{sid}
	url := fmt.Sprintf("%s/chat/history/%s", c.baseURL, twilioSID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create get-history http request: %w", err)
	}

	// Make the call
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get-history request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chat service (get-history) returned non-200 status: %d", resp.StatusCode)
	}

	// decode the response
	var chatHistory []*chatServiceMessage
	if err := json.NewDecoder(resp.Body).Decode(&chatHistory); err != nil {
		return nil, fmt.Errorf("could not decode chat history response: %w", err)
	}

	// This service's domain should not be coupled to the chat service's domain.
	llmHistory := make([]*ChatMessage, len(chatHistory))
	for i, msg := range chatHistory {

		// Map the author to either "user" or "model" (for the llm)
		var role string
		if msg.Author == "LLM_BOT_IDENTITY" {
			role = "model"
		} else {
			// For now, treat all other participants as the user
			role = "user"
		}

		llmHistory[i] = &ChatMessage{
			Role:    role,
			Content: msg.Content,
		}
	}

	return llmHistory, nil
}
