package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"project-sage/internal/chat"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// main is the entry point for the ChatGatewayService.
func main() {

	// This service's main dependency is the Twilio client.
	_ = os.Getenv("TWILIO_ACCOUNT_SID")
	_ = os.Getenv("TWILIO_AUTH_TOKEN")
	_ = os.Getenv("TWILIO_API_KEY")
	_ = os.Getenv("TWILIO_API_SECRET")

	// We'll use our stub client for now.
	twilioClient := chat.NewStubTwilioClient()
	// When we build the real client, we'll swap this line, eg:
	// twilioClient := chat.NewRealTwilioClient(accountSID, authToken, ...)

	// Inject the client into the service
	chatService := chat.NewService(twilioClient)

	// Inject service into the handler
	chatHandler := chat.NewHandler(chatService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ChatGatewayService OK"))
	})

	// Register all the API routes from the handler
	chatHandler.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	log.Printf("ChatGatewayService starting on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
