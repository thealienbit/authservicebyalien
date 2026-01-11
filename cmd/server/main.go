package main

import (
	"log"
	"net/http"

	"github.com/thealienbit/authservicebyalien/internal/config"
	"github.com/thealienbit/authservicebyalien/internal/db"

	httpRouter "github.com/thealienbit/authservicebyalien/internal/http"
)

func main() {
	cfg := config.Load()

	dbPool := db.NewPostgresPool(cfg.DatabaseURL)
	defer dbPool.Close()

	router := httpRouter.NewRouter(dbPool)

	log.Printf("auth service starting on :%s\n", cfg.HTTPPort)
	log.Fatal(http.ListenAndServe(":"+cfg.HTTPPort, router))
}
