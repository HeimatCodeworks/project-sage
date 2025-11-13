package billing

import (
	"context"
	"database/sql"
	"log"
	"os"
	"project-sage/internal/domain" // Shared domain models
	"testing"

	"github.com/google/uuid"
)

// These are package-level variables so all tests can share the same
// database connection and test data.
var (
	testDB   *sql.DB
	testRepo Repository
	testUser *domain.User // The user we debit from.
)

// TestMain is a special function that Go runs before any other tests in this file
func TestMain(m *testing.M) {
	connStr := os.Getenv("TEST_DB_URL")
	if connStr == "" {
		// If the test DB isn't set, we just skip these tests.
		log.Println("TEST_DB_URL not set. Skipping billing integration tests.")
		os.Exit(0)
	}

	var err error
	testDB, err = sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Could not connect to test database: %v", err)
	}

	// Create the repository using the test database connection.
	testRepo = NewPostgresRepository(testDB)

	// Since this service only updates users, we have to create one first for the tests to run against.
	if err := setupTestUser(); err != nil {
		log.Fatalf("Could not set up test user: %v", err)
	}

	// m.Run() actually runs all the other tests (TestDebitToken_*, etc.)
	code := m.Run()

	// clean up
	cleanTables()
	testDB.Close()
	os.Exit(code)
}

// setupTestUser is a helper to create a fresh user with 3 tokens.
func setupTestUser() error {
	cleanTables() // Start with a clean slate.

	testUser = &domain.User{
		UserID:                 uuid.New(),
		FirebaseAuthID:         "fb-billing-test-user", // A unique id just for this test
		DisplayName:            "Billing Test User",
		MembershipTier:         "free",
		AssistanceTokenBalance: 3, // Start with 3 tokens
	}

	query := `
		INSERT INTO users (user_id, firebase_auth_id, display_name, membership_tier, assistance_token_balance)
		VALUES ($1, $2, $3, $4, $5)
	`
	// Execute the insert.
	_, err := testDB.Exec(query,
		testUser.UserID,
		testUser.FirebaseAuthID,
		testUser.DisplayName,
		testUser.MembershipTier,
		testUser.AssistanceTokenBalance,
	)
	return err
}

// cleanTables cleans up only the user this test created.
func cleanTables() {
	testDB.Exec("DELETE FROM users WHERE firebase_auth_id = 'fb-billing-test-user'")
}

// resetUserTokens is a helper to reset the user's token balance before a test.
func resetUserTokens(balance int) error {
	testUser.AssistanceTokenBalance = balance
	_, err := testDB.Exec("UPDATE users SET assistance_token_balance = $1 WHERE user_id = $2",
		balance, testUser.UserID)
	return err
}

// TestDebitToken_Success tests the "happy path" of debiting tokens one by one.
func TestDebitToken_Success(t *testing.T) {
	// Make sure we start with 3 tokens.
	if err := resetUserTokens(3); err != nil {
		t.Fatalf("Failed to reset user tokens: %v", err)
	}
	ctx := context.Background()

	// Using sub-tests to check the balance as it decrements.
	t.Run("Debit 1 (3 -> 2)", func(t *testing.T) {
		newBalance, err := testRepo.DebitToken(ctx, testUser.UserID)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if newBalance != 2 {
			t.Fatalf("Expected new balance of 2, got %d", newBalance)
		}
	})

	t.Run("Debit 2 (2 -> 1)", func(t *testing.T) {
		newBalance, err := testRepo.DebitToken(ctx, testUser.UserID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if newBalance != 1 {
			t.Fatalf("Expected new balance of 1, got %d", newBalance)
		}
	})

	// This is the last debit, taking the balance to 0.
	t.Run("Debit 3 (1 -> 0)", func(t *testing.T) {
		newBalance, err := testRepo.DebitToken(ctx, testUser.UserID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if newBalance != 0 {
			t.Fatalf("Expected new balance of 0, got %d", newBalance)
		}
	})
}

// TestDebitToken_InsufficientFunds tests that it fails when the balance is already 0.
func TestDebitToken_InsufficientFunds(t *testing.T) {
	// Set the balance to 0.
	if err := resetUserTokens(0); err != nil {
		t.Fatalf("Failed to reset user tokens: %v", err)
	}
	ctx := context.Background()

	// Try to debit... this should fail.
	_, err := testRepo.DebitToken(ctx, testUser.UserID)

	if err == nil {
		t.Fatal("Expected an error for insufficient funds, but got nil")
	}

	// Check for the specific error our repository is supposed to return.
	if err.Error() != "insufficient funds or user not found" {
		t.Fatalf("Expected 'insufficient funds or user not found', got '%v'", err)
	}
}

// TestDebitToken_UserNotFound tests what happens if I try to debit from a user that doesn't exist.
func TestDebitToken_UserNotFound(t *testing.T) {
	ctx := context.Background()
	nonExistentUUID := uuid.New() // Just a random UUID.

	// This should also fail.
	_, err := testRepo.DebitToken(ctx, nonExistentUUID)

	if err == nil {
		t.Fatal("Expected an error for non-existent user, but got nil")
	}

	// It should return the same error as insufficient funds.
	if err.Error() != "insufficient funds or user not found" {
		t.Fatalf("Expected 'insufficient funds or user not found', got '%v'", err)
	}
}
