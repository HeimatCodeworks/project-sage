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
	cleanUserTable() // Clean up at the end
	testDB.Close()
	os.Exit(code)
}

// cleanUserTable is a helper function to delete all rows from the users table, ensuring a clean state between tests.
func cleanUserTable() {
	if testDB == nil {
		return
	}
	// Use a prefix to avoid deleting users from other tests
	_, err := testDB.Exec("DELETE FROM users WHERE firebase_auth_id LIKE 'fb-test-%'")
	if err != nil {
		log.Fatalf("Could not clean users table: %v", err)
	}
}

// TestCreateAndGetUserByFirebaseID verifies that a user can be inserted and then retrieved by FirebaseID.
func TestCreateAndGetUserByFirebaseID(t *testing.T) {
	// Start with a clean table.
	cleanUserTable()

	// Define the user to be created.
	newUser := &domain.User{
		FirebaseAuthID:         "fb-test-123",
		DisplayName:            "Test User",
		ProfileImageURL:        "http://example.com/img.png",
		MembershipTier:         "premium",
		AssistanceTokenBalance: 5,
		Role:                   "user",
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
	if fetchedUser.Role != "user" {
		t.Errorf("Fetched user role mismatch: expected 'user', got '%s'", fetchedUser.Role)
	}
}

// TestGetUserByID_Success verifies fetching by the primary key UUID.
func TestGetUserByID_Success(t *testing.T) {
	cleanUserTable()
	ctx := context.Background()

	// Create a user
	newUser := &domain.User{
		FirebaseAuthID:         "fb-test-by-uuid",
		DisplayName:            "User UUID Test",
		Role:                   "superadmin", // Test with a different role
		AssistanceTokenBalance: 99,
	}
	err := testRepo.CreateUser(ctx, newUser)
	if err != nil {
		t.Fatalf("CreateUser() failed: %v", err)
	}

	// fetch that user by their *internal* UserID
	fetchedUser, err := testRepo.GetUserByID(ctx, newUser.UserID)
	if err != nil {
		t.Fatalf("GetUserByID() returned error: %v", err)
	}

	// Verify the data
	if fetchedUser == nil {
		t.Fatal("GetUserByID() returned a nil user")
	}
	if fetchedUser.FirebaseAuthID != "fb-test-by-uuid" {
		t.Errorf("FirebaseAuthID mismatch")
	}
	if fetchedUser.Role != "superadmin" {
		t.Errorf("Role mismatch: expected 'superadmin', got '%s'", fetchedUser.Role)
	}
	if fetchedUser.UserID != newUser.UserID {
		t.Errorf("UserID mismatch")
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

// TestGetUserByID_NotFound verifies the not found case for the UUID query.
func TestGetUserByID_NotFound(t *testing.T) {
	cleanUserTable()
	ctx := context.Background()

	nonExistentID := uuid.New() // A random, non-existent UUID

	_, err := testRepo.GetUserByID(ctx, nonExistentID)

	if err == nil {
		t.Fatal("Expected an error for a non-existent user, but got nil")
	}
	if err.Error() != "user not found" {
		t.Errorf("Expected 'user not found' error, got: %v", err)
	}
}
