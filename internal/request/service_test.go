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
func setupMocks(t *testing.T) (context.Context, *MockRepository, *MockBillingClient, *MockLLMClient, *MockChatClient, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	// This returns all the mocks and the controller to manage them.
	return context.Background(),
		NewMockRepository(ctrl),
		NewMockBillingClient(ctrl),
		NewMockLLMClient(ctrl),
		NewMockChatClient(ctrl),
		ctrl
}

// TestService_CreateRequest_Success tests the "happy path" for the entire request creation orchestration.
func TestService_CreateRequest_Success(t *testing.T) {
	// set up all mocks.
	ctx, mockRepo, mockBilling, mockLLM, mockChat, ctrl := setupMocks(t)
	defer ctrl.Finish()

	userID := uuid.New()
	twilioSID := "twilio-sid-123"
	expectedSummary := "User needs help."

	// We define theexact sequence of calls we expect the service to make.
	gomock.InOrder(
		// Debit token must be caled first.
		mockBilling.EXPECT().DebitToken(ctx, userID).Return(nil).Times(1),

		// Summarize must be called next.
		mockLLM.EXPECT().Summarize(ctx, twilioSID).Return(expectedSummary, nil).Times(1),

		// CreateRequest in my own repo is called third.
		// I use gomock.Any() because the struct is created inside the service.
		mockRepo.EXPECT().CreateRequest(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, req *domain.AssistanceRequest) error {
				// I can still inspect the struct inside the mock.
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
	s := NewService(mockRepo, mockBilling, mockLLM, mockChat)
	req, err := s.CreateRequest(ctx, userID, twilioSID)

	// check that everything went well
	if err != nil {
		t.Fatalf("CreateRequest() returned unexpected error: %v", err)
	}
	if req == nil {
		t.Fatal("CreateRequest() returned nil request")
	}
	if req.LLMSummary != expectedSummary {
		t.Errorf("Expected summary '%s', got '%s'", expectedSummary, req.LLMSummary)
	}
}

// TestService_CreateRequest_InsufficientFunds tests the failure case where the first step (debiting) fails.
func TestService_CreateRequest_InsufficientFunds(t *testing.T) {
	ctx, mockRepo, mockBilling, mockLLM, mockChat, ctrl := setupMocks(t)
	defer ctrl.Finish()

	userID := uuid.New()
	twilioSID := "twilio-sid-456"
	expectedErr := fmt.Errorf("insufficient funds")

	// I expect only the billing client to be called, and it will return an error.
	mockBilling.EXPECT().DebitToken(ctx, userID).Return(expectedErr).Times(1)

	// Expect the other clients to never be called.
	mockLLM.EXPECT().Summarize(gomock.Any(), gomock.Any()).Times(0)
	mockRepo.EXPECT().CreateRequest(gomock.Any(), gomock.Any()).Times(0)
	mockChat.EXPECT().RemoveBot(gomock.Any(), gomock.Any()).Times(0)

	s := NewService(mockRepo, mockBilling, mockLLM, mockChat)
	_, err := s.CreateRequest(ctx, userID, twilioSID)

	if err == nil {
		t.Fatal("Expected an error but got nil")
	}
	// Check that the service wrapped the error message correctly
	if err.Error() != "token debit failed: insufficient funds" {
		t.Fatalf("Expected 'insufficient funds' error, got: %v", err)
	}
}

// TestService_CreateRequest_LLMFailure tests a failure in the middle of the orchestration.
func TestService_CreateRequest_LLMFailure(t *testing.T) {
	ctx, mockRepo, mockBilling, mockLLM, mockChat, ctrl := setupMocks(t)
	defer ctrl.Finish()

	userID := uuid.New()
	twilioSID := "twilio-sid-789"
	expectedErr := fmt.Errorf("LLM API is down")

	// Expect the first two steps to happen in order.
	gomock.InOrder(
		//  Debit succeeds.
		mockBilling.EXPECT().DebitToken(ctx, userID).Return(nil).Times(1),
		//  LLM fails.
		mockLLM.EXPECT().Summarize(ctx, twilioSID).Return("", expectedErr).Times(1),
	)

	// The flow should stop here. These should not ve called.
	mockRepo.EXPECT().CreateRequest(gomock.Any(), gomock.Any()).Times(0)
	mockChat.EXPECT().RemoveBot(gomock.Any(), gomock.Any()).Times(0)

	s := NewService(mockRepo, mockBilling, mockLLM, mockChat)
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
	ctx, mockRepo, mockBilling, mockLLM, mockChat, ctrl := setupMocks(t)
	defer ctrl.Finish()

	reqID := uuid.New()
	expertID := uuid.New()
	twilioSID := "twilio-sid-abc"

	// This is the mock request I expect GetRequestByID to return.
	mockRequest := &domain.AssistanceRequest{
		RequestID:             reqID,
		ExpertID:              uuid.NullUUID{UUID: expertID, Valid: true},
		TwilioConversationSID: twilioSID,
		Status:                "active",
	}

	// Define the expected sequence of calls.
	gomock.InOrder(
		// Update the DB.
		mockRepo.EXPECT().AcceptRequest(ctx, reqID, expertID).Return(nil).Times(1),
		// Fetch the request to get its Twilio SID.
		mockRepo.EXPECT().GetRequestByID(ctx, reqID).Return(mockRequest, nil).Times(1),
		// Add the expert to the chat.
		mockChat.EXPECT().AddExpert(ctx, twilioSID, expertID).Return(nil).Times(1),
	)

	s := NewService(mockRepo, mockBilling, mockLLM, mockChat)
	req, err := s.AcceptRequest(ctx, reqID, expertID)

	if err != nil {
		t.Fatalf("AcceptRequest() returned unexpected error: %v", err)
	}
	if req.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", req.Status)
	}
}

// TestService_AcceptRequest_AlreadyAccepted tests the race condition where a request is already accepted.
func TestService_AcceptRequest_AlreadyAccepted(t *testing.T) {
	ctx, mockRepo, mockBilling, mockLLM, mockChat, ctrl := setupMocks(t)
	defer ctrl.Finish()

	reqID := uuid.New()
	expertID := uuid.New()
	// This is the specific error from the repository
	expectedErr := fmt.Errorf("request... already accepted")

	// I expet only AcceptRequest to be called.
	mockRepo.EXPECT().AcceptRequest(ctx, reqID, expertID).Return(expectedErr).Times(1)

	// The flow must stop here.
	mockRepo.EXPECT().GetRequestByID(gomock.Any(), gomock.Any()).Times(0)
	mockChat.EXPECT().AddExpert(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	s := NewService(mockRepo, mockBilling, mockLLM, mockChat)
	_, err := s.AcceptRequest(ctx, reqID, expertID)

	if err == nil {
		t.Fatal("Expected an error but got nil")
	}
	if err.Error() != "could not accept request: request... already accepted" {
		t.Fatalf("Wrong error returned: %v", err)
	}
}
