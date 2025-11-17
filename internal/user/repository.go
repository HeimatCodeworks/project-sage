package user

//go:generate mockgen -destination=./repository_mock_test.go -package=user -source=repository.go Repository

import (
	"context"
	"database/sql"
	"fmt"
	"project-sage/internal/domain" // Shared domain models

	"github.com/google/uuid"
)

// Repository is the interface for all user related database operations.
// It defines the contract for the data layer
type Repository interface {
	// CreateUser inserts a new user record.
	CreateUser(ctx context.Context, user *domain.User) error
	// GetUserByFirebaseID finds a user by their unique auth ID.
	GetUserByFirebaseID(ctx context.Context, firebaseID string) (*domain.User, error)
	// GetUserByID finds a user by their primary key (UUID).
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

// postgresRepository is the concrete implementation of the Repository that uses a Postgres database
type postgresRepository struct {
	db *sql.DB // The database connection pool.
}

// NewPostgresRepository is the constructor for the repository.
func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepository{
		db: db,
	}
}

// CreateUser inserts a new row into the users table
func (pr *postgresRepository) CreateUser(ctx context.Context, user *domain.User) error {
	// Generate a new uuid for the users primary key.
	user.UserID = uuid.New()

	query := `
		INSERT INTO users (user_id, firebase_auth_id, display_name, profile_image_url, 
		                 membership_tier, assistance_token_balance, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	// Execute the query.
	_, err := pr.db.ExecContext(ctx, query,
		user.UserID,
		user.FirebaseAuthID,
		user.DisplayName,
		user.ProfileImageURL,
		user.MembershipTier,
		user.AssistanceTokenBalance,
		user.Role,
	)

	if err != nil {
		return fmt.Errorf("could not insert user: %w", err)
	}

	return nil
}

// GetUserByFirebaseID retrieves a single user based on their Firebase ID.
func (pr *postgresRepository) GetUserByFirebaseID(ctx context.Context, firebaseID string) (*domain.User, error) {
	user := &domain.User{}

	query := `
		SELECT user_id, firebase_auth_id, display_name, profile_image_url, 
		       membership_tier, assistance_token_balance, role
		FROM users
		WHERE firebase_auth_id = $1
	`

	// Use QueryRowContext as I'm expecting only one user
	// Scan the results directly into the user struct
	err := pr.db.QueryRowContext(ctx, query, firebaseID).Scan(
		&user.UserID,
		&user.FirebaseAuthID,
		&user.DisplayName,
		&user.ProfileImageURL,
		&user.MembershipTier,
		&user.AssistanceTokenBalance,
		&user.Role,
	)

	if err != nil {
		// This is the standard error for "not found".
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		// Some other database error occurred.
		return nil, fmt.Errorf("could not get user: %w", err)
	}

	return user, nil
}

// [GetUserByID retrieves a single user based on their internal UUID.
func (pr *postgresRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user := &domain.User{}

	query := `
		SELECT user_id, firebase_auth_id, display_name, profile_image_url, 
		       membership_tier, assistance_token_balance, role
		FROM users
		WHERE user_id = $1
	`
	err := pr.db.QueryRowContext(ctx, query, userID).Scan(
		&user.UserID,
		&user.FirebaseAuthID,
		&user.DisplayName,
		&user.ProfileImageURL,
		&user.MembershipTier,
		&user.AssistanceTokenBalance,
		&user.Role,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("could not get user: %w", err)
	}

	return user, nil
}
