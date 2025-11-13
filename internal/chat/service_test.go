package chat

import (
	"context"
	"project-sage/internal/domain"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func setupMocks(t *testing.T) (context.Context, *MockTwilioClient, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	return context.Background(), NewMockTwilioClient(ctrl), ctrl
}

func TestService_GenerateUserToken_Success(t *testing.T) {
	ctx, mockTwilio, ctrl := setupMocks(t)
	defer ctrl.Finish()

	user := &domain.User{UserID: uuid.New()}
	identity := user.UserID.String()
	expectedToken := "ey...token"

	// Expect GenerateToken to be called with the user's uuid string
	mockTwilio.EXPECT().
		GenerateToken(ctx, identity).
		Return(expectedToken, nil).
		Times(1)

	s := NewService(mockTwilio)
	token, err := s.GenerateUserToken(ctx, user)

	if err != nil {
		t.Fatalf("GenerateUserToken() returned unexpected error: %v", err)
	}
	if token != expectedToken {
		t.Errorf("want token '%s', got '%s'", expectedToken, token)
	}
}

func TestService_CreateConversation_Success(t *testing.T) {
	ctx, mockTwilio, ctrl := setupMocks(t)
	defer ctrl.Finish()

	user := &domain.User{UserID: uuid.New(), DisplayName: "Test User"}
	userUUID := user.UserID.String()
	convoSID := "CH-123"

	// Define the expected sequence of calls
	gomock.InOrder(
		// Create the conversation
		mockTwilio.EXPECT().
			CreateConversation(ctx, gomock.Any()). // We don't need to check the friendly name
			Return(convoSID, nil).
			Times(1),

		// Add the user
		mockTwilio.EXPECT().
			AddParticipant(ctx, convoSID, userUUID).
			Return(nil).
			Times(1),

		// Add the bot
		mockTwilio.EXPECT().
			AddParticipant(ctx, convoSID, "LLM_BOT_IDENTITY").
			Return(nil).
			Times(1),
	)

	s := NewService(mockTwilio)
	sid, err := s.CreateConversation(ctx, user)

	if err != nil {
		t.Fatalf("CreateConversation() returned unexpected error: %v", err)
	}
	if sid != convoSID {
		t.Errorf("want SID '%s', got '%s'", convoSID, sid)
	}
}

func TestService_RemoveBot_Success(t *testing.T) {
	ctx, mockTwilio, ctrl := setupMocks(t)
	defer ctrl.Finish()

	convoSID := "CH-123"
	botIdentity := "LLM_BOT_IDENTITY"

	// Expect RemoveParticipant to be called with the bot's identity
	mockTwilio.EXPECT().
		RemoveParticipant(ctx, convoSID, botIdentity).
		Return(nil).
		Times(1)

	s := NewService(mockTwilio)
	err := s.RemoveBot(ctx, convoSID)

	if err != nil {
		t.Fatalf("RemoveBot() returned unexpected error: %v", err)
	}
}

func TestService_GetChatHistory_Success(t *testing.T) {
	ctx, mockTwilio, ctrl := setupMocks(t)
	defer ctrl.Finish()

	convoSID := "CH-123"
	expectedHistory := []*Message{{Author: "user", Content: "Hello"}}

	// Expect GetConversationHistory to be called
	mockTwilio.EXPECT().
		GetConversationHistory(ctx, convoSID).
		Return(expectedHistory, nil).
		Times(1)

	s := NewService(mockTwilio)
	history, err := s.GetChatHistory(ctx, convoSID)

	if err != nil {
		t.Fatalf("GetChatHistory() returned unexpected error: %v", err)
	}
	if len(history) != 1 || history[0].Content != "Hello" {
		t.Errorf("Unexpected history returned")
	}
}
