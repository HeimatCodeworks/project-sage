package chat

import (
	"encoding/json"
	"net/http"
	"project-sage/internal/domain"

	// "project-sage/internal/auth" // We'll need this for real auth
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler is the HTTP API layer for the ChatGatewayService.
type Handler struct {
	service Service
	// We also need a UserService client here to fetch user/expert profiles userSvcClient auth.UserServiceClient // (or similar)
}

// NewHandler creates a new handler.
func NewHandler(s Service) *Handler {
	return &Handler{
		service: s,
	}
}

// RegisterRoutes attaches all chat-related endpoints to the router.
func (h *Handler) RegisterRoutes(r chi.Router) {

	// This one endpoint is for both users and experts.
	// The auth middleware will tell us which one they are.
	r.Post("/chat/token", h.handleGenerateToken)

	// Called by RequestService
	r.Post("/chat/remove-bot", h.handleRemoveBot)
	r.Post("/chat/add-expert", h.handleAddExpert)

	// Called by LLMGatewayService
	r.Get("/chat/history/{sid}", h.handleGetChatHistory)

}

// --- DTOs ---

type tokenResponse struct {
	Token string `json:"token"`
}

type addExpertRequest struct {
	TwilioConversationSID string `json:"twilio_conversation_sid"`
	ExpertID              string `json:"expert_id"`
}

type removeBotRequest struct {
	TwilioConversationSID string `json:"twilio_conversation_sid"`
}

// handleGenerateToken generates a Twilio token for the authenticated user
func (h *Handler) handleGenerateToken(w http.ResponseWriter, r *http.Request) {
	// userID, userErr := auth.GetUserID(r.Context())
	// expertID, expertErr := auth.GetExpertID(r.Context())

	// --- This is a placeholder for auth ---
	// Faking it by looking for a query param
	userID_str := r.URL.Query().Get("user_id")
	expertID_str := r.URL.Query().Get("expert_id")
	// --- End placeholder ---

	var token string
	var err error

	if userID_str != "" {
		// This is a Standard User
		// We'd normally fetch the user object from the UserService
		// faking it here
		userID, _ := uuid.Parse(userID_str)
		fakeUser := &domain.User{UserID: userID}
		token, err = h.service.GenerateUserToken(r.Context(), fakeUser)

	} else if expertID_str != "" {
		// This is an Expert User
		// We'd fetch the expert object
		expertID, _ := uuid.Parse(expertID_str)
		fakeExpert := &domain.Expert{ExpertID: expertID}
		token, err = h.service.GenerateExpertToken(r.Context(), fakeExpert)

	} else {
		writeError(w, http.StatusUnauthorized, "Not authorized")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not generate token")
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{Token: token})
}

// handleRemoveBot is an internal endpoint to remove the bot.
func (h *Handler) handleRemoveBot(w http.ResponseWriter, r *http.Request) {
	var req removeBotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err := h.service.RemoveBot(r.Context(), req.TwilioConversationSID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not remove bot")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "bot_removed"})
}

// handleAddExpert is an internal endpoint to add an expert.
func (h *Handler) handleAddExpert(w http.ResponseWriter, r *http.Request) {
	var req addExpertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	expertID, err := uuid.Parse(req.ExpertID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid expert_id format")
		return
	}

	err = h.service.AddExpert(r.Context(), req.TwilioConversationSID, expertID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not add expert")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "expert_added"})
}

// handleGetChatHistory is an internal endpoint for the LLMGatewayService.
func (h *Handler) handleGetChatHistory(w http.ResponseWriter, r *http.Request) {
	// We get the SID from the URL path, eg /chat/history/CH123
	sid := chi.URLParam(r, "sid")
	if sid == "" {
		writeError(w, http.StatusBadRequest, "Missing conversation SID")
		return
	}

	history, err := h.service.GetChatHistory(r.Context(), sid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not fetch history")
		return
	}

	writeJSON(w, http.StatusOK, history)
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
