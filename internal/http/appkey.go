package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/thealienbit/authservicebyalien/internal/auth"
	"github.com/thealienbit/authservicebyalien/internal/store"
)

const appKeyHeader = "X-App-Key"

// AppKeyMiddleware authenticates the calling app. It is mounted on the
// /auth/v1 group only; JWKS, /health and the portal do not use it.
func AppKeyMiddleware(appStore *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimSpace(r.Header.Get(appKeyHeader))
			if key == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()

			app, err := appStore.GetAppByKeyHash(ctx, store.HashAppKey(key))
			switch {
			case errors.Is(err, store.ErrNotFound):
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			case err != nil:
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			if app.Status != store.AppStatusActive {
				http.Error(w, "app disabled", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithApp(r.Context(), app)))
		})
	}
}
