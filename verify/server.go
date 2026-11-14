// Package verify implements the forward-auth decision endpoint. The reverse
// proxy calls GET /verify for every incoming request; this package reads the
// bearer token, authorizes it against the key store, and, on success, hands the
// sender's identity back to the proxy as X-Source and X-Env response headers.
package verify

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/garbaczdev/authscope/store"
)

// bearerScheme is the only authentication scheme the endpoint accepts.
const bearerScheme = "Bearer"

type Store interface {
	FindKeyByToken(token string) (store.Key, bool, error)
}

type server struct {
	store  Store
	logger *slog.Logger
}

// handleVerify authorizes the request's bearer token. On success it writes the
// key's Source and Environment as X-Source and X-Env response headers. The body
// is always empty.
func (s *server) handleVerify(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		s.unauthorized(w)
		return
	}

	key, found, err := s.store.FindKeyByToken(token)
	if err != nil {
		s.logger.Error("verify find key by token", "err", err)
		// Fail closed: answer with 5xx (no identity headers) so the proxy
		// applies its "treat 5xx as denial" rule.
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// A revoked (inactive) key must not authorize, even though the store still
	// returns it by token.
	if !found || !key.IsActive {
		s.unauthorized(w)
		return
	}

	w.Header().Set("X-Source", key.Source)
	w.Header().Set("X-Env", key.Environment)
	w.WriteHeader(http.StatusOK)
}

// handleHealthz reports that the service is ready to answer /verify.
func (s *server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// unauthorized answers 401 with the WWW-Authenticate challenge and an empty
// body. It never echoes the token.
func (s *server) unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", bearerScheme)
	w.WriteHeader(http.StatusUnauthorized)
}

// bearerToken extracts the token from "Authorization" header value,
// which should be in the format: "Bearer <token>".
// It returns ok=false when the header is absent, uses another scheme, or carries
// a missing or malformed token.
func bearerToken(header string) (token string, ok bool) {
	scheme, value, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, bearerScheme) {
		return "", false
	}

	// Remove OWS
	value = strings.Trim(value, " \t")

	// The token must be non-empty and free of further whitespace.
	if value == "" || strings.Contains(value, "\t") {
		return "", false
	}
	return value, true
}

// Option configures the verify server.
type Option func(s *server) error

// New builds the verify handler, serving GET /verify and GET /healthz.
func New(st Store, opts ...Option) (http.Handler, error) {
	if st == nil {
		return nil, errors.New("verify: store must not be nil")
	}

	// Create a server with default settings
	s := &server{
		store:  st,
		logger: slog.Default(),
	}

	// Apply options to the server
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, fmt.Errorf("option returned an error: %w", err)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /verify", s.handleVerify)
	mux.HandleFunc("GET /healthz", s.handleHealthz)

	return mux, nil
}
