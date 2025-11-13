package user

import (
	"context"
	"project-sage/internal/domain" // The shared domain models
	"testing"

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
	// I'm testing that the service correctly sets the default MembershipTier and AssistanceTokenBalance
	expectedUser := &domain.User{
		FirebaseAuthID:         "fb-new-user-123",
		DisplayName:            "New User",
		ProfileImageURL:        "http://new.com/img.png",
		MembershipTier:         "free", // This default is important.
		AssistanceTokenBalance: 3,      // So is this one.
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
}
