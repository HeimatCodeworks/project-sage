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

// ChatClient is for talking to the ChatGateway (talks to Twilio).
type ChatClient interface {
	RemoveBot(ctx context.Context, twilioSID string) error
	AddExpert(ctx context.Context, twilioSID string, expertID uuid.UUID) error
}

// httpBillingClient is the real implementation for the BillingClient. makes hhtp call.
type httpBillingClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewHTTPBillingClient is the constructor
func NewHTTPBillingClient(baseURL string) BillingClient {
	return &httpBillingClient{
		httpClient: &http.Client{Timeout: 5 * time.Second}, // 5 sec timeout
		baseURL:    baseURL,
	}
}

// this is the DTO we need to send to the billing service.
type debitRequest struct {
	UserID string `json:"user_id"`
}

func (c *httpBillingClient) DebitToken(ctx context.Context, userID uuid.UUID) error {
	// turn my go struct into json. this shouldn't fail...
	reqBody, err := json.Marshal(debitRequest{UserID: userID.String()})
	if err != nil {
		return fmt.Errorf("could not marshal debit request: %w", err)
	}

	// build the actual http request
	url := c.baseURL + "/token/debit"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("could not create debit http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// fire it off
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("debit request failed: %w", err)
	}
	defer resp.Body.Close()

	// check if the billing service was happy
	if resp.StatusCode != http.StatusOK {
		// We know that 409 means "no money"
		if resp.StatusCode == http.StatusConflict {
			return fmt.Errorf("insufficient funds")
		}
		// some other disaster
		return fmt.Errorf("billing service returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}

// --- STUBBED CLIENTS ---
// I havent built the other services yet, so I'm using these fake ones for now. My RequestService will get injected with these stubs until I build the real ones.

type stubLLMClient struct{}

func NewStubLLMClient() LLMClient { return &stubLLMClient{} }
func (c *stubLLMClient) Summarize(ctx context.Context, twilioSID string) (string, error) {
	// TODO: actually build the LLM Gateway and make this call it
	return "User needs help with their Wi-Fi.", nil
}

type stubChatClient struct{}

func NewStubChatClient() ChatClient { return &stubChatClient{} }
func (c *stubChatClient) RemoveBot(ctx context.Context, twilioSID string) error {
	// TODO: Connect to Chat Gateway
	return nil
}
func (c *stubChatClient) AddExpert(ctx context.Context, twilioSID string, expertID uuid.UUID) error {
	// TODO
	return nil
}
