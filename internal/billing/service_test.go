package billing

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock" // mock library
)

// This is a unit test for the service layer.

// TestService_DebitToken_Success tests the "happy path".
func TestService_DebitToken_Success(t *testing.T) {
	// Set up the mock controller.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish() // This makes sure all the expectss were met.

	// Create a mock of the repository.
	mockRepo := NewMockRepository(ctrl)
	// Create the service to test by injecting the mock repo.
	s := NewService(mockRepo)

	ctx := context.Background()
	testUserID := uuid.New()

	// We define the mock's behavior here.
	// I expect DebitToken to be called once, with these args and to return a balance of 2 and no error.
	mockRepo.EXPECT().
		DebitToken(ctx, testUserID).
		Return(2, nil).
		Times(1)

	// Now this calls the function being tested.
	newBalance, err := s.DebitToken(ctx, testUserID)

	// Check the results.
	if err != nil {
		t.Fatalf("Service returned an unexpected error: %v", err)
	}
	if newBalance != 2 {
		t.Fatalf("Expected new balance of 2, got %d", newBalance)
	}
}

// TestService_DebitToken_InsufficientFunds tests the error path.
// We need to make sure the service correctly passes up the error from the repository.
func TestService_DebitToken_InsufficientFunds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepository(ctrl)
	s := NewService(mockRepo)

	ctx := context.Background()
	testUserID := uuid.New()

	// This is the specific error I expect the repo to send.
	repoError := fmt.Errorf("insufficient funds or user not found")

	// Set up the mock to return my specific error.
	mockRepo.EXPECT().
		DebitToken(ctx, testUserID).
		Return(0, repoError). // Return 0 and the error.
		Times(1)

	// Call the function.
	_, err := s.DebitToken(ctx, testUserID)
	if err == nil {
		t.Fatal("Service did not return an error, but one was expected")
	}

	// Make sure the error is the exact one from the repo.
	if err.Error() != "insufficient funds or user not found" {
		t.Fatalf("Service returned wrong error: got '%v', want '%v'", err, repoError)
	}
}

// TestService_CreditToken_Success tests the "happy path" for crediting.
func TestService_CreditToken_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	s := NewService(mockRepo)

	ctx := context.Background()
	testUserID := uuid.New()
	amountToAdd := 5
	expectedNewBalance := 8 // eg if they had 3, now they have 8

	// Expect CreditToken to be called once with 5.
	// Return the new balance of 8.
	mockRepo.EXPECT().
		CreditToken(ctx, testUserID, amountToAdd).
		Return(expectedNewBalance, nil).
		Times(1)

	newBalance, err := s.CreditToken(ctx, testUserID, amountToAdd)

	if err != nil {
		t.Fatalf("Service returned an unexpected error: %v", err)
	}
	if newBalance != expectedNewBalance {
		t.Fatalf("Expected new balance of %d, got %d", expectedNewBalance, newBalance)
	}
}

func TestService_CreditToken_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	s := NewService(mockRepo)

	ctx := context.Background()
	testUserID := uuid.New()
	amountToAdd := 5
	repoError := fmt.Errorf("user not found") // The repo returns this

	// Expect CreditToken to be called, and return our fake error.
	mockRepo.EXPECT().
		CreditToken(ctx, testUserID, amountToAdd).
		Return(0, repoError).
		Times(1)

	_, err := s.CreditToken(ctx, testUserID, amountToAdd)

	if err == nil {
		t.Fatal("Service did not return an error, but one was expected")
	}
	// Check that the service passed the error up.
	if err.Error() != "user not found" {
		t.Fatalf("Service returned wrong error: got '%v', want '%v'", err, repoError)
	}
}
