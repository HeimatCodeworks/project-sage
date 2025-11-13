package request

//go:generate mockgen -destination=./repository_mock_test.go -package=request -source=repository.go Repository

import (
	"context"
	"database/sql"
	"fmt"
	"project-sage/internal/domain" // shared domain models
	"time"

	"github.com/google/uuid"
)

// Repository defines the contract for all database operations related to assistance requests and ratings.
type Repository interface {
	// CreateRequest inserts a new pending request
	CreateRequest(ctx context.Context, req *domain.AssistanceRequest) error
	// GetPendingRequests fetches all requests withpending status for the expert queue
	GetPendingRequests(ctx context.Context) ([]*domain.AssistanceRequest, error)
	// AcceptRequest assigns an expert and marks the request active.
	AcceptRequest(ctx context.Context, requestID, expertID uuid.UUID) error
	// ResolveRequest marks a request as resolved.
	ResolveRequest(ctx context.Context, requestID uuid.UUID) error
	// GetRequestByID fetches a single request (to check status, etc.).
	GetRequestByID(ctx context.Context, requestID uuid.UUID) (*domain.AssistanceRequest, error)
	// CreateRating inserts a new expert rating.
	CreateRating(ctx context.Context, rating *domain.ExpertRating) error
}

// postgresRepository is the concrete implementation of the repo using a Postgres database.
type postgresRepository struct {
	db *sql.DB // The database connection pool.
}

// NewPostgresRepository is the constructor for the repository.
func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepository{
		db: db,
	}
}

// CreateRequest inserts a new assistance_requests record.
func (pr *postgresRepository) CreateRequest(ctx context.Context, req *domain.AssistanceRequest) error {
	// Set server-side fields before insert.
	req.RequestID = uuid.New()
	req.Status = "pending" // all new requests start as pending.
	req.CreatedAt = time.Now().UTC()

	query := `
		INSERT INTO assistance_requests
			(request_id, user_id, status, llm_summary, twilio_conversation_sid, created_at)
		VALUES
			($1, $2, $3, $4, $5, $6)
	`
	// Execute the insert query.
	_, err := pr.db.ExecContext(ctx, query,
		req.RequestID,
		req.UserID,
		req.Status,
		req.LLMSummary,
		req.TwilioConversationSID,
		req.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("could not insert request: %w", err)
	}
	return nil
}

// GetPendingRequests fetches all requests with status='pending', ordered by creation time for the queue.
func (pr *postgresRepository) GetPendingRequests(ctx context.Context) ([]*domain.AssistanceRequest, error) {
	query := `
		SELECT request_id, user_id, twilio_conversation_sid, created_at
		FROM assistance_requests
		WHERE status = 'pending'
		ORDER BY created_at ASC
	` // ORDER BY ASC ensures the oldest requests are first.

	rows, err := pr.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("could not query pending requests: %w", err)
	}
	defer rows.Close()

	// Iterate over the rows and scan them into a slice.
	var requests []*domain.AssistanceRequest
	for rows.Next() {
		var req domain.AssistanceRequest
		// Note - This only scans the fields needed for the queue view.
		if err := rows.Scan(&req.RequestID, &req.UserID, &req.TwilioConversationSID, &req.CreatedAt); err != nil {
			return nil, fmt.Errorf("could not scan pending request: %w", err)
		}
		requests = append(requests, &req)
	}
	return requests, nil
}

// AcceptRequest atomically updates a request's status from pendin to active
func (pr *postgresRepository) AcceptRequest(ctx context.Context, requestID, expertID uuid.UUID) error {
	// This query is atomic. The where clause ensures we only update a request that is still pending.
	query := `
		UPDATE assistance_requests
		SET status = 'active', expert_id = $1, accepted_at = $2
		WHERE request_id = $3 AND status = 'pending'
	`

	res, err := pr.db.ExecContext(ctx, query, expertID, time.Now().UTC(), requestID)
	if err != nil {
		return fmt.Errorf("database error accepting request: %w", err)
	}

	// Check if a row was actually affected.
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not check rows affected: %w", err)
	}
	// If 0 rows, it means the request was not pending or didn't exist
	if rowsAffected == 0 {
		return fmt.Errorf("request not found or was already accepted")
	}

	return nil
}

// ResolveRequest marks an active request as resolved.
func (pr *postgresRepository) ResolveRequest(ctx context.Context, requestID uuid.UUID) error {
	// This query is also atomic.
	query := `
		UPDATE assistance_requests
		SET status = 'resolved', resolved_at = $1
		WHERE request_id = $2 AND status = 'active'
	`
	res, err := pr.db.ExecContext(ctx, query, time.Now().UTC(), requestID)
	if err != nil {
		return fmt.Errorf("database error resolving request: %w", err)
	}

	// Check rows affected to ensure a state transition occurred.
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("request not found or was not active")
	}

	return nil
}

// CreateRating inserts a new expert_ratings record.
func (pr *postgresRepository) CreateRating(ctx context.Context, rating *domain.ExpertRating) error {
	rating.RatingID = uuid.New() // Set the primary key.
	query := `
		INSERT INTO expert_ratings
			(rating_id, request_id, user_id, expert_id, score)
		VALUES
			($1, $2, $3, $4, $5)
	`
	_, err := pr.db.ExecContext(ctx, query,
		rating.RatingID,
		rating.RequestID,
		rating.UserID,
		rating.ExpertID,
		rating.Score,
	)
	if err != nil {
		return fmt.Errorf("could not insert rating: %w", err)
	}
	return nil
}

// GetRequestByID fetches a single complete request by its primary key.
func (pr *postgresRepository) GetRequestByID(ctx context.Context, requestID uuid.UUID) (*domain.AssistanceRequest, error) {
	var req domain.AssistanceRequest
	query := `
		SELECT request_id, user_id, expert_id, status, llm_summary, twilio_conversation_sid, created_at, accepted_at, resolved_at
		FROM assistance_requests
		WHERE request_id = $1
	`

	// This Scan call must match the query order and handle all the nullable fields (expert_id, accepted_at, resolved_at) which are defined as sql.NullTime/uuid.NullUUID in the domain.
	err := pr.db.QueryRowContext(ctx, query, requestID).Scan(
		&req.RequestID,
		&req.UserID,
		&req.ExpertID,
		&req.Status,
		&req.LLMSummary,
		&req.TwilioConversationSID,
		&req.CreatedAt,
		&req.AcceptedAt,
		&req.ResolvedAt,
	)
	if err != nil {
		// Handle the case where no row was found
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("request not found")
		}
		return nil, fmt.Errorf("could not get request: %w", err)
	}
	return &req, nil
}
