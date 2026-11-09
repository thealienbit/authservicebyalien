package portal

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/thealienbit/authservicebyalien/internal/store"
)

type appsPageData struct {
	Page
	Apps       []*store.App
	NewKey     string
	NewAppName string
}

func (h *Handler) AppsPage(w http.ResponseWriter, r *http.Request) {
	data, err := h.appsData(r)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.render(w, http.StatusOK, "apps", data)
}

func (h *Handler) CreateApp(w http.ResponseWriter, r *http.Request) {
	data, err := h.appsData(r)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	name := strings.TrimSpace(r.PostFormValue("name"))
	baseURL := strings.TrimSpace(r.PostFormValue("base_url"))
	if name == "" || baseURL == "" {
		data.Error = "Name and base URL are required."
		h.render(w, http.StatusBadRequest, "apps", data)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	app, key, err := h.store.CreateApp(ctx, name, baseURL)
	if errors.Is(err, store.ErrConflict) {
		data.Error = "An app with that key already exists. Try again."
		h.render(w, http.StatusConflict, "apps", data)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data, err = h.appsData(r)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	data.NewKey = key
	data.NewAppName = app.Name

	h.render(w, http.StatusCreated, "apps", data)
}

func (h *Handler) appsData(r *http.Request) (appsPageData, error) {
	session, _ := sessionFromContext(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	apps, err := h.store.ListApps(ctx)
	if err != nil {
		return appsPageData{}, err
	}

	return appsPageData{
		Page: Page{Email: session.Email, CSRF: session.CSRF},
		Apps: apps,
	}, nil
}
