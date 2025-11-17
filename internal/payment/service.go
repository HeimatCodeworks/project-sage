package payment

import (
	"context"
	"fmt"
	"project-sage/internal/domain"
	"time"

	"github.com/google/uuid"
)

// Service defines the business logic for payments.
type Service interface {
	GetAvailableProducts(ctx context.Context) ([]*domain.Product, error)
	VerifyAppleIAP(ctx context.Context, userID uuid.UUID, receipt string) (*domain.User, error)
	VerifyGoogleIAP(ctx context.Context, userID uuid.UUID, receipt string) (*domain.User, error)
	CreateStripeIntent(ctx context.Context, userID uuid.UUID, productID string) (string, error)
	HandleStripeEvent(ctx context.Context, payload []byte) error
}

// service is the concrete implementation.
type service struct {
	repo          Repository
	billingClient BillingClient
	userClient    UserClient
	appleClient   AppleClient
	googleClient  GoogleClient
	stripeClient  StripeClient
}

// NewService is the constructor. It injects all required dependencies.
func NewService(
	r Repository,
	bc BillingClient,
	uc UserClient,
	ac AppleClient,
	gc GoogleClient,
	sc StripeClient,
) Service {
	return &service{
		repo:          r,
		billingClient: bc,
		userClient:    uc,
		appleClient:   ac,
		googleClient:  gc,
		stripeClient:  sc,
	}
}

// --- Service Implementation ---

// GetAvailableProducts is a pass through to the repository
func (s *service) GetAvailableProducts(ctx context.Context) ([]*domain.Product, error) {
	return s.repo.GetProducts(ctx)
}

// VerifyAppleIAP orchestrates the Apple purchase verification.
func (s *service) VerifyAppleIAP(ctx context.Context, userID uuid.UUID, receipt string) (*domain.User, error) {
	// Call external Apple API to verify receipt
	productID, err := s.appleClient.VerifyReceipt(ctx, receipt)
	if err != nil {
		return nil, fmt.Errorf("apple receipt verification failed: %w", err)
	}

	// eceipt is valid, complete the purchase flow
	return s.completePurchase(ctx, userID, productID, "apple", receipt)
}

// VerifyGoogleIAP orchestrates the Google purchase verification.
func (s *service) VerifyGoogleIAP(ctx context.Context, userID uuid.UUID, receipt string) (*domain.User, error) {
	// Call external Google api to verify receipt
	productID, err := s.googleClient.VerifyReceipt(ctx, receipt)
	if err != nil {
		return nil, fmt.Errorf("google receipt verification failed: %w", err)
	}

	// Receipt is valid so we complete the purchase flow
	return s.completePurchase(ctx, userID, productID, "google", receipt)
}

// completePurchase is a private helper to handle the common logic after a receipt has been successfully verified by its provider.
func (s *service) completePurchase(ctx context.Context, userID uuid.UUID, productID, provider, txID string) (*domain.User, error) {
	// Get product details from our DB
	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("purchase failed: could not find product %s: %w", productID, err)
	}

	// Call BillingService to credit tokens
	_, err = s.billingClient.CreditToken(ctx, userID, product.TokenCredit)
	if err != nil {
		return nil, fmt.Errorf("purchase failed: could not credit tokens: %w", err)
	}

	// klog the transaction in our payment_transactions table
	tx := &domain.PaymentTransaction{
		TransactionID:         uuid.New(),
		UserID:                userID,
		ProductID:             product.ProductID,
		AmountCents:           product.PriceCents,
		Provider:              provider,
		ProviderTransactionID: txID,
		Status:                "succeeded",
		CreatedAt:             time.Now().UTC(),
	}
	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		// non-fatal error logged for reference
		fmt.Printf("WARNING: Failed to log transaction %s for user %s\n", tx.TransactionID, userID)
	}

	// Get the updated user profile to return to the app
	updatedUser, err := s.userClient.GetUserProfile(ctx, userID)
	if err != nil {
		// The purchase succeeded but we cant get the new profile
		return nil, fmt.Errorf("purchase succeeded, but failed to fetch updated profile: %w", err)
	}

	return updatedUser, nil
}

// CreateStripeIntent uses the Stripe client.
func (s *service) CreateStripeIntent(ctx context.Context, userID uuid.UUID, productID string) (string, error) {
	return s.stripeClient.CreateIntent(ctx, userID, productID)
}

// HandleStripeEvent is called by the webhook handler.
func (s *service) HandleStripeEvent(ctx context.Context, payload []byte) error {
	return s.stripeClient.HandleEvent(ctx, payload)
}
