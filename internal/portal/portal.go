package portal

import (
	"embed"
	"html/template"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/thealienbit/authservicebyalien/internal/store"
)

//go:embed templates/*.html
var templateFiles embed.FS

type Deps struct {
	Store           *store.Store
	SecureCookies   bool
	ClientIP        func(*http.Request) string
	LoginMiddleware func(http.Handler) http.Handler
}

type Handler struct {
	store           *store.Store
	secureCookies   bool
	clientIP        func(*http.Request) string
	loginMiddleware func(http.Handler) http.Handler
	templates       map[string]*template.Template
}

func New(deps Deps) *Handler {
	return &Handler{
		store:           deps.Store,
		secureCookies:   deps.SecureCookies,
		clientIP:        deps.ClientIP,
		loginMiddleware: deps.LoginMiddleware,
		templates:       loadTemplates(),
	}
}

type Page struct {
	Email string
	CSRF  string
	Error string
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/login", h.LoginPage)
	r.With(h.loginMiddleware).Post("/login", h.Login)

	r.Group(func(r chi.Router) {
		r.Use(h.requireSession)

		r.Post("/logout", h.requireCSRF(h.Logout))
		r.Get("/", h.AppsPage)
		r.Post("/apps", h.requireCSRF(h.CreateApp))
		r.Get("/apps/{appID}/orgs", h.OrgsPage)
		r.Post("/apps/{appID}/orgs", h.requireCSRF(h.CreateOrg))
		r.Get("/orgs/{orgID}/users", h.UsersPage)
		r.Post("/orgs/{orgID}/users", h.requireCSRF(h.CreateUser))
	})
}

func loadTemplates() map[string]*template.Template {
	pages := []string{"login", "apps", "orgs", "users"}
	templates := make(map[string]*template.Template, len(pages))

	for _, page := range pages {
		templates[page] = template.Must(template.ParseFS(
			templateFiles,
			"templates/layout.html",
			"templates/"+page+".html",
		))
	}

	return templates
}

func (h *Handler) render(w http.ResponseWriter, status int, page string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := h.templates[page].ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("portal: render %s: %v", page, err)
	}
}
