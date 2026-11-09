package portal

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/thealienbit/authservicebyalien/internal/store"
)

type orgsPageData struct {
	Page
	App  *store.App
	Orgs []*store.Organization
}

func (h *Handler) OrgsPage(w http.ResponseWriter, r *http.Request) {
	appID, err := uuid.Parse(chi.URLParam(r, "appID"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	app, err := h.store.GetAppByID(ctx, appID)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	orgs, err := h.store.ListOrganizationsByApp(ctx, appID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.render(w, http.StatusOK, "orgs", h.orgsData(r, app, orgs))
}

func (h *Handler) CreateOrg(w http.ResponseWriter, r *http.Request) {
	appID, err := uuid.Parse(chi.URLParam(r, "appID"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	name := strings.TrimSpace(r.PostFormValue("name"))
	if name == "" {
		http.Redirect(w, r, "/portal/apps/"+appID.String()+"/orgs", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := h.store.GetAppByID(ctx, appID); errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, err := h.store.CreateOrganization(ctx, appID, name); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/portal/apps/"+appID.String()+"/orgs", http.StatusSeeOther)
}

func (h *Handler) orgsData(r *http.Request, app *store.App, orgs []*store.Organization) orgsPageData {
	session, _ := sessionFromContext(r.Context())

	return orgsPageData{
		Page: Page{Email: session.Email, CSRF: session.CSRF},
		App:  app,
		Orgs: orgs,
	}
}
