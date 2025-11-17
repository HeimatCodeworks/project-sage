package billing

import (
	"context"

	"github.com/google/uuid"
)

// Service is the interface for the billing service's business logic.
// It defines the contract for what the service can do.
type Service interface {
	DebitToken(ctx context.Context, userID uuid.UUID) (int, error)
	CreditToken(ctx context.Context, userID uuid.UUID, amount int) (int, error)
}

// service is the concrete implementation of the Service interface.
type service struct {
	repo Repository
}

// NewService is the constructor for the service.
// It takes the repository as a dependency.
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// DebitToken attempts to debit one token from a user.
func (s *service) DebitToken(ctx context.Context, userID uuid.UUID) (int, error) {

	// For now, service logic is just a simple pass-through to the repository. The repo's SQL query has all the logic.
	newBalance, err := s.repo.DebitToken(ctx, userID)
	if err != nil {
		// Just pass the error up (eg "insufficient funds").
		return 0, err
	}

	return newBalance, nil
}

// This is also a simple passthrough to the repository's atomic SQL.
func (s *service) CreditToken(ctx context.Context, userID uuid.UUID, amount int) (int, error) {
	newBalance, err := s.repo.CreditToken(ctx, userID, amount)
	if err != nil {
		// Pass up errors like "user not found"
		return 0, err
	}
	return newBalance, nil
}
