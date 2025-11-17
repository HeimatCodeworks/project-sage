package request

import (
	"context"
	"fmt"
	"project-sage/internal/domain" // The shared domain models
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock" // Mocking library
)

// This is the unit test for the service layer, the orchestrator.

// setupMocks is a helper function to initialize all the mocks for each test.
func setupMocks(t *testing.T) (context.Context, *MockRepository, *MockBillingClient, *MockLLMClient, *MockChatClient, *MockUserClient, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	// This returns all the mocks and the controller to manage them.
	return context.Background(),
		NewMockRepository(ctrl),
		NewMockBillingClient(ctrl),
		NewMockLLMClient(ctrl),
		NewMockChatClient(ctrl),
		NewMockUserClient(ctrl),
		ctrl
}

// TestService_CreateRequest_Success_NormalUser tests the "happy path" for a regular user.
func TestService_CreateRequest_Success_NormalUser(t *testing.T) {
	// set up all mocks.
	ctx, mockRepo, mockBilling, mockLLM, mockChat, mockUserClient, ctrl := setupMocks(t)
	defer ctrl.Finish()

	userID := uuid.New()
	twilioSID := "twilio-sid-123"
	expectedSummary := "User needs help."
	mockUser := &domain.User{UserID: userID, Role: "user"}

	// We define the exact sequence of calls we expect the service to make.
	gomock.InOrder(
		mockUserClient.EXPECT().GetUserProfile(ctx, userID).Return(mockUser, nil).Times(1),

		// Debit token must be called next for a normal "user".
		mockBilling.EXPECT().DebitToken(ctx, userID).Return(nil).Times(1),

		// Summarize must be called next.
		mockLLM.EXPECT().Summarize(ctx, twilioSID).Return(expectedSummary, nil).Times(1),

		// CreateRequest in my own repo is called third.
		mockRepo.EXPECT().CreateRequest(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, req *domain.AssistanceRequest) error {
				if req.UserID != userID {
					t.Errorf("UserID mismatch in CreateRequest")
				}
				if req.LLMSummary != expectedSummary {
					t.Errorf("Summary mismatch in CreateRequest")
				}
				return nil
			}).Times(1),

		//  RemoveBot is the last step.
		mockChat.EXPECT().RemoveBot(ctx, twilioSID).Return(nil).Times(1),
	)

	// Create the service and call the method.
	s := NewService(mockRepo, mockBilling, mockLLM, mockChat, mockUserClient)
	req, err := s.CreateRequest(ctx, userID, twilioSID)

	// check that everything went well
	if err != nil {
		t.Fatalf("CreateRequest() returned unexpected error: %v", err)
	}
	if req == nil {
		t.Fatal("CreateRequest() returned nil request")
	}
}

// TestService_CreateRequest_Success_SuperAdmin tests the path for a superadmin.
func TestService_CreateRequest_Success_SuperAdmin(t *testing.T) {
	ctx, mockRepo, mockBilling, mockLLM, mockChat, mockUserClient, ctrl := setupMocks(t)
	defer ctrl.Finish()

	userID := uuid.New()
	twilioSID := "twilio-sid-super"
	expectedSummary := "Admin needs help."
	mockSuperAdmin := &domain.User{UserID: userID, Role: "superadmin"}

	gomock.InOrder(
		mockUserClient.EXPECT().GetUserProfile(ctx, userID).Return(mockSuperAdmin, nil).Times(1),

		// Summarize is called next.
		mockLLM.EXPECT().Summarize(ctx, twilioSID).Return(expectedSummary, nil).Times(1),

		// CreateRequest is called.
		mockRepo.EXPECT().CreateRequest(ctx, gomock.Any()).Return(nil).Times(1),

		// RemoveBot is the last step.
		mockChat.EXPECT().RemoveBot(ctx, twilioSID).Return(nil).Times(1),
	)

	// Expect the billing client to *never* be called.
	mockBilling.EXPECT().DebitToken(gomock.Any(), gomock.Any()).Times(0)

	s := NewService(mockRepo, mockBilling, mockLLM, mockChat, mockUserClient)
	_, err := s.CreateRequest(ctx, userID, twilioSID)

	if err != nil {
		t.Fatalf("CreateRequest() returned unexpected error: %v", err)
	}
}

// TestService_CreateRequest_Fail_GetUserProfile tests when the very first step fails.
func TestService_CreateRequest_Fail_GetUserProfile(t *testing.T) {
	ctx, mockRepo, mockBilling, mockLLM, mockChat, mockUserClient, ctrl := setupMocks(t)
	defer ctrl.Finish()

	userID := uuid.New()
	twilioSID := "twilio-sid-456"
	expectedErr := fmt.Errorf("user service is down")

	mockUserClient.EXPECT().GetUserProfile(ctx, userID).Return(nil, expectedErr).Times(1)

	// Expect all other clients to never be called.
	mockBilling.EXPECT().DebitToken(gomock.Any(), gomock.Any()).Times(0)
	mockLLM.EXPECT().Summarize(gomock.Any(), gomock.Any()).Times(0)
	mockRepo.EXPECT().CreateRequest(gomock.Any(), gomock.Any()).Times(0)
	mockChat.EXPECT().RemoveBot(gomock.Any(), gomock.Any()).Times(0)

	s := NewService(mockRepo, mockBilling, mockLLM, mockChat, mockUserClient)
	_, err := s.CreateRequest(ctx, userID, twilioSID)

	if err == nil {
		t.Fatal("Expected an error but got nil")
	}
	if err.Error() != "could not fetch user profile: user service is down" {
		t.Fatalf("Expected 'user service is down' error, got: %v", err)
	}
}

