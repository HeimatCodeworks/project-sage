package chat

//go:generate mockgen -destination=./service_mock_test.go -package=chat -source=service.go Service

import (
	"context"
	"fmt"
	"project-sage/internal/domain"

	"github.com/google/uuid"
)

// Service defines the business logic for the ChatGatewayService.
type Service interface {
	// Generates a Twilio token for a standard user.
	GenerateUserToken(ctx context.Context, user *domain.User) (string, error)

	// Generates a Twilio token for an expert user.
	GenerateExpertToken(ctx context.Context, expert *domain.Expert) (string, error)

	// Creates a new chat conversation and adds the user and bot.
	CreateConversation(ctx context.Context, user *domain.User) (string, error)

	// Adds an expert to a conversation (called on accept).
	AddExpert(ctx context.Context, twilioSID string, expertID uuid.UUID) error

	// Removes the bot from a conversation (called on handoff).
	RemoveBot(ctx context.Context, twilioSID string) error

	// Fetches the chat history (called by LLMGatewayService).
	GetChatHistory(ctx context.Context, twilioSID string) ([]*Message, error)
}

// service is the concrete implementation of the Service interface.
type service struct {
	twilio TwilioClient
}

// NewService is the constructor for the ChatGatewayService.
func NewService(twilio TwilioClient) Service {
	return &service{
		twilio: twilio,
	}
}

// GenerateUserToken creates a token for a user.
// The identity for Twilio will be the user's UUID.
func (s *service) GenerateUserToken(ctx context.Context, user *domain.User) (string, error) {
	identity := user.UserID.String()
	token, err := s.twilio.GenerateToken(ctx, identity)
	if err != nil {
		return "", fmt.Errorf("could not generate user token: %w", err)
	}
	return token, nil
}

// GenerateExpertToken creates a token for an expert.
// The identity for Twilio will be the expert's UUID.
func (s *service) GenerateExpertToken(ctx context.Context, expert *domain.Expert) (string, error) {
	identity := expert.ExpertID.String()
	token, err := s.twilio.GenerateToken(ctx, identity)
	if err != nil {
		return "", fmt.Errorf("could not generate expert token: %w", err)
	}
	return token, nil
}

// CreateConversation creates a new Twilio Conversation.
func (s *service) CreateConversation(ctx context.Context, user *domain.User) (string, error) {
	// Create the conversation
	friendlyName := fmt.Sprintf("User Session: %s", user.UserID)
	convoSID, err := s.twilio.CreateConversation(ctx, friendlyName)
	if err != nil {
		return "", fmt.Errorf("could not create conversation: %w", err)
	}

	// Add user as the first participant
	if err := s.twilio.AddParticipant(ctx, convoSID, user.UserID.String()); err != nil {
		return "", fmt.Errorf("failed to add user to new conversation: %w", err)
	}

	// Add the llm as the second participant
	if err := s.twilio.AddParticipant(ctx, convoSID, "LLM_BOT_IDENTITY"); err != nil {
		// Log this as a non fatal error for now, as the chat can proceed.
		fmt.Printf("WARNING: Failed to add bot to new conversation %s: %v\n", convoSID, err)
	}

	return convoSID, nil
}

// AddExpert adds an expert to an existing conversation.
func (s *service) AddExpert(ctx context.Context, twilioSID string, expertID uuid.UUID) error {
	return s.twilio.AddParticipant(ctx, twilioSID, expertID.String())
}

// RemoveBot removes the bot from the conversation.
func (s *service) RemoveBot(ctx context.Context, twilioSID string) error {
	// "LLM_BOT_IDENTITY" is the static identity we use for the bot.
	// In a real implementation, this will come from config.
	return s.twilio.RemoveParticipant(ctx, twilioSID, "LLM_BOT_IDENTITY")
}

// GetChatHistory fetches messages from Twilio.
func (s *service) GetChatHistory(ctx context.Context, twilioSID string) ([]*Message, error) {
	return s.twilio.GetConversationHistory(ctx, twilioSID)
}
