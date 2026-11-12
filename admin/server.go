package admin

import (
	"embed"
	"fmt"
	"bytes"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/garbaczdev/authscope/store"
)

//go:embed templates/*.html
var templateFiles embed.FS

type Store interface {
	Authorize(token string) (store.AuthInfo, error)
	Get(keyID uint) (store.Key, bool, error)
	Delete(keyID uint) error
	Create(key store.Key) (store.Key, string, error)
	Update(key store.Key) (store.Key, error)
	List(limit, offset int) ([]store.Key, error)
}


type server struct {
	store Store
	logger *slog.Logger
	tmpl *template.Template
}

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	keys, err := s.store.List(10, 0)
	if err != nil {
		s.logger.Error("list keys", "err", err)
	}

	s.render(
		w,
		http.StatusOK,
		"index.html",
		struct{
			Keys []store.Key
		}{
			Keys: keys,
		},
	)
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
	mux.HandleFunc("GET /{$}", s.list)

	return mux, nil
}

