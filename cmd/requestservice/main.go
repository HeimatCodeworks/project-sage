package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"project-sage/internal/request" // The internal package for this service

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// main is the entry point for the RequestService.
// THis initializes dependencies and starts the HTTP server.
func main() {
	// Must get the database connection string from the environment.
	connStr := os.Getenv("DB_CONNECTION_STRING")
	if connStr == "" {
		log.Fatal("DB_CONNECTION_STRING not set")
	}

	db, err := connectDB(connStr)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Database connected!")

	// Initialize the repository.
	requestRepo := request.NewPostgresRepository(db)

	// Get external service urls from the environment.
	billingSvcURL := os.Getenv("BILLING_SERVICE_URL")
	llmSvcURL := os.Getenv("LLM_SERVICE_URL")
	chatSvcURL := os.Getenv("CHAT_SERVICE_URL")

	// Initialize the HTTP clients for other services.
	billingClient := request.NewHTTPBillingClient(billingSvcURL)
	llmClient := request.NewHTTPLLMClient(llmSvcURL)
	chatClient := request.NewHTTPChatClient(chatSvcURL)

	// Initialize the service, injecting dependencies.
	requestService := request.NewService(requestRepo, billingClient, llmClient, chatClient)

	// Initialize the handler.
	requestHandler := request.NewHandler(requestService)

	// Set up the chi router.
	r := chi.NewRouter()
	r.Use(middleware.Logger)    // Log incoming requests.
	r.Use(middleware.Recoverer) // Prevent panics from crashing the server.

	// Simple health check endpoint.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("RequestService OK"))
	})

	// Register all the API routes from the handler.
	requestHandler.RegisterRoutes(r)

	// Get the port from the environment. Use a default if not set.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("RequestService starting on port %s", port)

	// Block and run the web server.
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}

// connectDB opens and verifies the database connection.
func connectDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}
	// Ping() verifies the connection is actually alive.
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
