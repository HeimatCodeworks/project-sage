package request

//go:generate mockgen -destination=./clients_mock_test.go -package=request -source=clients.go BillingClient,LLMClient,ChatClient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// This file defines the clients the RequestService needs to talk to all the other services.

// BillingClient is the contract for talking to the BillingService.
type BillingClient interface {
	// DebitToken returns nil on success or an error.
	DebitToken(ctx context.Context, userID uuid.UUID) error
}

// LLMClient is what we use to talk to the LLM gateway.
type LLMClient interface {
	Summarize(ctx context.Context, twilioSID string) (string, error)
}

// ChatClient is for talking to the ChatGateway.
type ChatClient interface {
	RemoveBot(ctx context.Context, twilioSID string) error
	AddExpert(ctx context.Context, twilioSID string, expertID uuid.UUID) error
}

// httpBillingClient is the implementation for the BillingClient.
type httpBillingClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewHTTPBillingClient is the constructor
func NewHTTPBillingClient(baseURL string) BillingClient {
	return &httpBillingClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    baseURL,
	}
}

type debitRequest struct {
	UserID string `json:"user_id"`
}

func (c *httpBillingClient) DebitToken(ctx context.Context, userID uuid.UUID) error {
	reqBody, err := json.Marshal(debitRequest{UserID: userID.String()})
	if err != nil {
		return fmt.Errorf("could not marshal debit request: %w", err)
	}

	url := c.baseURL + "/token/debit"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("could not create debit http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("debit request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusConflict { //
			return fmt.Errorf("insufficient funds")
		}
		return fmt.Errorf("billing service returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}

type httpLLMClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewHTTPLLMClient is the constructor for the llm client.
func NewHTTPLLMClient(baseURL string) LLMClient {
	return &httpLLMClient{
		httpClient: &http.Client{Timeout: 15 * time.Second}, // Longer timeout for LLM
		baseURL:    baseURL,
	}
}

// DTOs for LLMGatewayService
type summarizeRequest struct {
	TwilioConversationSID string `json:"twilio_conversation_sid"`
}
type summarizeResponse struct {
	Summary string `json:"summary"`
}

// Summarize makes an http call to the LLMGatewayService.
func (c *httpLLMClient) Summarize(ctx context.Context, twilioSID string) (string, error) {
	// Create the request body
	reqBody, err := json.Marshal(summarizeRequest{TwilioConversationSID: twilioSID})
	if err != nil {
		return "", fmt.Errorf("could not marshal summarize request: %w", err)
	}

	// Create the http request
	url := c.baseURL + "/chat/summarize" // This matches llm handler
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("could not create summarize http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make the call
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("summarize request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm service returned non-200 status: %d", resp.StatusCode)
	}

	// decode the response
	var summaryResp summarizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&summaryResp); err != nil {
		return "", fmt.Errorf("could not decode summarize response: %w", err)
	}

	return summaryResp.Summary, nil
}

type httpChatClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewHTTPChatClient is the constructor for the real Chat client.
func NewHTTPChatClient(baseURL string) ChatClient {
	return &httpChatClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    baseURL,
	}
}

// DTOs for ChatGatewayService
type removeBotRequest struct {
	TwilioConversationSID string `json:"twilio_conversation_sid"`
}
type addExpertRequest struct {
	TwilioConversationSID string `json:"twilio_conversation_sid"`
	ExpertID              string `json:"expert_id"`
}

// RemoveBot makes an http call to the ChatGatewayService.
func (c *httpChatClient) RemoveBot(ctx context.Context, twilioSID string) error {
	reqBody, err := json.Marshal(removeBotRequest{TwilioConversationSID: twilioSID})
	if err != nil {
		return fmt.Errorf("could not marshal remove-bot request: %w", err)
	}

	url := c.baseURL + "/chat/remove-bot"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("could not create remove-bot http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("remove-bot request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chat service (remove-bot) returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}

// AddExpert makes an http call to the ChatGatewayService.
func (c *httpChatClient) AddExpert(ctx context.Context, twilioSID string, expertID uuid.UUID) error {
	reqBody, err := json.Marshal(addExpertRequest{
		TwilioConversationSID: twilioSID,
		ExpertID:              expertID.String(),
	})
	if err != nil {
		return fmt.Errorf("could not marshal add-expert request: %w", err)
	}

	url := c.baseURL + "/chat/add-expert"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("could not create add-expert http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("add-expert request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chat service (add-expert) returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}
