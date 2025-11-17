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
	Role                   string    `json:"role" db:"role"`
	StripeCustomerID       string    `json:"-" db:"stripe_customer_id"`
}

type Expert struct {
	ExpertID       uuid.UUID `json:"expert_id" db:"expert_id"`
	FirebaseAuthID string    `json:"-" db:"firebase_auth_id"`
	DisplayName    string    `json:"display_name" db:"display_name"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	Role           string    `json:"role" db:"role"`
}

type Product struct {
	ProductID       string `json:"product_id" db:"product_id"`
	Name            string `json:"name" db:"name"`
	Description     string `json:"description" db:"description"`
	PriceCents      int    `json:"price_cents" db:"price_cents"`
	TokenCredit     int    `json:"token_credit" db:"token_credit"`
	IsSubscription  bool   `json:"is_subscription" db:"is_subscription"`
	StripePriceID   string `json:"-" db:"stripe_price_id"`
	AppleProductID  string `json:"apple_product_id" db:"apple_product_id"`
	GoogleProductID string `json:"google_product_id" db:"google_product_id"`
}

type Subscription struct {
	SubscriptionID       uuid.UUID `json:"subscription_id" db:"subscription_id"`
	UserID               uuid.UUID `json:"user_id" db:"user_id"`
	ProductID            string    `json:"product_id" db:"product_id"`
	Status               string    `json:"status" db:"status"`
	CurrentPeriodEnd     time.Time `json:"current_period_end" db:"current_period_end"`
	StripeSubscriptionID string    `json:"-" db:"stripe_subscription_id"`
}

type PaymentTransaction struct {
	TransactionID         uuid.UUID `json:"transaction_id" db:"transaction_id"`
	UserID                uuid.UUID `json:"user_id" db:"user_id"`
	ProductID             string    `json:"product_id" db:"product_id"`
	AmountCents           int       `json:"amount_cents" db:"amount_cents"`
	Provider              string    `json:"provider" db:"provider"`
	ProviderTransactionID string    `json:"provider_transaction_id" db:"provider_transaction_id"`
	Status                string    `json:"status" db:"status"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

type AssistanceRequest struct {
	RequestID             uuid.UUID     `json:"request_id" db:"request_id"`
	UserID                uuid.UUID     `json:"user_id" db:"user_id"`
	ExpertID              uuid.NullUUID `json:"expert_id,omitempty" db:"expert_id"` // Use sql.NullUUID
	Status                string        `json:"status" db:"status"`
	LLMSummary            string        `json:"llm_summary" db:"llm_summary"`
	TwilioConversationSID string        `json:"twilio_conversation_sid" db:"twilio_conversation_sid"`
	CreatedAt             time.Time     `json:"created_at" db:"created_at"`
	AcceptedAt            sql.NullTime  `json:"accepted_at,omitempty" db:"accepted_at"` // Use sql.NullTime
	ResolvedAt            sql.NullTime  `json:"resolved_at,omitempty" db:"resolved_at"` // Use sql.NullTime
}

// ExpertRating stores the 1-5 star rating
type ExpertRating struct {
	RatingID  uuid.UUID `json:"rating_id" db:"rating_id"`
	RequestID uuid.UUID `json:"request_id" db:"request_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	ExpertID  uuid.UUID `json:"expert_id" db:"expert_id"`
	Score     int       `json:"score" db:"score"`
}
