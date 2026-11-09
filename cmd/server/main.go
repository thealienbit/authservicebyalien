package main

import (
	"context"
	"log"
	"net/http"

	"github.com/thealienbit/authservicebyalien/internal/auth"
	"github.com/thealienbit/authservicebyalien/internal/config"
	"github.com/thealienbit/authservicebyalien/internal/db"
	"github.com/thealienbit/authservicebyalien/internal/signing"
	"github.com/thealienbit/authservicebyalien/internal/store"
	"github.com/thealienbit/authservicebyalien/migrations"

	emailInt "github.com/thealienbit/authservicebyalien/internal/email"
	httpRouter "github.com/thealienbit/authservicebyalien/internal/http"
)

func main() {
	cfg := config.Load()

	dbPool := db.NewPostgresPool(cfg.DatabaseURL)
	defer dbPool.Close()

	ctx := context.Background()

	if err := migrations.Run(ctx, dbPool); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	keyCipher, err := signing.NewCipher(cfg.JWTKEK)
	if err != nil {
		log.Fatal(err)
	}

	appStore := store.New(dbPool, keyCipher)
	if _, err := appStore.EnsureSigningKey(ctx); err != nil {
		log.Fatalf("signing key: %v", err)
	}

	signingKeys, err := appStore.LoadSigningKeys(ctx)
	if err != nil {
		log.Fatalf("signing keys: %v", err)
	}

	activeKey := signing.Active(signingKeys)
	if activeKey == nil {
		log.Fatal("signing keys: no active key")
	}

	jwtIssuer, err := auth.NewJWTIssuer(
		"auth-service",
		activeKey,
	)
	if err != nil {
		log.Fatal(err)
	}

	emailSender := emailInt.NewSMTPSender(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPass,
		cfg.SMTPFrom,
	)



	trustedProxies, err := httpRouter.ParseTrustedProxies(cfg.TrustedProxies)
	if err != nil {
		log.Fatal(err)
	}

	router := httpRouter.NewRouter(httpRouter.Deps{
		DB:             dbPool,
		JWTIssuer:      jwtIssuer,
		SigningKeys:    signingKeys,
		EmailSender:    emailSender,
		Store:          appStore,
		TrustedProxies: trustedProxies,
		SecureCookies:  cfg.AppEnv != "development",
	})

	log.Printf("auth service starting on :%s\n", cfg.HTTPPort)
	log.Fatal(http.ListenAndServe(":"+cfg.HTTPPort, router))
}
