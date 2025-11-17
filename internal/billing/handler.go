package billing

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler is the API layer for the billing service.
// It holds a reference to the service, which has the business logic.
type Handler struct {
	service Service
}

// NewHandler is the constructor for the handler.
func NewHandler(s Service) *Handler {
	return &Handler{
		service: s,
	}
}

// RegisterRoutes sets up the API routes for this handler
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/token/debit", h.handleDebitToken)

	r.Post("/token/add", h.handleCreditToken)
}

// --- DTOs ---

type creditRequest struct {
	UserID string `json:"user_id"`
	Amount int    `json:"amount"`
}

type creditResponse struct {
	NewBalance int `json:"new_balance"`
}

type debitRequest struct {
	UserID string `json:"user_id"`
}

type debitResponse struct {
	NewBalance int `json:"new_balance"`
}

// --- Handlers ---

// handleDebitToken is the main handler function for our one endpoint.
func (h *Handler) handleDebitToken(w http.ResponseWriter, r *http.Request) {
	// Try to decode the json body into our struct.
	var req debitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate that the UserID is a real uuid.
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user_id format")
		return
	}

	// This calls the business logic.
	newBalance, err := h.service.DebitToken(r.Context(), userID)
	if err != nil {
		// This is the specific error from the service for "no tokens".
		if err.Error() == "insufficient funds or user not found" {
			// Using 409 Conflict to signal this specific business rule failure.
			writeError(w, http.StatusConflict, "Insufficient funds or user not found")
			return
		}
		// Something else went wrong, probably the database.
		writeError(w, http.StatusInternalServerError, "Could not process debit")
		return
	}

	// Success. Send back the new balance.
	writeJSON(w, http.StatusOK, debitResponse{NewBalance: newBalance})
}

// This is called by the PaymentService.
func (h *Handler) handleCreditToken(w http.ResponseWriter, r *http.Request) {
	// Try to decode the json body.
	var req creditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate the UserID.
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user_id format")
		return
	}

	// we should only be adding positive amounts.
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "Amount must be positive")
		return
	}

	// Call the business logic layer.
	newBalance, err := h.service.CreditToken(r.Context(), userID, req.Amount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not process credit")
		return
	}

	// Success. Send back the new balance.
	writeJSON(w, http.StatusOK, creditResponse{NewBalance: newBalance})
}

// --- Helper Functions ---

// writeJSON is a helper to send json responses.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// writeError is a helper for sending a standard json error message.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
