package request

import (
	"context"
	"database/sql"
	"log"
	"os"
	"project-sage/internal/domain" // The shared domain models
	"testing"
	"time"

	"github.com/google/uuid"
)

// These package level variables hold the test database connection and the prerequisite data (user, expert) for all tests to use.
var (
	testDB     *sql.DB
	testRepo   Repository
	testUser   *domain.User   // The user who creates the request
	testExpert *domain.Expert // The expert who accepts the request
)

// TestMain sets up the database connection and prerequisite data before any tests in this package run.
func TestMain(m *testing.M) {
	connStr := os.Getenv("TEST_DB_URL")
	if connStr == "" {
		log.Println("TEST_DB_URL not set. Skipping request integration tests.")
		os.Exit(0) // Skip tests if no DB is configured.
	}

	var err error
	testDB, err = sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Could not connect to test database: %v", err)
	}

	// Create the repository using the test DB.
	testRepo = NewPostgresRepository(testDB)

	// My assistance_requests table has foreign keys to 'users' and 'experts', so I must create them first.
	if err := setupPrerequisites(); err != nil {
		log.Fatalf("Could not set up prerequisite data: %v", err)
	}

	// Run all the tests.
	code := m.Run()

	// Clean up everything after the tests are done.
	cleanAllTables()
	testDB.Close()
	os.Exit(code)
}

