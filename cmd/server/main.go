package main

import (
	"log"
	"net/http"
	"os"

	"github.com/thealienbit/authservicebyalien/internal/auth"
	"github.com/thealienbit/authservicebyalien/internal/config"
	"github.com/thealienbit/authservicebyalien/internal/db"

	emailInt "github.com/thealienbit/authservicebyalien/internal/email"
	httpRouter "github.com/thealienbit/authservicebyalien/internal/http"
)

func main() {
	cfg := config.Load()

	dbPool := db.NewPostgresPool(cfg.DatabaseURL)
	defer dbPool.Close()

	key, err := auth.GenerateRSAKey("auth-key-1")
	if err != nil {
		log.Fatal(err)
	}

	jwtIssuer, err := auth.NewJWTIssuer(
		"auth-service",
		"your-app",
		key,
	)
	if err != nil {
		log.Fatal(err)
	}

	emailSender := emailInt.NewSMTPSender(
		"smtp.gmail.com",
		587,
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASS"),
		"no-reply@yourapp.com",
	)



	router := httpRouter.NewRouter(dbPool, jwtIssuer, []*auth.SigningKey{key}, emailSender)

	log.Printf("auth service starting on :%s\n", cfg.HTTPPort)
	log.Fatal(http.ListenAndServe(":"+cfg.HTTPPort, router))
}
