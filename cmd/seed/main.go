package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"github.com/thealienbit/authservicebyalien/internal/auth"
	"github.com/thealienbit/authservicebyalien/internal/db"
	"github.com/thealienbit/authservicebyalien/internal/store"
	"github.com/thealienbit/authservicebyalien/migrations"
)

func main() {
	_ = godotenv.Load()

	email := flag.String("email", "", "platform admin email")
	password := flag.String("password", "", "platform admin password (min 8 chars)")
	flag.Parse()

	normalizedEmail := strings.ToLower(strings.TrimSpace(*email))
	if normalizedEmail == "" || len(*password) < 8 {
		flag.Usage()
		log.Fatal("seed: -email and a password of at least 8 characters are required")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	pool := db.NewPostgresPool(databaseURL)
	defer pool.Close()

	ctx := context.Background()
	if err := migrations.Run(ctx, pool); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	passwordHash, err := auth.HashPassword(*password)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	appStore := store.New(pool, nil)
	userID, err := appStore.EnsurePlatformAdmin(ctx, normalizedEmail, passwordHash)
	if err != nil {
		log.Fatalf("seed platform admin: %v", err)
	}

	fmt.Printf("platform admin ready: %s (%s)\n", normalizedEmail, userID)
}
