package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"project-sage/internal/llm" // The internal package for this service

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// main is the entry point for the LLMGatewayService.
func main() {

	// This service depends on clients for other services.

	// Get external service URLs. For now, they aren't used by the stubs.
	_ = os.Getenv("CHAT_GATEWAY_URL") // eg "http://chatgateway:8084"
	_ = os.Getenv("GEMINI_API_KEY")   // This will be needed for the real client

	// Stub clients for now.
	geminiClient := llm.NewStubGeminiClient()
	chatClient := llm.NewStubChatGatewayClient()
	// For real implementation:
	// geminiClient := llm.NewRealGeminiClient(geminiKey)
	// chatClient := llm.NewHTTPChatGatewayClient(chatGatewayURL)

	// Inject clients into the service
	llmService := llm.NewService(geminiClient, chatClient)

	// Inject service into the handler
	llmHandler := llm.NewHandler(llmService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("LLMGatewayService OK"))
	})

	// Register all the API routes from the handler ( /chat/social, /chat/summarize )
	llmHandler.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	log.Printf("LLMGatewayService starting on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
