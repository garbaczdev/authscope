package verify

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/garbaczdev/authscope/store"
)

// fakeStore records the tokens it is asked about and returns canned results.
type fakeStore struct {
	key    store.Key
	found  bool
	err    error
	tokens []string
}

func (f *fakeStore) FindKeyByToken(token string) (store.Key, bool, error) {
	f.tokens = append(f.tokens, token)
	return f.key, f.found, f.err
}

func TestVerifyAuthorizesActiveKey(t *testing.T) {
	const token = "authscope-abc123"
	st := &fakeStore{
		key:   store.Key{IsActive: true, Environment: "prod", Source: "web"},
		found: true,
	}
	h, err := New(st)
	if err != nil {
		t.Fatalf("could not create verify handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/verify", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Source"); got != "web" {
		t.Errorf("X-Source = %q, want %q", got, "web")
	}
	if got := rec.Header().Get("X-Env"); got != "prod" {
		t.Errorf("X-Env = %q, want %q", got, "prod")
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
	if len(st.tokens) != 1 || st.tokens[0] != token {
		t.Errorf("store saw tokens %q, want [%q]", st.tokens, token)
	}
}

func TestVerifyRejections(t *testing.T) {
	tests := []struct {
		name  string
		auth  string
		found bool
		key   store.Key
	}{
		{"missing header", "", true, store.Key{IsActive: true, Source: "web", Environment: "prod"}},
		{"wrong scheme", "Basic abc123", true, store.Key{IsActive: true, Source: "web", Environment: "prod"}},
		{"empty token", "Bearer ", true, store.Key{IsActive: true, Source: "web", Environment: "prod"}},
		{"unknown token", "Bearer nope", false, store.Key{}},
		{"inactive key", "Bearer authscope-abc", true, store.Key{IsActive: false, Source: "web", Environment: "prod"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := &fakeStore{key: tc.key, found: tc.found}
			h, err := New(st)
			if err != nil {
				t.Fatalf("could not create verify handler: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, "/verify", nil)
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if got := rec.Header().Get("WWW-Authenticate"); got != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want %q", got, "Bearer")
			}
			if rec.Header().Get("X-Source") != "" || rec.Header().Get("X-Env") != "" {
				t.Error("identity headers set on a rejected request")
			}
			if rec.Body.Len() != 0 {
				t.Errorf("body = %q, want empty", rec.Body.String())
			}
		})
	}
}

func TestVerifyStoreErrorFailsClosed(t *testing.T) {
	st := &fakeStore{err: errors.New("boom")}
	h, err := New(st)
	if err != nil {
		t.Fatalf("could not create verify handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/verify", nil)
	req.Header.Set("Authorization", "Bearer authscope-abc")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if rec.Header().Get("X-Source") != "" || rec.Header().Get("X-Env") != "" {
		t.Error("identity headers set despite a store failure")
	}
}

func TestHealthz(t *testing.T) {
	h, err := New(&fakeStore{})
	if err != nil {
		t.Fatalf("could not create verify handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
		ok     bool
	}{
		{"well formed", "Bearer authscope-abc", "authscope-abc", true},
		{"scheme is case-insensitive", "bearer authscope-abc", "authscope-abc", true},
		{"empty header", "", "", false},
		{"no scheme", "authscope-abc", "", false},
		{"wrong scheme", "Basic authscope-abc", "", false},
		{"missing token", "Bearer", "", false},
		{"empty token", "Bearer ", "", false},
		{"token with space", "Bearer a b", "a b", true},
		{"token with tab", "Bearer a\tb", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := bearerToken(tc.header)
			if ok != tc.ok {
				t.Fatalf("ok = %t, want %t", ok, tc.ok)
			}
			if got != tc.want {
				t.Errorf("token = %q, want %q", got, tc.want)
			}
		})
	}
}
