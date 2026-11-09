package portal

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/thealienbit/authservicebyalien/internal/auth"
	"github.com/thealienbit/authservicebyalien/internal/store"
)

type usersPageData struct {
	Page
	Org   *store.Organization
	Users []*store.OrgUser
}

func (h *Handler) UsersPage(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	org, err := h.store.GetOrganizationByID(ctx, orgID)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	users, err := h.store.ListUsersByOrganization(ctx, orgID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.render(w, http.StatusOK, "users", h.usersData(r, org, users))
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	org, err := h.store.GetOrganizationByID(ctx, orgID)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	email := strings.ToLower(strings.TrimSpace(r.PostFormValue("email")))
	password := r.PostFormValue("password")

	users, listErr := h.store.ListUsersByOrganization(ctx, orgID)
	if listErr != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data := h.usersData(r, org, users)
	if email == "" || len(password) < 8 {
		data.Error = "Email and a password of at least 8 characters are required."
		h.render(w, http.StatusBadRequest, "users", data)
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_, err = h.store.CreateUser(ctx, store.NewOrgUser{
		AppID:          org.AppID,
		OrganizationID: org.ID,
		Role:           store.RoleSuperAdmin,
		Email:          email,
		PasswordHash:   passwordHash,
	})
	if errors.Is(err, store.ErrConflict) {
		data.Error = "A user with that email already exists in this app."
		h.render(w, http.StatusConflict, "users", data)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/portal/orgs/"+org.ID.String()+"/users", http.StatusSeeOther)
}

func (h *Handler) usersData(r *http.Request, org *store.Organization, users []*store.OrgUser) usersPageData {
	session, _ := sessionFromContext(r.Context())

	return usersPageData{
		Page:  Page{Email: session.Email, CSRF: session.CSRF},
		Org:   org,
		Users: users,
	}
}
