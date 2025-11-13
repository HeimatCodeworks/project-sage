package request

import (
	"encoding/json"
	"net/http"

	// "project-sage/internal/auth" // I'll need this when I add real auth.

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler is the HTTP API layer for the RequestService.
// It holds a dependency on the business logic service.
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
// This includes both user facing and expert-facing routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	// User facing routes
	r.Post("/request/create", h.handleCreateRequest)
	r.Post("/request/rate", h.handleRateRequest)

	// Expert facing routes
	r.Get("/request/pending", h.handleGetPendingRequests)
	r.Post("/request/accept", h.handleAcceptRequest)
	r.Post("/request/resolve", h.handleResolveRequest)
}

// CreateRequestPayload is the DTO for the POST /request/create endpoint.
type CreateRequestPayload struct {
	TwilioConversationSID string `json:"twilio_conversation_sid"`
}

// RateRequestPayload is the DTO for the POST /request/rate endpoint.
type RateRequestPayload struct {
	RequestID string `json:"request_id"`
	ExpertID  string `json:"expert_id"`
	Score     int    `json:"score"`
}

// AcceptRequestPayload is the DTO for the POST /request/accept endpoint.
type AcceptRequestPayload struct {
	RequestID string `json:"request_id"`
}

// ResolveRequestPayload is the DTO for the POST /request/resolve endpoint.
type ResolveRequestPayload struct {
	RequestID string `json:"request_id"`
}

// handleCreateRequest is the handler for the user-facing request creation endpoint.
func (h *Handler) handleCreateRequest(w http.ResponseWriter, r *http.Request) {
	// Need to replace this placeholder with real auth middleware
	userID := uuid.New() // Placeholder
	// userID, err := auth.GetUserID(r.Context())
	// if err != nil {
	// 	writeError(w, http.StatusUnauthorized, "Not authorized")
	// 	return
	// }

	// Decode the incoming json payload.
	var payload CreateRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Call the core business logic in the service.
	req, err := h.service.CreateRequest(r.Context(), userID, payload.TwilioConversationSID)
	if err != nil {
		// This is a specific business error.
		if err.Error() == "token debit failed: insufficient funds" {
			// Return 402 Payment Required.
			writeError(w, http.StatusPaymentRequired, "Insufficient assistance tokens")
			return
		}
		// Something else went wrong.
		writeError(w, http.StatusInternalServerError, "Could not create request")
		return
	}

	// Respond with the new request object.
	writeJSON(w, http.StatusCreated, req)
}

// handleRateRequest allows a user to submit a rating for a completed request.
func (h *Handler) handleRateRequest(w http.ResponseWriter, r *http.Request) {
	userID := uuid.New() // Placeholder
	// userID, err := auth.GetUserID(r.Context()) ...

	var payload RateRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	reqID, _ := uuid.Parse(payload.RequestID)
	expertID, _ := uuid.Parse(payload.ExpertID)
	// TODO: I need to add proper error handling for bad UUIDs here.

	err := h.service.SubmitRating(r.Context(), reqID, userID, expertID, payload.Score)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not submit rating")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rating received"})
}

// handleGetPendingRequests is the expert facing endpoint to fetch the queue.
func (h *Handler) handleGetPendingRequests(w http.ResponseWriter, r *http.Request) {
	// _ , err := auth.GetExpertID(r.Context()) ...

	requests, err := h.service.GetPendingRequests(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not fetch pending requests")
		return
	}

	writeJSON(w, http.StatusOK, requests)
}

// handleAcceptRequest allows an expert to accept a pending request.
func (h *Handler) handleAcceptRequest(w http.ResponseWriter, r *http.Request) {
	expertID := uuid.New() // Placeholder
	// expertID, err := auth.GetExpertID(r.Context()) ...

	var payload AcceptRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	reqID, _ := uuid.Parse(payload.RequestID) // TODO: Handle parse error.

	req, err := h.service.AcceptRequest(r.Context(), reqID, expertID)
	if err != nil {
		// Handle the specific concurrency error.
		if err.Error() == "could not accept request: request not found or was already accepted" {
			writeError(w, http.StatusConflict, "Request already accepted")
			return
		}
		writeError(w, http.StatusInternalServerError, "Could not accept request")
		return
	}

	writeJSON(w, http.StatusOK, req)
}

// handleResolveRequest allows an expert to mark a request as resolved.
func (h *Handler) handleResolveRequest(w http.ResponseWriter, r *http.Request) {
	expertID := uuid.New() // Placeholder
	// expertID, err := auth.GetExpertID(r.Context()) ...

	var payload ResolveRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	reqID, _ := uuid.Parse(payload.RequestID) // TODO: Handle parse error.

	err := h.service.ResolveRequest(r.Context(), reqID, expertID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not resolve request")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

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