// TestService_CreateRequest_InsufficientFunds tests the failure case where the debiting fails.
func TestService_CreateRequest_InsufficientFunds(t *testing.T) {
	ctx, mockRepo, mockBilling, mockLLM, mockChat, mockUserClient, ctrl := setupMocks(t)
	defer ctrl.Finish()

	userID := uuid.New()
	twilioSID := "twilio-sid-456"
	expectedErr := fmt.Errorf("insufficient funds")
	mockUser := &domain.User{UserID: userID, Role: "user"} // A normal user

	gomock.InOrder(
		mockUserClient.EXPECT().GetUserProfile(ctx, userID).Return(mockUser, nil).Times(1),
		// Debit token fails.
		mockBilling.EXPECT().DebitToken(ctx, userID).Return(expectedErr).Times(1),
	)

	// Expect the other clients to never be called.
	mockLLM.EXPECT().Summarize(gomock.Any(), gomock.Any()).Times(0)
	mockRepo.EXPECT().CreateRequest(gomock.Any(), gomock.Any()).Times(0)
	mockChat.EXPECT().RemoveBot(gomock.Any(), gomock.Any()).Times(0)

	s := NewService(mockRepo, mockBilling, mockLLM, mockChat, mockUserClient)
	_, err := s.CreateRequest(ctx, userID, twilioSID)

	if err == nil {
		t.Fatal("Expected an error but got nil")
	}
	if err.Error() != "token debit failed: insufficient funds" {
		t.Fatalf("Expected 'insufficient funds' error, got: %v", err)
	}
}

// TestService_CreateRequest_LLMFailure tests a failure in the middle of the orchestration.
func TestService_CreateRequest_LLMFailure(t *testing.T) {
	ctx, mockRepo, mockBilling, mockLLM, mockChat, mockUserClient, ctrl := setupMocks(t)
	defer ctrl.Finish()

	userID := uuid.New()
	twilioSID := "twilio-sid-789"
	expectedErr := fmt.Errorf("LLM API is down")
	mockUser := &domain.User{UserID: userID, Role: "user"} // A normal user

	// Expect the first steps to happen in order.
	gomock.InOrder(
		mockUserClient.EXPECT().GetUserProfile(ctx, userID).Return(mockUser, nil).Times(1),
		// Debit succeeds.
		mockBilling.EXPECT().DebitToken(ctx, userID).Return(nil).Times(1),
		// LLM fails.
		mockLLM.EXPECT().Summarize(ctx, twilioSID).Return("", expectedErr).Times(1),
	)

	// The flow should stop here. These should not be called.
	mockRepo.EXPECT().CreateRequest(gomock.Any(), gomock.Any()).Times(0)
	mockChat.EXPECT().RemoveBot(gomock.Any(), gomock.Any()).Times(0)

	s := NewService(mockRepo, mockBilling, mockLLM, mockChat, mockUserClient)
	_, err := s.CreateRequest(ctx, userID, twilioSID)

	if err == nil {
		t.Fatal("Expected an error but got nil")
	}
	if err.Error() != "could not summarize chat: LLM API is down" {
		t.Fatalf("Expected 'LLM API is down' error, got: %v", err)
	}
}

// TestService_AcceptRequest_Success tests the happy path for an expert accepting a request.
func TestService_AcceptRequest_Success(t *testing.T) {
	ctx, mockRepo, mockBilling, mockLLM, mockChat, mockUserClient, ctrl := setupMocks(t)
	defer ctrl.Finish()

	reqID := uuid.New()
	expertID := uuid.New()
	twilioSID := "twilio-sid-abc"
	mockRequest := &domain.AssistanceRequest{
		RequestID:             reqID,
		ExpertID:              uuid.NullUUID{UUID: expertID, Valid: true},
		TwilioConversationSID: twilioSID,
		Status:                "active",
	}

	gomock.InOrder(
		mockRepo.EXPECT().AcceptRequest(ctx, reqID, expertID).Return(nil).Times(1),
		mockRepo.EXPECT().GetRequestByID(ctx, reqID).Return(mockRequest, nil).Times(1),
		mockChat.EXPECT().AddExpert(ctx, twilioSID, expertID).Return(nil).Times(1),
	)

	s := NewService(mockRepo, mockBilling, mockLLM, mockChat, mockUserClient)
	req, err := s.AcceptRequest(ctx, reqID, expertID)

	if err != nil {
		t.Fatalf("AcceptRequest() returned unexpected error: %v", err)
	}
	if req.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", req.Status)
	}
}

// TestService_AcceptRequest_AlreadyAccepted tests the race condition.
func TestService_AcceptRequest_AlreadyAccepted(t *testing.T) {
	ctx, mockRepo, mockBilling, mockLLM, mockChat, mockUserClient, ctrl := setupMocks(t)
	defer ctrl.Finish()

	reqID := uuid.New()
	expertID := uuid.New()
	expectedErr := fmt.Errorf("request... already accepted")

	mockRepo.EXPECT().AcceptRequest(ctx, reqID, expertID).Return(expectedErr).Times(1)

	mockRepo.EXPECT().GetRequestByID(gomock.Any(), gomock.Any()).Times(0)
	mockChat.EXPECT().AddExpert(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	s := NewService(mockRepo, mockBilling, mockLLM, mockChat, mockUserClient)
	_, err := s.AcceptRequest(ctx, reqID, expertID)

	if err == nil {
		t.Fatal("Expected an error but got nil")
	}
	if err.Error() != "could not accept request: request... already accepted" {
		t.Fatalf("Wrong error returned: %v", err)
	}
}
