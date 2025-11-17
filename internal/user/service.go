package user

import (
	"context"
	"fmt"
	"project-sage/internal/domain" // Shared domain models

	"github.com/google/uuid"
)

// Service defines the interface for the user service's business logic.
type Service interface {
	// RegisterNewUser handles the logic for creating a new user.
	RegisterNewUser(ctx context.Context, firebaseID, displayName, profileURL string) (*domain.User, error)
	// GetUser retrieves a user by their Firebase id
	GetUserByFirebaseID(ctx context.Context, firebaseID string) (*domain.User, error) // Renamed for clarity
	// GetUserByID retrieves a user by their internal UUID.
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

// service is the concrete implementation of the Service interface.
type service struct {
	repo Repository // It depends on the repository
}

// NewService is the constructor for the service injecting the repository.
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// RegisterNewUser contains the business logic for creating a new user.
func (s *service) RegisterNewUser(ctx context.Context, firebaseID, displayName, profileURL string) (*domain.User, error) {

	// This is where business logic lives.
	// We set default values for new users here.
	newUser := &domain.User{
		FirebaseAuthID:         firebaseID,
		DisplayName:            displayName,
		ProfileImageURL:        profileURL,
		MembershipTier:         "free", // All new users start on free tier.
		AssistanceTokenBalance: 3,      // Give new users 3 starting tokens.
		Role:                   "user",
	}

	// Pass the completed user object to the repository to be saved.
	err := s.repo.CreateUser(ctx, newUser)
	if err != nil {
		// Wrap the error for better context.
		return nil, fmt.Errorf("service could not register user: %w", err)
	}

	// Return the user object that now includes the server generated UserID.
	return newUser, nil
}

// GetUserByFirebaseID is a simple pass through to the repository.
func (s *service) GetUserByFirebaseID(ctx context.Context, firebaseID string) (*domain.User, error) {
	// Any future caching logic goes here
	return s.repo.GetUserByFirebaseID(ctx, firebaseID)
}

// GetUserByID is the passthrough for the internal endpoint.
func (s *service) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	// This is also a simple passthrough.
	return s.repo.GetUserByID(ctx, userID)
}
