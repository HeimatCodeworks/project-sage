package user

import (
	"context"
	"fmt"
	"project-sage/internal/domain" // The shared domain models
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock" // Mocking library
)

// This is the unit test for the user service.
// It mocks the repository to test the service's business logic in isolation.

// TestService_RegisterNewUser verifies the logic for new user registration.
func TestService_RegisterNewUser(t *testing.T) {
	// Set up the mock controler
	ctrl := gomock.NewController(t)
	defer ctrl.Finish() // Asserts that all expectss were met.

	// Create a mock of the repository
	mockRepo := NewMockRepository(ctrl)

	// Create the service and inject the mock.
	s := NewService(mockRepo)

	ctx := context.Background()

	// This is the struct we expect the service to build.
	// I'm testing that the service correctly sets the defaults.
	expectedUser := &domain.User{
		FirebaseAuthID:         "fb-new-user-123",
		DisplayName:            "New User",
		ProfileImageURL:        "http://new.com/img.png",
		MembershipTier:         "free", // This default is important.
		AssistanceTokenBalance: 3,      // So is this one.
		Role:                   "user",
	}

	// Define the mock's behavior.
	// I expect CreateUser to be called once, with the expectedUser struct, and to return no error.
	mockRepo.EXPECT().
		CreateUser(ctx, expectedUser).
		Return(nil).
		Times(1)

	// Call the service method I'm testing.
	user, err := s.RegisterNewUser(ctx, "fb-new-user-123", "New User", "http://new.com/img.png")

	// Check the results.
	if err != nil {
		t.Fatalf("RegisterNewUser() returned an unexpected error: %v", err)
	}

	if user == nil {
		t.Fatal("RegisterNewUser() returned a nil user")
	}

	// Check that the returned user object has the defaults set.
	if user.MembershipTier != "free" {
		t.Errorf("Expected user tier 'free', got '%s'", user.MembershipTier)
	}
	if user.AssistanceTokenBalance != 3 {
		t.Errorf("Expected user balance 3, got %d", user.AssistanceTokenBalance)
	}
	if user.Role != "user" {
		t.Errorf("Expected user role 'user', got '%s'", user.Role)
	}
}

// TestService_GetUserByID tests the passthrough for GetUserByID.
func TestService_GetUserByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	s := NewService(mockRepo)

	ctx := context.Background()
	testID := uuid.New()
	expectedUser := &domain.User{
		UserID:      testID,
		DisplayName: "Test User",
		Role:        "user",
	}

	// Expect the service to call the repo's GetUserByID
	mockRepo.EXPECT().
		GetUserByID(ctx, testID).
		Return(expectedUser, nil).
		Times(1)

	user, err := s.GetUserByID(ctx, testID)
	if err != nil {
		t.Fatalf("GetUserByID() returned an unexpected error: %v", err)
	}
	if user.UserID != testID {
		t.Errorf("Expected user ID %v, got %v", testID, user.UserID)
	}
}

// TestService_GetUserByID_NotFound tests the not found case.
func TestService_GetUserByID_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	s := NewService(mockRepo)

	ctx := context.Background()
	testID := uuid.New()
	repoError := fmt.Errorf("user not found")

	// Expect the service to call the repo and return an error
	mockRepo.EXPECT().
		GetUserByID(ctx, testID).
		Return(nil, repoError).
		Times(1)

	_, err := s.GetUserByID(ctx, testID)
	if err == nil {
		t.Fatal("Expected an error but got nil")
	}
	if err.Error() != "user not found" {
		t.Fatalf("Expected 'user not found', got '%v'", err)
	}
}
