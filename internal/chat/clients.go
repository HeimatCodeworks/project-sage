package chat

//go:generate mockgen -destination=./clients_mock_test.go -package=chat -source=clients.go

import (
	"context"
	"fmt"
	"time"
)

// TwilioClient defines the contract for an external client that interacts with the Twilio conversations API.
type TwilioClient interface {
	// GenerateToken creates an access token for a user/expert identity.
	GenerateToken(ctx context.Context, identity string) (string, error)

	// CreateConversation creates a new chat session.
	CreateConversation(ctx context.Context, friendlyName string) (string, error)

	// AddParticipant adds a user/expert to a conersation.
	AddParticipant(ctx context.Context, conversationSID, identity string) error

	// RemoveParticipant removes a participant (eg. the llm).
	RemoveParticipant(ctx context.Context, conversationSID, participantSID string) error

	// GetConversationHistory fetches all messages from a conversation.
	GetConversationHistory(ctx context.Context, conversationSID string) ([]*Message, error)
}

type stubTwilioClient struct{}

// NewStubTwilioClient is the constructor for the fake client.
func NewStubTwilioClient() TwilioClient {
	return &stubTwilioClient{}
}

func (s *stubTwilioClient) GenerateToken(ctx context.Context, identity string) (string, error) {
	// Return a fake, static token.
	return fmt.Sprintf("fake-twilio-token-for-%s", identity), nil
}

func (s *stubTwilioClient) CreateConversation(ctx context.Context, friendlyName string) (string, error) {
	// Return a fake static conversation SID.
	return "CH_FAKE_SID_123456789", nil
}

func (s *stubTwilioClient) AddParticipant(ctx context.Context, conversationSID, identity string) error {
	// Log what we're doing and return nil.
	fmt.Printf("STUB: Added participant %s to %s\n", identity, conversationSID)
	return nil
}

func (s *stubTwilioClient) RemoveParticipant(ctx context.Context, conversationSID, participantSID string) error {
	// Log what we're doing and return nil.
	fmt.Printf("STUB: Removed participant %s from %s\n", participantSID, conversationSID)
	return nil
}

func (s *stubTwilioClient) GetConversationHistory(ctx context.Context, conversationSID string) ([]*Message, error) {
	// Return a static hardcoded history.
	return []*Message{
		{
			SID:       "MSG_FAKE_1",
			Author:    "user-uuid",
			Content:   "Hello, my Wi-Fi isn't working.",
			Timestamp: time.Now().Add(-5 * time.Minute),
		},
		{
			SID:       "MSG_FAKE_2",
			Author:    "LLM_BOT_IDENTITY",
			Content:   "I see. Have you tried turning it off and on again?",
			Timestamp: time.Now().Add(-4 * time.Minute),
		},
	}, nil
}
