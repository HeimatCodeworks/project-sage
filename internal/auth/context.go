package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// This package provides helper functions for setting and getting authenticated user/expert ids from a request context.
// This is how middleware will pass auth info to handlers.

// contextKey is a private type to avoid key collisions in the context.
type contextKey string

// These are the keys wel'l use to store and retrieve IDs.
const (
	UserIDKey   = contextKey("user_id")
	ExpertIDKey = contextKey("expert_id")
)

// SetUserID returns a new request with the user's ID added to its context.
// The auth middleware will call this.
func SetUserID(r *http.Request, id uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), UserIDKey, id)
	return r.WithContext(ctx)
}

// GetUserID retrieves the user's ID from the context.
// HTTP handlers will call this to see who is making the request.
func GetUserID(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(UserIDKey).(uuid.UUID)
	if !ok {
		// This will probably happen if the middleware is broken or misconfigured.
		return uuid.Nil, fmt.Errorf("no user ID in context")
	}
	return id, nil
}

// SetExpertID returns a new request with the expert's ID added to its context.
func SetExpertID(r *http.Request, id uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), ExpertIDKey, id)
	return r.WithContext(ctx)
}

// GetExpertID retrieves the expert's id from the context.
func GetExpertID(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(ExpertIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("no expert ID in context")
	}
	return id, nil
}
