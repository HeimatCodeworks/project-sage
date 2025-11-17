package payment

import (
	"encoding/json"
	"net/http"

	"project-sage/internal/domain"
	// "project-sage/internal/auth"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler is the HTTP API layer for the PaymentService.
type Handler struct {
	service Service
}

// NewHandler creates a new Handler, injecting the service.
func NewHandler(s Service) *Handler {
	return &Handler{
		service: s,
	}
}

// RegisterRoutes attaches all the service's http endpoints to the router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	// ---client-facing Endpoints ---

	// GET /payment/products:
	// Returns a list of available subscriptions and token packs.
	r.Get("/payment/products", h.handleGetProducts)

	// POST /payment/verify-iap:
	// Verifies a receipt from Apple or Google.
	r.Post("/payment/verify-iap", h.handleVerifyIAP)

	// POST /payment/create-intent:
	// Creates a payment intent with Stripe.
	r.Post("/payment/create-intent", h.handleCreateStripeIntent)

	// --- Internal/Webhook Endpoints ---

	// POST /payment/webhook-stripe:
	// Listens for successful payment events from Stripe.
	r.Post("/payment/webhook-stripe", h.handleStripeWebhook)
}

// --- DTOs (Data Transfer Objects) ---

type createIntentRequest struct {
	ProductID string `json:"product_id"`
}

type createIntentResponse struct {
	ClientSecret string `json:"client_secret"`
}

type verifyIAPRequest struct {
	Provider string `json:"provider"` // "apple" or "google"
	Receipt  string `json:"receipt_data"`
}

// --- Handler Functions ---

// handleGetProducts fetches the list of all purchasable items.
func (h *Handler) handleGetProducts(w http.ResponseWriter, r *http.Request) {
	// TODO: Add auth middleware
	// _, err := auth.GetUserID(r.Context())
	// if err != nil {
	// 	writeError(w, http.StatusUnauthorized, "Not authorized")
	// 	return
	// }

	products, err := h.service.GetAvailableProducts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not fetch products")
		return
	}

	writeJSON(w, http.StatusOK, products)
}

// handleVerifyIAP receives a receipt from the client app and sends it to the service to be verified and to credit tokens.
func (h *Handler) handleVerifyIAP(w http.ResponseWriter, r *http.Request) {
	// TODO: Add auth middleware
	// userID, err := auth.GetUserID(r.Context())
	// if err != nil {
	// 	writeError(w, http.StatusUnauthorized, "Not authorized")
	// 	return
	// }
	// Faking userID for now
	userID := uuid.New()

	var req verifyIAPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var updatedUser *domain.User
	var err error

	if req.Provider == "apple" {
		updatedUser, err = h.service.VerifyAppleIAP(r.Context(), userID, req.Receipt)
	} else if req.Provider == "google" {
		updatedUser, err = h.service.VerifyGoogleIAP(r.Context(), userID, req.Receipt)
	} else {
		writeError(w, http.StatusBadRequest, "Invalid provider, must be 'apple' or 'google'")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not verify purchase")
		return
	}

	// On success, return the user's updated profile (or just the new token balance)
	writeJSON(w, http.StatusOK, updatedUser)
}

// handleCreateStripeIntent creates a Stripe PaymentIntent for credit card payments.
func (h *Handler) handleCreateStripeIntent(w http.ResponseWriter, r *http.Request) {
	// TODO: Add auth middleware
	// userID, err := auth.GetUserID(r.Context())
	// if err != nil {
	// 	writeError(w, http.StatusUnauthorized, "Not authorized")
	// 	return
	// }
	// Faking userID for now
	userID := uuid.New()

	var req createIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	clientSecret, err := h.service.CreateStripeIntent(r.Context(), userID, req.ProductID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create payment intent")
		return
	}

	writeJSON(w, http.StatusOK, createIntentResponse{ClientSecret: clientSecret})
}

// handleStripeWebhook is the endpoint Stripe sends events to.
func (h *Handler) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {

	// err := h.service.HandleStripeEvent(r.Body)
	// if err != nil {
	// 	writeError(w, http.StatusBadRequest, "Failed to process webhook")
	// 	return
	// }

	writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
}

// --- Helper Functions ---

// writeJSON is a helper function for sending json responses.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// writeError is a helper for sending a standardized json error.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
