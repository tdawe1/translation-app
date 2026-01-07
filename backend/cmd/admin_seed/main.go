// Command admin_seed creates or updates an admin user in the database
// and outputs a valid JWT token for testing.
//
// Usage:
//
//	go run ./cmd/admin_seed <email> <password>
//
// Example:
//
//	go run ./cmd/admin_seed admin@example.com SecurePass123!
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/seeds"
)

func main() {
	// Parse command line flags
	email := flag.String("email", "admin@example.com", "Admin user email")
	password := flag.String("password", "", "Admin user password (required)")
	flag.Parse()

	// Password is required
	if *password == "" {
		fmt.Fprintln(os.Stderr, "Error: password is required")
		fmt.Fprintln(os.Stderr, "\nUsage:")
		fmt.Fprintln(os.Stderr, "  go run ./cmd/admin_seed -email <email> -password <password>")
		fmt.Fprintln(os.Stderr, "\nExample:")
		fmt.Fprintln(os.Stderr, "  go run ./cmd/admin_seed -email admin@example.com -password SecurePass123!")
		os.Exit(1)
	}

	// Load config from environment
	cfg := config.Load()

	// Connect to database
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create token service
	tokenSvc := auth.NewTokenService(cfg.JWTSecret)

	// Create admin seeder
	seeder := seeds.NewAdminSeeder(db, tokenSvc)

	// Ensure admin user exists
	user, token, err := seeder.EnsureAdminUser(*email, *password)
	if err != nil {
		log.Fatalf("Failed to ensure admin user: %v", err)
	}

	// Output results
	fmt.Println("Admin user created/updated successfully!")
	fmt.Printf("Email: %s\n", user.Email)
	fmt.Printf("Role: %s\n", user.Role)
	fmt.Printf("User ID: %s\n", user.ID)
	fmt.Printf("\nJWT Token (valid for 15 minutes):\n%s\n", token)
}
