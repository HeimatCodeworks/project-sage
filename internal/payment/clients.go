package payment

//go:generate mockgen -destination=./clients_mock_test.go -package=payment -source=clients.go

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"project-sage/internal/domain"
	"time"

	"github.com/google/uuid"
)

// --- Client Interfaces ---

// BillingClient is the client for the internal BillingService.
type BillingClient interface {
	// alls POST /token/add
	CreditToken(ctx context.Context, userID uuid.UUID, amount int) (int, error)
}

// UserClient is the client for the internal UserService.
type UserClient interface {
	// GetUserProfile fetches a user's profile.
	GetUserProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

// AppleClient is for Apple's IAP verification API.
type AppleClient interface {
	VerifyReceipt(ctx context.Context, receipt string) (string, error) // Returns (productID, error)
}

// GoogleClient is for Google's IAP verification API.
type GoogleClient interface {
	VerifyReceipt(ctx context.Context, receipt string) (string, error) // Returns (productID, error)
}

// StripeClient is for Stripe.
type StripeClient interface {
	CreateIntent(ctx context.Context, userID uuid.UUID, productID string) (string, error)
	HandleEvent(ctx context.Context, payload []byte) error
}

// --- BillingClient Implementation ---

type httpBillingClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewHTTPBillingClient(baseURL string) BillingClient {
	return &httpBillingClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    baseURL,
	}
}

type creditRequest struct {
	UserID string `json:"user_id"`
	Amount int    `json:"amount"`
}
type creditResponse struct {
	NewBalance int `json:"new_balance"`
}

func (c *httpBillingClient) CreditToken(ctx context.Context, userID uuid.UUID, amount int) (int, error) {
	reqBody, err := json.Marshal(creditRequest{
		UserID: userID.String(),
		Amount: amount,
	})
	if err != nil {
		return 0, fmt.Errorf("could not marshal credit request: %w", err)
	}

	url := c.baseURL + "/token/add"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, fmt.Errorf("could not create credit http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("credit request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("billing service returned non-200 status: %d", resp.StatusCode)
	}

	var creditResp creditResponse
	if err := json.NewDecoder(resp.Body).Decode(&creditResp); err != nil {
		return 0, fmt.Errorf("could not decode credit response: %w", err)
	}
	return creditResp.NewBalance, nil
}

// --- UserClient Implementation ---

type httpUserClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewHTTPUserClient(baseURL string) UserClient {
	return &httpUserClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    baseURL,
	}
}

// GetUserProfile fetches a user by their internal UUID.
func (c *httpUserClient) GetUserProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	url := fmt.Sprintf("%s/users/internal/%s", c.baseURL, userID.String())
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create get-user http request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get-user request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user service returned non-200 status: %d", resp.StatusCode)
	}

	var user domain.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("could not decode user profile: %w", err)
	}
	return &user, nil
}

// --- Stub Implementations for External APIs ---

type stubAppleClient struct{}

func NewStubAppleClient() AppleClient {
	return &stubAppleClient{}
}
func (s *stubAppleClient) VerifyReceipt(ctx context.Context, receipt string) (string, error) {
	fmt.Printf("STUB: Verifying Apple receipt: %s\n", receipt)
	return "pack_5_tokens", nil
}

type stubGoogleClient struct{}

func NewStubGoogleClient() GoogleClient {
	return &stubGoogleClient{}
}
func (s *stubGoogleClient) VerifyReceipt(ctx context.Context, receipt string) (string, error) {
	fmt.Printf("STUB: Verifying Google receipt: %s\n", receipt)
	return "pack_5_tokens", nil
}

type stubStripeClient struct{}

func NewStubStripeClient() StripeClient {
	return &stubStripeClient{}
}
func (s *stubStripeClient) CreateIntent(ctx context.Context, userID uuid.UUID, productID string) (string, error) {
	fmt.Printf("STUB: Creating Stripe intent for user %s, product %s\n", userID, productID)
	return "fake_client_secret_for_stripe", nil
}
func (s *stubStripeClient) HandleEvent(ctx context.Context, payload []byte) error {
	fmt.Printf("STUB: Handling Stripe webhook event\n")
	return nil
}
