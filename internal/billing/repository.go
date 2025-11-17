package billing

//go:generate mockgen -destination=./repository_mock_test.go -package=billing -source=repository.go Repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// Repository is the interface for billing database operations.
// It just defines the contract for whatever database implementation we use.
type Repository interface {
	// DebitToken should atomically decrement a user's token balance.
	DebitToken(ctx context.Context, userID uuid.UUID) (int, error)
	CreditToken(ctx context.Context, userID uuid.UUID, amount int) (int, error)
}

// postgresRepository is the concrete implementation of the Repository that uses Postgres.
type postgresRepository struct {
	db *sql.DB // database connection pool.
}

// NewPostgresRepository is the constructor for the repository.
func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepository{
		db: db,
	}
}

// DebitToken implements the interface.
func (pr *postgresRepository) DebitToken(ctx context.Context, userID uuid.UUID) (int, error) {
	var newBalance int

	// This query is the core of this service.
	// Atomic update that only works if the balance is > 0.
	// This prevents race conditions and overdrafts.
	query := `
		UPDATE users
		SET assistance_token_balance = assistance_token_balance - 1
		WHERE user_id = $1 AND assistance_token_balance > 0
		RETURNING assistance_token_balance
	`

	// I use QueryRowContext().Scan() because the returning clause gives me back the one row and new balance.
	err := pr.db.QueryRowContext(ctx, query, userID).Scan(&newBalance)
	if err != nil {
		// If no rows were affected (either user not found or balance was 0), Scan() returns ErrNoRows.
		if err == sql.ErrNoRows {
			// This returns a specific error that the service layer can check for.
			return 0, fmt.Errorf("insufficient funds or user not found")
		}
		// something else went wrong (eg. connection dropped)
		return 0, fmt.Errorf("database error during debit: %w", err)
	}

	return newBalance, nil
}

func (pr *postgresRepository) CreditToken(ctx context.Context, userID uuid.UUID, amount int) (int, error) {
	var newBalance int

	query := `
		UPDATE users
		SET assistance_token_balance = assistance_token_balance + $1
		WHERE user_id = $2
		RETURNING assistance_token_balance
	`

	// Use QueryRowContext().Scan() because returning gives new balance
	err := pr.db.QueryRowContext(ctx, query, amount, userID).Scan(&newBalance)
	if err != nil {
		// If the user_id doesn't existreturn sql.ErrNoRows.
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("user not found")
		}
		return 0, fmt.Errorf("database error during credit: %w", err)
	}

	return newBalance, nil
}
