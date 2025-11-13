package user

import (
	"encoding/json"
	"net/http"

	// "project-sage/internal/auth" // For when auth exists

	"github.com/go-chi/chi/v5"
)

// Handler is the HTTP API layer for the UserService.
// It holds a dependency on the service layer.
type Handler struct {
	service Service
}

// NewHandler is the constructor for the Handler.
func NewHandler(s Service) *Handler {
	return &Handler{
		service: s,
	}
}

// RegisterRoutes attaches all the user related endpoints to the router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Endpoint for a new user to register their profile
	r.Post("/users/register", h.handleRegisterNewUser)

	// Endpoint for a user to fetch their own profile.
	r.Get("/users/profile", h.handleGetMyProfile)
}

// registerUserRequest is the DTO for the post /users/register endpoint.
type registerUserRequest struct {
	DisplayName string `json:"display_name"`
	ProfileURL  string `json:"profile_image_url"`
}

// handleRegisterNewUser handles the creation of a new user profile after they have authenticated with Firebase.
func (h *Handler) handleRegisterNewUser(w http.ResponseWriter, r *http.Request) {
	// This is a placeholder for real auth middleware.
	// The middleware should validate the token and put the ID in the context.
	firebaseID := r.Header.Get("X-Firebase-ID")
	if firebaseID == "" {
		writeError(w, http.StatusUnauthorized, "Missing auth token")
		return
	}

	// Decode the json request body into the DTO.
	var req registerUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Call the business logic layer to create the user.
	user, err := h.service.RegisterNewUser(r.Context(), firebaseID, req.DisplayName, req.ProfileURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not register user")
		return
	}

	// Return the newly created user object.
	writeJSON(w, http.StatusCreated, user)
}

// handleGetMyProfile fetches the profile for the authenticated user.
func (h *Handler) handleGetMyProfile(w http.ResponseWriter, r *http.Request) {
	// Placeholder for auth middleware.
	firebaseID := r.Header.Get("X-Firebase-ID")
	if firebaseID == "" {
		writeError(w, http.StatusUnauthorized, "Missing auth token")
		return
	}

	// Call the service to get the user's data.
	user, err := h.service.GetUser(r.Context(), firebaseID)
	if err != nil {
		// Handle the "not found" case.
		if err.Error() == "user not found" {
			writeError(w, http.StatusNotFound, "User profile not found")
			return
		}
		// Handle other potential database errors.
		writeError(w, http.StatusInternalServerError, "Could not retrieve profile")
		return
	}

	// Send the user profile as json.
	writeJSON(w, http.StatusOK, user)
}

// writeJSON is a helper function to send json formatted responses.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// writeError is a helper function to send a standardized json error message
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
