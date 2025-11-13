package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"project-sage/internal/billing" // internal package for billing logic

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// main is the entry point for the BillingService.
func main() {
	// Can't do anything without a database, so check this first.
	connStr := os.Getenv("DB_CONNECTION_STRING")
	if connStr == "" {
		log.Fatal("DB_CONNECTION_STRING environment variable is not set")
	}

	db, err := connectDB(connStr)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	defer db.Close() // Make sure this closes on exit.
	log.Println("Database connected!")

	// manual dependency injection
	// create the repository, pass it to the service, and pass the service to the handler.
	billingRepo := billing.NewPostgresRepository(db)
	billingService := billing.NewService(billingRepo)
	billingHandler := billing.NewHandler(billingService)

	// Set up the router
	r := chi.NewRouter()
	r.Use(middleware.Logger)    // Log requests
	r.Use(middleware.Recoverer) // For any panics

	// Basic health check endpoint.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("BillingService OK"))
	})

	// Let the handler set up its routes (like /token/debit).
	billingHandler.RegisterRoutes(r)

	// Get the port from env or use the default.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // Default for BillingService
	}

	log.Printf("BillingService starting on port %s", port)

	// Start the server and block.
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}

// connectDB is a helper to open and verify the database connection.
func connectDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}
	// Ping() to make sure the connection is actually valid.
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
