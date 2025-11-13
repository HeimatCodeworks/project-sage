package user

import (
	"context"
	"database/sql"
	"log"
	"os"
	"project-sage/internal/domain" // Shared domain models
	"testing"

	"github.com/google/uuid"
)

// These are package level variables for sharing the test database connection and repository across all tests in this file.
var (
	testDB   *sql.DB
	testRepo Repository
)

// TestMain is the entry point for this test package.
// It sets up the database connection before any tests run and tears it down after they all complete.
func TestMain(m *testing.M) {
	connStr := os.Getenv("TEST_DB_URL")
	if connStr == "" {
		// Skip integration tests if the DB URL isnt provided.
		log.Println("TEST_DB_URL not set. Skipping integration tests.")
		os.Exit(0)
	}

	var err error
	testDB, err = sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Could not connect to test database: %v", err)
	}

	// Create the repository instance for tests to use.
	testRepo = NewPostgresRepository(testDB)

	// Run all the tests (TestCreateUser, TestGetUserByFirebaseID_NotFound)
	code := m.Run()

	// Clean up and exit
	testDB.Close()
	os.Exit(code)
}

// cleanUserTable is a helper function to delete all rows from the users table, ensuring a clean state between tests.
func cleanUserTable() {
	_, err := testDB.Exec("DELETE FROM users")
	if err != nil {
		log.Fatalf("Could not clean users table: %v", err)
	}
}

// TestCreateUser verifies that a user can be inserted and then retrieved.
func TestCreateUser(t *testing.T) {
	// Start with a clean table.
	cleanUserTable()
	// Define the user to be created.
	newUser := &domain.User{
		FirebaseAuthID:         "fb-test-123",
		DisplayName:            "Test User",
		ProfileImageURL:        "http://example.com/img.png",
		MembershipTier:         "premium",
		AssistanceTokenBalance: 5,
	}
	ctx := context.Background()

	// Call the method under test.
	err := testRepo.CreateUser(ctx, newUser)

	// Check for insertion errors.
	if err != nil {
		t.Fatalf("CreateUser() returned an unexpected error: %v", err)
	}

	// Verify the repository set the UserID on the input struct.
	if newUser.UserID == (uuid.UUID{}) {
		t.Errorf("CreateUser() did not assign a UserID")
	}

	// Verify by fetching the user back from the database.
	fetchedUser, err := testRepo.GetUserByFirebaseID(ctx, "fb-test-123")
	if err != nil {
		t.Fatalf("Failed to fetch user back for verification: %v", err)
	}

	// Check that the fetched data matches what was inserted.
	if fetchedUser.DisplayName != "Test User" {
		t.Errorf("Fetched user display name mismatch: expected 'Test User', got '%s'", fetchedUser.DisplayName)
	}
	if fetchedUser.AssistanceTokenBalance != 5 {
		t.Errorf("Fetched user token balance mismatch: expected 5, got %d", fetchedUser.AssistanceTokenBalance)
	}
}

// TestGetUserByFirebaseID_NotFound verifies that the correct "not found" error is returned for a non-existent user.
func TestGetUserByFirebaseID_NotFound(t *testing.T) {
	cleanUserTable()
	ctx := context.Background()

	// Call the method with an ID that doesn't exist.
	_, err := testRepo.GetUserByFirebaseID(ctx, "non-existent-id")

	// An error is expected.
	if err == nil {
		t.Fatal("Expected an error for a non-existent user, but got nil")
	}

	// Check for the specific error string from the repository.
	if err.Error() != "user not found" {
		t.Errorf("Expected 'user not found' error, got: %v", err)
	}
}
