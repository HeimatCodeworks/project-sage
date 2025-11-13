package request

import (
	"context"
	"fmt"
	"project-sage/internal/domain" // The shared domain models

	"github.com/google/uuid"
)

// Service defines the business logic operations for the request orchestrator.
type Service interface {
	// User-facing operations
	CreateRequest(ctx context.Context, userID uuid.UUID, twilioSID string) (*domain.AssistanceRequest, error)
	SubmitRating(ctx context.Context, reqID, userID, expertID uuid.UUID, score int) error

	// Expert-facing operations
	GetPendingRequests(ctx context.Context) ([]*domain.AssistanceRequest, error)
	AcceptRequest(ctx context.Context, requestID, expertID uuid.UUID) (*domain.AssistanceRequest, error)
	ResolveRequest(ctx context.Context, requestID, expertID uuid.UUID) error
}

// service implements the Service interface and orchestrates all other clients and repositories
type service struct {
	repo          Repository    // Our own database access
	billingClient BillingClient // Client for the BillingService
	llmClient     LLMClient     // Client for the LLMGatewayService
	chatClient    ChatClient    // Client for the ChatGatewayService
}

// NewService is the constructor, injecting all required dependencies.
func NewService(r Repository, bc BillingClient, lc LLMClient, cc ChatClient) Service {
	return &service{
		repo:          r,
		billingClient: bc,
		llmClient:     lc,
		chatClient:    cc,
	}
}

// CreateRequest orchestrates the new request handoff: debiting a token, summarizing the chat, and creating the request record.
func (s *service) CreateRequest(ctx context.Context, userID uuid.UUID, twilioSID string) (*domain.AssistanceRequest, error) {

	// Attempt to debit a token from the billing service first.
	if err := s.billingClient.DebitToken(ctx, userID); err != nil {
		// If debit fails (eg insufficient funds), stop the process.
		return nil, fmt.Errorf("token debit failed: %w", err)
	}

	// Get the LLM summary of the chat.
	summary, err := s.llmClient.Summarize(ctx, twilioSID)
	if err != nil {
		// If summary fails, the token was already debited. Log this as a warning.
		fmt.Printf("WARNING: Token debited for user %s, but LLM summary failed: %v\n", userID, err)
		return nil, fmt.Errorf("could not summarize chat: %w", err)
	}

	// Create the new request object to be saved.
	req := &domain.AssistanceRequest{
		UserID:                userID,
		LLMSummary:            summary,
		TwilioConversationSID: twilioSID,
	}
	// Persist the new pending request to our database.
	if err := s.repo.CreateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("could not save request: %w", err)
	}

	// Remove the bot from the chat. Log a warning if this fails, but don't fail the request.
	if err := s.chatClient.RemoveBot(ctx, twilioSID); err != nil {
		fmt.Printf("WARNING: Failed to remove bot from %s: %v\n", twilioSID, err)
	}

	return req, nil
}

// AcceptRequest orchestrates an expert accepting a pending request.
func (s *service) AcceptRequest(ctx context.Context, requestID, expertID uuid.UUID) (*domain.AssistanceRequest, error) {
	// Atomically update the DB. This handles the already accepted race condition.
	if err := s.repo.AcceptRequest(ctx, requestID, expertID); err != nil {
		return nil, fmt.Errorf("could not accept request: %w", err)
	}

	// Need to fetch the request to get its Twilio SID.
	req, err := s.repo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch accepted request: %w", err)
	}

	// Add the expert to the Twilio chat.
	if err := s.chatClient.AddExpert(ctx, req.TwilioConversationSID, expertID); err != nil {
		// Critical failure - the DB says they accepted, but they can't join the chat.
		fmt.Printf("CRITICAL: Failed to add expert %s to chat %s: %v\n", expertID, req.TwilioConversationSID, err)
		return nil, fmt.Errorf("failed to add expert to chat: %w", err)
	}

	return req, nil
}

// GetPendingRequests is a simple pass through to the repository.
func (s *service) GetPendingRequests(ctx context.Context) ([]*domain.AssistanceRequest, error) {
	return s.repo.GetPendingRequests(ctx)
}

// ResolveRequest is a pass through to the repository.
func (s *service) ResolveRequest(ctx context.Context, requestID, expertID uuid.UUID) error {
	// TODO: Verify the expertID here matches the one on the request.
	return s.repo.ResolveRequest(ctx, requestID)
}

// SubmitRating builds the rating object and passes it to the repository
func (s *service) SubmitRating(ctx context.Context, reqID, userID, expertID uuid.UUID, score int) error {
	rating := &domain.ExpertRating{
		RequestID: reqID,
		UserID:    userID,
		ExpertID:  expertID,
		Score:     score,
	}
	return s.repo.CreateRating(ctx, rating)
}
