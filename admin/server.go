package admin

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/garbaczdev/authscope/store"
)

const (
	// Maximum number of keys listed on a page
	maxKeysPerPage=10
	tokenCookie="key_token"
)

//go:embed templates/*.html
var templateFiles embed.FS

type Store interface {
	FindKeyByToken(token string) (store.Key, bool, error)
	Get(keyID uint) (store.Key, bool, error)
	Delete(keyID uint) error
	Create(key store.Key) (store.Key, string, error)
	Update(key store.Key) (store.Key, error)
	List(offset, limit int) ([]store.Key, int, error)
	Search(keywords []string, offset, limit int) ([]store.Key, int, error)
}


type server struct {
	store Store
	logger *slog.Logger
	tmpl *template.Template
}

func (s *server) handleHome(w http.ResponseWriter, r *http.Request) {
	// search keywords provided by the user (default is empty)
	searchKeywords := []string{}
	// current page (default 1)
	page := 1

	// Extract URL query string
	q := r.URL.Query()

	// Handle search queries
	qSearch := strings.TrimSpace(q.Get("q"))
	if len(qSearch) > 0 {
		searchKeywords = strings.Split(
			strings.ToLower(qSearch),
			" ",
		)
	}

	// Handle pagination
	qPage := strings.TrimSpace(q.Get("page"))
	if len(qPage) > 0 {
		p, err := strconv.ParseUint(qPage, 10, 32)
		if err != nil || p <= 0 {
			http.Error(w, "malformed page number", http.StatusBadRequest)
			return
		}
		page = int(p)
	}

	keys, total, err := s.store.Search(searchKeywords, (page - 1) * maxKeysPerPage, maxKeysPerPage)
	if err != nil {
		s.logger.Error("index search keys", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Derive the pagination bounds so the template stays arithmetic-free.
	totalPages := (total + maxKeysPerPage - 1) / maxKeysPerPage
	if totalPages < 1 {
		totalPages = 1
	}

	s.render(
		w,
		http.StatusOK,
		"index",
		struct{
			Keys []store.Key
			SearchQuery string
			Page int
			PrevPage int
			NextPage int
			TotalPages int
			HasPrev bool
			HasNext bool
		}{
			Keys: keys,
			SearchQuery: qSearch,
			Page: page,
			PrevPage: page - 1,
			NextPage: page + 1,
			TotalPages: totalPages,
			HasPrev: page > 1,
			HasNext: page < totalPages,
		},
	)
}

func (s *server) handleKeyNewForm(w http.ResponseWriter, r *http.Request) {
	// Mock key given into the form by default
	mockKey := store.Key{
		IsActive: true,
		Environment: "",
		Source: "",
		Description: "",
	}

	s.render(
		w,
		http.StatusOK,
		"keyform",
		struct{
			Key store.Key
			Title string
			FormHeader string
			Errors []string
		}{
			Key: mockKey,
			Title: "authscope - new key",
			FormHeader: "Create a new key",
			Errors: []string{},
		},
	)
}

func (s *server) handleKeyNewSubmit(w http.ResponseWriter, r *http.Request) {
	// Try to parse the submitted form.
	if err := r.ParseForm(); err != nil {
		http.Error(w, "malformed form", http.StatusBadRequest)
		return
	}

	// Extract form data into a new key
	key := store.Key{}
	ok := s.parseFormIntoKey(w, r, &key)
	if !ok { return }

	// Re-render the form if the submitted key is incorrect
	ok = s.ensureCorrectKeyForm(
		w,
		r,
		key,
		"authscope - new key",
		"Create a new key",
	)
	if !ok { return }

	// Create key in the datastore
	createdKey, token, err := s.store.Create(key)
	if err != nil {
		s.logger.Error("create key", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Set the token as a cookie to be deleted later.
	// This is a method to have a stateless backend while preventing the 
	// user from creating multiple keys by accident.
	http.SetCookie(w, &http.Cookie{
		Name:     tokenCookie,
		Value:    token,
		Path:     "/key/",
		MaxAge:   60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to the key page which should show the token
	http.Redirect(w, r, fmt.Sprintf("/key/%d", createdKey.GetID()), http.StatusSeeOther)
}

func (s *server) handleKeyPage(w http.ResponseWriter, r *http.Request) {
	// Get the key based on the id provided in the path.
	key, ok := s.getKeyFromPath(w, r)
	if !ok {
		return
	}

	// Check whether the token is provided as cookie (key just created)
	c, err := r.Cookie(tokenCookie)
	token := ""

	if err == nil && len(c.Value) > 0 {
		// The token is present as a cookie - user just created that key.
		token = c.Value
		// Clear the token cookie as soon as it's read.
		http.SetCookie(w, &http.Cookie{
			Name:     tokenCookie,
			Value:    "",
			Path:     "/key/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	// Render key page
	s.render(
		w,
		http.StatusOK,
		"key",
		struct{
			Key store.Key
			Token string
		}{
			Key: key,
			Token: token,
		},
	)
}

func (s *server) handleKeyEditForm(w http.ResponseWriter, r *http.Request) {
	// Get the key based on the id provided in the path.
	key, ok := s.getKeyFromPath(w, r)
	if !ok {
		return
	}

	// Render the key form
	s.render(
		w,
		http.StatusOK,
		"keyform",
		struct{
			Key store.Key
			Title string
			FormHeader string
			Errors []string
		}{
			Key: key,
			Title: fmt.Sprintf("authscope - edit key %d", key.GetID()),
			FormHeader: fmt.Sprintf("Edit Key %d", key.GetID()),
			Errors: []string{},
		},
	)
}

func (s *server) handleKeyEditSubmit(w http.ResponseWriter, r *http.Request) {
	// Get the key based on the id provided in the path.
	key, ok := s.getKeyFromPath(w, r)
	if !ok { return }

	// Extract form data into the key
	ok = s.parseFormIntoKey(w, r, &key)
	if !ok { return }

	// If key is incorrect, it will re-render the form!
	ok = s.ensureCorrectKeyForm(
		w,
		r,
		key,
		fmt.Sprintf("authscope - edit key %d", key.GetID()),
		fmt.Sprintf("Edit Key %d", key.GetID()),
	)
	if !ok { return }

	// Create key in the datastore
	updatedKey, err := s.store.Update(key)
	if err != nil {
		s.logger.Error("edit key submit update", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Redirect to the key page which should show the token
	http.Redirect(w, r, fmt.Sprintf("/key/%d", updatedKey.GetID()), http.StatusSeeOther)
}

func (s *server) handleKeyDelete(w http.ResponseWriter, r *http.Request) {
	// Get the key based on the id provided in the path.
	key, ok := s.getKeyFromPath(w, r)
	if !ok {
		return
	}

	// Delete the key from the store
	err := s.store.Delete(key.GetID())
	if err != nil {
		s.logger.Error("delete key delete", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Return to the main page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// render buffers the output so a mid-render template error doesn't leave a
// half-written 200 on the wire.
func (s *server) render(w http.ResponseWriter, status int, templateName string, data any) {
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		s.logger.Error("render", "template", templateName, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// getKeyFromPath extracts the key based on the {id} provided in the path.
// If key has been successfully extracted, it returns that key along with ok=true.
// If key could not be extracted for some reason, it answers the request returns ok=false.
func (s *server) getKeyFromPath(w http.ResponseWriter, r *http.Request) (key store.Key, ok bool) {
	// Extract the key id from the path
	keyID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return store.Key{}, false
	}

	// Retrieve the key from the store
	key, ok, err = s.store.Get(uint(keyID))
	if err != nil {
		s.logger.Error("get key from path", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return store.Key{}, false
	}
	if !ok {
		http.NotFound(w, r)
		return store.Key{}, false
	}

	return key, true
}

// parseFormIntoKey parses the request form into the key struct.
// If key was successfully parsed and is validated, it returns ok=true.
// If key is incorrect, it will respond to the request and return ok=false.
func (s *server) parseFormIntoKey(w http.ResponseWriter, r *http.Request, key *store.Key) (ok bool) {
	// Try to parse the submitted form.
	if err := r.ParseForm(); err != nil {
		http.Error(w, "malformed form", http.StatusBadRequest)
		return false
	}

	// Set the values to the key
	_, key.IsActive = r.PostForm["isActive"]
	key.Environment = strings.ToLower(strings.TrimSpace(r.PostForm.Get("environment")))
	key.Source = strings.ToLower(strings.TrimSpace(r.PostForm.Get("source")))
	key.Description = strings.TrimSpace(r.PostForm.Get("description"))

	return true
}

// ensureCorrectKeyForm checks whether the provided key is correct.
// If the provided key is correct it returns ok=true.
// If the provided key is invalid, it responds to the request with an error-formatted form.
func (s *server) ensureCorrectKeyForm(
	w http.ResponseWriter,
	r *http.Request,
	key store.Key,
	pageTitle string,
	formHeader string,
) (ok bool) {
	// Validate the submitted fields
	validationErrors := key.ValidateAll()

	if len(validationErrors) > 0 {
		// Convert errors to strings
		errorStrings := make([]string, len(validationErrors))
		for idx := range validationErrors {
			errorStrings[idx] = validationErrors[idx].Error()
		}
		// Render page with errors
		s.render(
			w,
			http.StatusOK,
			"keyform",
			struct{
				Key store.Key
				Title string
				FormHeader string
				Errors []string
			}{
				Key: key,
				Title: pageTitle,
				FormHeader: formHeader,
				Errors: errorStrings,
			},
		)
		return false
	}

	return true
}

type Option func(s *server) error

func New(reg Store, opts ...Option) (http.Handler, error) {
	// Load in the templates
	tmpl, err := template.New("").ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("could not load templates: %w", err)
	}

	// Create a server with default settings
	s := &server{
		store: reg,
		logger: slog.Default(),
		tmpl: tmpl,
	}

	// Apply options to the server
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, fmt.Errorf("option returned an error: %w", err)
		}
	}

	// Register routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleHome)
	mux.HandleFunc("GET /key/new", s.handleKeyNewForm)
	mux.HandleFunc("POST /key/new", s.handleKeyNewSubmit)
	mux.HandleFunc("GET /key/{id}", s.handleKeyPage)
	mux.HandleFunc("GET /key/{id}/edit", s.handleKeyEditForm)
	mux.HandleFunc("POST /key/{id}/edit", s.handleKeyEditSubmit)
	mux.HandleFunc("POST /key/{id}/delete", s.handleKeyDelete)

	return mux, nil
}


func isSafeCharacter(char byte) bool {
	// number
	if 48 <= char && char <= 57 { return true }
	// lowercase letter
	if 97 <= char && char <= 122 { return true }
	// -
	if char == 45 { return true }
	// _
	if char == 95 { return true }
	// .
	if char == 46 { return true }
	return false
}
