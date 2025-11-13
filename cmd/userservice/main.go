package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"project-sage/internal/user" // internal package for user logic

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// main is the entry point for the UserService.
func main() {
	// Need the database connection string. Fail fast if it's not set.
	connStr := os.Getenv("DB_CONNECTION_STRING")
	if connStr == "" {
		log.Fatal("DB_CONNECTION_STRING environment variable is not set")
	}

	db, err := connectDB(connStr)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	defer db.Close() // Make sure the connection is closed on exit.
	log.Println("Database connected!")

	// This is the dependency injection part, done manually.
	// We create each layer and pass it to the next.

	//  Data access layer.
	userRepo := user.NewPostgresRepository(db)

	// business logic layer.
	userService := user.NewService(userRepo)

	// API layer. Takes the service.
	userHandler := user.NewHandler(userService)

	// Set up the chi router.
	r := chi.NewRouter()

	// Add standard middleware.
	r.Use(middleware.Logger)    // Log requests
	r.Use(middleware.Recoverer) // Handle panics gracefully

	// Simple health check.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("UserService OK"))
	})

	// Let the handler define all its specific routes (eg /users/register).
	userHandler.RegisterRoutes(r)

	// Find the port to run on, or use a default.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default for UserService
	}

	log.Printf("UserService starting on port %s", port)

	// Start the server and block until it errors or is stopped.
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
	// Ping() ensures the connection is actually valid.
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