// setupPrerequisites inserts the testUser and testExpert into the DB.
func setupPrerequisites() error {
	cleanAllTables() // Start from a clean state.

	// Define the test user.
	testUser = &domain.User{
		UserID:                 uuid.New(),
		FirebaseAuthID:         "fb-req-test-user",
		DisplayName:            "Request Test User",
		MembershipTier:         "free",
		AssistanceTokenBalance: 3,
		Role:                   "user",
	}
	// Define the test expert.
	testExpert = &domain.Expert{
		ExpertID:       uuid.New(),
		FirebaseAuthID: "fb-req-test-expert",
		DisplayName:    "Expert Joe",
		IsActive:       true,
		Role:           "expert",
	}

	// Insert the user.
	queryUser := `INSERT INTO users (user_id, firebase_auth_id, display_name, membership_tier, assistance_token_balance, role)
				 VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := testDB.Exec(queryUser,
		testUser.UserID,
		testUser.FirebaseAuthID,
		testUser.DisplayName,
		testUser.MembershipTier,
		testUser.AssistanceTokenBalance,
		testUser.Role,
	)
	if err != nil {
		return err
	}

	// Insert the expert.
	queryExpert := `INSERT INTO experts (expert_id, firebase_auth_id, display_name, is_active, role)
					 VALUES ($1, $2, $3, $4, $5)`
	_, err = testDB.Exec(queryExpert,
		testExpert.ExpertID,
		testExpert.FirebaseAuthID,
		testExpert.DisplayName,
		testExpert.IsActive,
		testExpert.Role,
	)
	return err
}

// cleanAllTables wipes all test data respecting foreign key constraints.
func cleanAllTables() {
	if testDB == nil {
		return
	}
	// Delete in order of dependency.
	testDB.Exec("DELETE FROM expert_ratings")
	testDB.Exec("DELETE FROM assistance_requests")
	testDB.Exec("DELETE FROM users WHERE firebase_auth_id LIKE 'fb-req-test-%'")
	testDB.Exec("DELETE FROM experts WHERE firebase_auth_id LIKE 'fb-req-test-%'")
}

// cleanRequestTables is a helper to clean just requests/ratings between tests.
func cleanRequestTables() {
	testDB.Exec("DELETE FROM expert_ratings")
	testDB.Exec("DELETE FROM assistance_requests")
}

// createTestRequest is a helper to insert a single pending request.
func createTestRequest(ctx context.Context, twilioSid string) (*domain.AssistanceRequest, error) {
	req := &domain.AssistanceRequest{
		UserID:                testUser.UserID, // Uses the global test user
		LLMSummary:            "Test summary",
		TwilioConversationSID: twilioSid,
	}
	err := testRepo.CreateRequest(ctx, req)
	return req, err
}

// TestCreateAndGetRequest verifies that a request can be created and then fetched by its id.,
func TestCreateAndGetRequest(t *testing.T) {
	cleanRequestTables()
	ctx := context.Background()

	// Create a request.
	req, err := createTestRequest(ctx, "twil-123")
	if err != nil {
		t.Fatalf("Failed to create test request: %v", err)
	}

	// Try to get it back.
	fetchedReq, err := testRepo.GetRequestByID(ctx, req.RequestID)

	// Check that it was found and all fields match.
	if err != nil {
		t.Fatalf("GetRequestByID() returned error: %v", err)
	}
	if fetchedReq.RequestID != req.RequestID {
		t.Errorf("Expected RequestID %v, got %v", req.RequestID, fetchedReq.RequestID)
	}
	if fetchedReq.UserID != testUser.UserID {
		t.Errorf("Expected UserID %v, got %v", testUser.UserID, fetchedReq.UserID)
	}
	if fetchedReq.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", fetchedReq.Status)
	}
	if fetchedReq.LLMSummary != "Test summary" {
		t.Errorf("Expected summary 'Test summary', got '%s'", fetchedReq.LLMSummary)
	}
}

// TestGetRequestByID_NotFound verifies the not found error case.
func TestGetRequestByID_NotFound(t *testing.T) {
	cleanRequestTables()
	ctx := context.Background()
	nonExistentUUID := uuid.New()

	_, err := testRepo.GetRequestByID(ctx, nonExistentUUID)

	// Expect a specific error.
	if err == nil {
		t.Fatal("Expected an error for non-existent request, but got nil")
	}
	if err.Error() != "request not found" {
		t.Errorf("Expected 'request not found', got '%v'", err)
	}
}

// TestRequestLifecycle tests the main state transitions: pending -> active -> resolved.
func TestRequestLifecycle(t *testing.T) {
	cleanRequestTables()
	ctx := context.Background()

	// Start with a pending request.
	req, err := createTestRequest(ctx, "twil-lifecycle-456")
	if err != nil {
		t.Fatalf("Failed to create test request: %v", err)
	}

	// Accept the request.
	err = testRepo.AcceptRequest(ctx, req.RequestID, testExpert.ExpertID)

	if err != nil {
		t.Fatalf("AcceptRequest() returned error: %v", err)
	}

	// Verify it's active and has the experts ID.
	activeReq, _ := testRepo.GetRequestByID(ctx, req.RequestID)
	if activeReq.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", activeReq.Status)
	}
	if !activeReq.ExpertID.Valid || activeReq.ExpertID.UUID != testExpert.ExpertID {
		t.Errorf("Expected ExpertID %v, got %v", testExpert.ExpertID, activeReq.ExpertID.UUID)
	}
	if !activeReq.AcceptedAt.Valid {
		t.Error("Expected AcceptedAt to be set, but it was null")
	}

	// Resolve the request.
	err = testRepo.ResolveRequest(ctx, req.RequestID)

	if err != nil {
		t.Fatalf("ResolveRequest() returned error: %v", err)
	}

	// Verify its resolved.
	resolvedReq, _ := testRepo.GetRequestByID(ctx, req.RequestID)
	if resolvedReq.Status != "resolved" {
		t.Errorf("Expected status 'resolved', got '%s'", resolvedReq.Status)
	}
	if !resolvedReq.ResolvedAt.Valid {
		t.Error("Expected ResolvedAt to be set, but it was null")
	}
}

// TestGetPendingRequests verifies the expert queue logic.
func TestGetPendingRequests(t *testing.T) {
	cleanRequestTables()
	ctx := context.Background()

	// Create 3 requests with slight delays to ensure created_at is different.
	req1, _ := createTestRequest(ctx, "twil-p-1")
	time.Sleep(10 * time.Millisecond)
	req2, _ := createTestRequest(ctx, "twil-p-2")
	time.Sleep(10 * time.Millisecond)
	req3, _ := createTestRequest(ctx, "twil-p-3")

	// Accept one of them, so it's no longer pending.
	_ = testRepo.AcceptRequest(ctx, req2.RequestID, testExpert.ExpertID)

	// Fetch the pending queue.
	pending, err := testRepo.GetPendingRequests(ctx)

	if err != nil {
		t.Fatalf("GetPendingRequests() returned error: %v", err)
	}

	// Should only get 2 requests back.
	if len(pending) != 2 {
		t.Fatalf("Expected 2 pending requests, got %d", len(pending))
	}

	// verify they are in the correct order (oldest first).
	if pending[0].RequestID != req1.RequestID {
		t.Errorf("Expected first request to be %v (oldest), got %v", req1.RequestID, pending[0].RequestID)
	}
	if pending[1].RequestID != req3.RequestID {
		t.Errorf("Expected second request to be %v (newest), got %v", req3.RequestID, pending[1].RequestID)
	}
}

// TestAcceptRequest_Concurrency verifies that a request can't be accepted more than once.
func TestAcceptRequest_Concurrency(t *testing.T) {
	cleanRequestTables()
	ctx := context.Background()
	req, _ := createTestRequest(ctx, "twil-concur-789")

	// Accept it the first time.
	err := testRepo.AcceptRequest(ctx, req.RequestID, testExpert.ExpertID)
	if err != nil {
		t.Fatalf("First accept failed: %v", err)
	}

	// Try to accept it again.
	err = testRepo.AcceptRequest(ctx, req.RequestID, testExpert.ExpertID)

	// This should fail with a specific error.
	if err == nil {
		t.Fatal("Expected an error for double-accept, but got nil")
	}
	if err.Error() != "request not found or was already accepted" {
		t.Errorf("Expected '...already accepted' error, got '%v'", err)
	}
}

// TestCreateRating verifies a rating can be inserted.
func TestCreateRating(t *testing.T) {
	cleanRequestTables()
	ctx := context.Background()
	// Create a full request lifecycle first.
	req, _ := createTestRequest(ctx, "twil-rating-101")
	_ = testRepo.AcceptRequest(ctx, req.RequestID, testExpert.ExpertID)
	_ = testRepo.ResolveRequest(ctx, req.RequestID)

	// Define the rating.
	rating := &domain.ExpertRating{
		RequestID: req.RequestID,
		UserID:    testUser.UserID,
		ExpertID:  testExpert.ExpertID,
		Score:     5,
	}

	// Create the rating.
	err := testRepo.CreateRating(ctx, rating)

	if err != nil {
		t.Fatalf("CreateRating() returned error: %v", err)
	}
	// The repo should have set the ID.
	if rating.RatingID == (uuid.UUID{}) {
		t.Error("RatingID was not set by CreateRating")
	}

	// Verify it's actually in the database.
	var score int
	err = testDB.QueryRow("SELECT score FROM expert_ratings WHERE rating_id = $1", rating.RatingID).Scan(&score)
	if err != nil {
		t.Fatalf("Failed to verify rating in DB: %v", err)
	}
	if score != 5 {
		t.Errorf("Expected score 5, got %d", score)
	}
}
