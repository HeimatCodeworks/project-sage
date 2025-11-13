package llm

import (
	"encoding/json"
	"net/http"

	// "project-sage/internal/auth" // Placeholder for auth middleware

	"github.com/go-chi/chi/v5"
)

// Handler is the http api layer for the LLMGatewayService.
type Handler struct {
	service Service
}

// NewHandler creates a new handler injecting the service.
func NewHandler(s Service) *Handler {
	return &Handler{
		service: s,
	}
}

// RegisterRoutes attaches the llm endpoints to the router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	// User facing endpoint for social chat
	r.Post("/chat/social", h.handleSocialChat)

	// Internal endpoint for summarization
	r.Post("/chat/summarize", h.handleSummarizeChat)
}

// --- DTOs ---

// socialChatRequest is the DTO for what the client app sends.
type socialChatRequest struct {
	History []*ChatMessage `json:"history"`
}

// summarizeRequest is the DTO for what the RequestService sends.
type summarizeRequest struct {
	TwilioConversationSID string `json:"twilio_conversation_sid"`
}

// summarizeResponse is the DTO we send back to the RequestServce
type summarizeResponse struct {
	Summary string `json:"summary"`
}

// --- Handlers ---

// handleSocialChat handles requests for the general-purpose social chat.
func (h *Handler) handleSocialChat(w http.ResponseWriter, r *http.Request) {
	// TODO: Add auth middleware to get UserID
	// _, err := auth.GetUserID(r.Context())
	// if err != nil {
	// 	writeError(w, http.StatusUnauthorized, "Not authorized")
	// 	return
	// }

	var req socialChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Call the service with the provided history
	response, err := h.service.SocialChat(r.Context(), req.History)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not process chat")
		return
	}

	// Send back the single new message from the model
	writeJSON(w, http.StatusOK, response)
}

// handleSummarizeChat handles internal requests to summarize a chat
func (h *Handler) handleSummarizeChat(w http.ResponseWriter, r *http.Request) {
	// This is an internal service-to-service endpoint so it does not use user-facing auth middleware

	var req summarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	summary, err := h.service.SummarizeChatHistory(r.Context(), req.TwilioConversationSID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not summarize chat history")
		return
	}

	// Send back the summary string
	writeJSON(w, http.StatusOK, summarizeResponse{Summary: summary})
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
