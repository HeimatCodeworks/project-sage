package domain

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type User struct {
	UserID                 uuid.UUID `json:"user_id" db:"user_id"`
	FirebaseAuthID         string    `json:"-" db:"firebase_auth_id"`
	DisplayName            string    `json:"display_name" db:"display_name"`
	ProfileImageURL        string    `json:"profile_image_url" db:"profile_image_url"`
	MembershipTier         string    `json:"membership_tier" db:"membership_tier"`
	AssistanceTokenBalance int       `json:"assistance_token_balance" db:"assistance_token_balance"`
}

type Expert struct {
	ExpertID       uuid.UUID `json:"expert_id" db:"expert_id"`
	FirebaseAuthID string    `json:"-" db:"firebase_auth_id"`
	DisplayName    string    `json:"display_name" db:"display_name"`
	IsActive       bool      `json:"is_active" db:"is_active"`
}

type AssistanceRequest struct {
	RequestID             uuid.UUID     `json:"request_id" db:"request_id"`
	UserID                uuid.UUID     `json:"user_id" db:"user_id"`
	ExpertID              uuid.NullUUID `json:"expert_id,omitempty" db:"expert_id"`
	Status                string        `json:"status" db:"status"`
	LLMSummary            string        `json:"llm_summary" db:"llm_summary"`
	TwilioConversationSID string        `json:"twilio_conversation_sid" db:"twilio_conversation_sid"`
	CreatedAt             time.Time     `json:"created_at" db:"created_at"`
	AcceptedAt            sql.NullTime  `json:"accepted_at,omitempty" db:"accepted_at"`
	ResolvedAt            sql.NullTime  `json:"resolved_at,omitempty" db:"resolved_at"`
}

type ExpertRating struct {
	RatingID  uuid.UUID `json:"rating_id" db:"rating_id"`
	RequestID uuid.UUID `json:"request_id" db:"request_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	ExpertID  uuid.UUID `json:"expert_id" db:"expert_id"`
	Score     int       `json:"score" db:"score"`
}
