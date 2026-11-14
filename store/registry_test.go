package store

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestRegistry(t *testing.T) *Registry {
	// Make sure this does not get recognized as a test
	t.Helper()
	// Create a temporary directory for the database files
	dbPath := filepath.Join(t.TempDir(), "test.db")
	// Create the registry for testing
	reg, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("could not create the registry: %v", err)
	}
	return reg
}

func TestCreateDeleteKey(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	// Ensure there are no keys at the start
	keys, err := reg.List(10, 0)
	if err != nil {
		t.Fatalf("could not list keys: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("there are %d keys in a newly initialized db", len(keys))
	}

	// Create a key
	keyToCreate := Key{
		IsActive:    true,
		Environment: "env",
		Source:      "src",
		Description: "src",
	}
	createdKey, token, err := reg.Create(keyToCreate)
	if err != nil {
		t.Fatalf("could not create a key: %v", err)
	}

	// Ensure that there is a key in the database
	keys, err = reg.List(10, 0)
	if err != nil {
		t.Fatalf("could not list keys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("there are %d keys after creating 1 key", len(keys))
	}

	// Delete the created key
	if err = reg.Delete(createdKey.GetID()); err != nil {
		t.Fatalf("could not delete the created key: %v", err)
	}

	// Ensure the deletion was correct
	keys, err = reg.List(10, 0)
	if err != nil {
		t.Fatalf("could not list keys: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("there are %d keys after deleting created key", len(keys))
	}

	if _, ok, _ := reg.Get(createdKey.GetID()); ok {
		t.Fatalf("key returned by get after deletion")
	}

	// Check authorization after deletion
	authInfo, err := reg.Authorize(token)
	if err != nil {
		t.Fatalf("could not authorize the created key: %v", err)
	}
	if authInfo.IsAuthorized {
		t.Errorf("just deleted key is still authorized")
	}

}

func TestGetKey(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	// Create a key
	keyToCreate := Key{
		IsActive:    true,
		Environment: "env",
		Source:      "src",
		Description: "src",
	}
	createdKey, _, err := reg.Create(keyToCreate)
	if err != nil {
		t.Fatalf("could not create a key: %v", err)
	}

	// Get the created key
	fetchedKey, ok, err := reg.Get(createdKey.GetID())
	if err != nil {
		t.Fatalf("could not get the created key: %v", err)
	}
	if !ok {
		t.Fatalf("could not get the created key: not found")
	}

	// Compare whether all fields are correct
	if createdKey.GetID() != fetchedKey.GetID() {
		t.Errorf(
			"key ID is not correct, expected=%d, got=%d",
			createdKey.GetID(),
			fetchedKey.GetID(),
		)
	}
	if keyToCreate.IsActive != fetchedKey.IsActive {
		t.Errorf(
			"key IsActive is not correct, expected=%t, got=%t",
			keyToCreate.IsActive,
			fetchedKey.IsActive,
		)
	}
	if keyToCreate.Environment != fetchedKey.Environment {
		t.Errorf(
			"key Environment is not correct, expected=%s, got=%s",
			keyToCreate.Environment,
			fetchedKey.Environment,
		)
	}
	if keyToCreate.Source != fetchedKey.Source {
		t.Errorf(
			"key Source is not correct, expected=%s, got=%s",
			keyToCreate.Source,
			fetchedKey.Source,
		)
	}
	if keyToCreate.Description != fetchedKey.Description {
		t.Errorf(
			"key Description is not correct, expected=%s, got=%s",
			keyToCreate.Description,
			fetchedKey.Description,
		)
	}
}

func TestAuthorizeKey(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	// Create a key
	keyToCreate := Key{
		IsActive:    true,
		Environment: "env",
		Source:      "src",
		Description: "src",
	}
	_, token, err := reg.Create(keyToCreate)
	if err != nil {
		t.Fatalf("could not create a key: %v", err)
	}

	// Authorize the key
	authInfo, err := reg.Authorize(token)
	if err != nil {
		t.Fatalf("could not authorize the created key: %v", err)
	}
	if !authInfo.IsAuthorized {
		t.Errorf("newly created key is not authorized")
	}
	if authInfo.Environment != keyToCreate.Environment {
		t.Errorf(
			"authInfo environment does not match. expected:%s, got:%s",
			keyToCreate.Environment,
			authInfo.Environment,
		)
	}
	if authInfo.Source != keyToCreate.Source {
		t.Errorf(
			"authInfo source does not match. expected:%s, got:%s",
			keyToCreate.Source,
			authInfo.Source,
		)
	}

	authInfo, err = reg.Authorize("random-nonexisting-key")
	if err != nil {
		t.Fatalf("could not authorize a nonexisting key: %v", err)
	}
	if authInfo.IsAuthorized {
		t.Errorf("nonexisting key correctly authorized")
	}
}

func TestUpdateKey(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	// Create a key
	keyToCreate := Key{
		IsActive:    true,
		Environment: "env",
		Source:      "src",
		Description: "src",
	}
	createdKey, token, err := reg.Create(keyToCreate)
	if err != nil {
		t.Fatalf("could not create a key: %v", err)
	}

	// Ensure key without ID cannot be updated!
	if _, err = reg.Update(keyToCreate); err == nil {
		t.Fatalf("update for key without id is allowed!")
	}

	// Update the created key. Environment and Source must stay non-empty, so
	// Description is used to exercise zero-value persistence.
	createdKey.Environment = "changed-env"
	createdKey.Source = "changed-source"
	createdKey.Description = ""
	createdKey, err = reg.Update(createdKey)
	if err != nil {
		t.Fatalf("could not update the key: %v", err)
	}

	// Authorize the key
	authInfo, err := reg.Authorize(token)
	if err != nil {
		t.Fatalf("could not authorize the created key: %v", err)
	}
	if !authInfo.IsAuthorized {
		t.Errorf("newly created key is not authorized")
	}
	if authInfo.Environment != createdKey.Environment {
		t.Errorf(
			"authInfo environment does not match. expected:%s, got:%s",
			createdKey.Environment,
			authInfo.Environment,
		)
	}
	if authInfo.Source != createdKey.Source {
		t.Errorf(
			"authInfo source does not match. expected:%s, got:%s",
			createdKey.Source,
			authInfo.Source,
		)
	}

	// An empty Description must be persisted, not skipped as a zero value.
	fetchedKey, ok, err := reg.Get(createdKey.GetID())
	if err != nil {
		t.Fatalf("could not get the updated key: %v", err)
	}
	if !ok {
		t.Fatal("could not get the updated key: not found")
	}
	if fetchedKey.Description != "" {
		t.Errorf(
			"key Description was not cleared, got=%q",
			fetchedKey.Description,
		)
	}
}

func TestCreateRejectsInvalidKey(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	key := invalidateKey(Key{IsActive: true, Source: "src"})
	if _, _, err := reg.Create(key); err == nil || !strings.Contains(err.Error(), "environment") {
		t.Fatalf("Create() error = %v, want a validation error about the environment field", err)
	}

	// A rejected key must not be persisted.
	keys, err := reg.List(10, 0)
	if err != nil {
		t.Fatalf("could not list keys: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("there are %d keys after a rejected create", len(keys))
	}
}

func TestUpdateRejectsInvalidKey(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	createdKey := Key{
		IsActive:    true,
		Environment: "env",
		Source:      "src",
		Description: "original",
	}
	createdKey, _, err := reg.Create(createdKey)
	if err != nil {
		t.Fatalf("could not create a key: %v", err)
	}

	key := invalidateKey(createdKey)
	if _, err := reg.Update(key); err == nil || !strings.Contains(err.Error(), "environment") {
		t.Fatalf("Update() error = %v, want a validation error about the environment field", err)
	}

	// The rejected update must not change the stored key.
	fetched, ok, err := reg.Get(createdKey.GetID())
	if err != nil {
		t.Fatalf("could not get the created key: %v", err)
	}
	if !ok {
		t.Fatal("could not get the created key: not found")
	}
	if fetched.IsActive != createdKey.IsActive ||
		fetched.Environment != createdKey.Environment ||
		fetched.Source != createdKey.Source ||
		fetched.Description != createdKey.Description {
		t.Errorf(
			"stored key changed after a rejected update: got IsActive=%t Environment=%q Source=%q Description=%q",
			fetched.IsActive,
			fetched.Environment,
			fetched.Source,
			fetched.Description,
		)
	}
}

func TestNewRegistryRejectsBadPaths(t *testing.T) {
	notADatabase := filepath.Join(t.TempDir(), "not-a-database")
	if err := os.WriteFile(notADatabase, []byte("this is not a database"), 0o600); err != nil {
		t.Fatalf("could not create test file: %v", err)
	}

	paths := []struct {
		name string
		path string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"directory", t.TempDir()},
		{"missing parent", filepath.Join(t.TempDir(), "missing", "test.db")},
		{"not a database", notADatabase},
	}

	for _, tc := range paths {
		reg, err := NewRegistry(tc.path)
		if err == nil {
			_ = reg.Close()
			t.Errorf("%s: NewRegistry() succeeded, want an error", tc.name)
		}
	}
}

func TestAuthorizeRejectsBadTokens(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	tokens := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"nul byte", "authscope-\x00"},
		{"invalid utf8", string([]byte{0xff, 0xfe})},
		{"sql injection", "' OR 1=1 --"},
		{"oversized", strings.Repeat("a", maxTokenLen+1)},
	}

	for _, tc := range tokens {
		info, err := reg.Authorize(tc.token)
		if err != nil {
			t.Errorf("%s: Authorize() error = %v, want nil", tc.name, err)
		}
		if info.IsAuthorized {
			t.Errorf("%s: Authorize() authorized an invalid token", tc.name)
		}
	}
}

func TestGetInvalidIDs(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	ids := []struct {
		name string
		id   uint
	}{
		{"zero", 0},
		{"unknown", 42},
		{"max uint", math.MaxUint},
	}

	for _, tc := range ids {
		key, ok, err := reg.Get(tc.id)
		if err != nil {
			t.Errorf("%s: Get() error = %v, want nil", tc.name, err)
		}
		if ok {
			t.Errorf("%s: Get() returned ok=true, want false", tc.name)
		}
		if key.GetID() != 0 {
			t.Errorf("%s: Get() returned a key, want the zero Key", tc.name)
		}
	}
}

func TestCreateRejectsKeyWithID(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	createdKey, _, err := reg.Create(Key{
		IsActive:    true,
		Environment: "env",
		Source:      "src",
	})
	if err != nil {
		t.Fatalf("could not create a key: %v", err)
	}

	if _, _, err := reg.Create(createdKey); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("Create() error = %v, want ErrInvalidKey", err)
	}

	keys, err := reg.List(10, 0)
	if err != nil {
		t.Fatalf("could not list keys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("there are %d keys after a rejected create", len(keys))
	}
}

func TestUpdateMissingIDs(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	createdKey, _, err := reg.Create(Key{
		IsActive:    true,
		Environment: "env",
		Source:      "src",
	})
	if err != nil {
		t.Fatalf("could not create a key: %v", err)
	}
	if err := reg.Delete(createdKey.GetID()); err != nil {
		t.Fatalf("could not delete the key: %v", err)
	}

	keys := []struct {
		name string
		key  Key
	}{
		{"zero id", Key{IsActive: true, Environment: "env", Source: "src"}},
		{"unknown id", Key{IsActive: true, Environment: "env", Source: "src", id: 42}},
		{"max uint id", Key{IsActive: true, Environment: "env", Source: "src", id: math.MaxUint}},
		{"deleted id", createdKey},
	}

	for _, tc := range keys {
		if _, err := reg.Update(tc.key); !errors.Is(err, ErrKeyNotFound) {
			t.Errorf("%s: Update() error = %v, want ErrKeyNotFound", tc.name, err)
		}
	}
}

func TestDeleteMissingIDs(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	ids := []struct {
		name string
		id   uint
	}{
		{"zero", 0},
		{"unknown", 42},
		{"max uint", math.MaxUint},
	}

	for _, tc := range ids {
		if err := reg.Delete(tc.id); !errors.Is(err, ErrKeyNotFound) {
			t.Errorf("%s: Delete() error = %v, want ErrKeyNotFound", tc.name, err)
		}
	}

	first, _, err := reg.Create(Key{IsActive: true, Environment: "env", Source: "first"})
	if err != nil {
		t.Fatalf("could not create the first key: %v", err)
	}
	second, _, err := reg.Create(Key{IsActive: true, Environment: "env", Source: "second"})
	if err != nil {
		t.Fatalf("could not create the second key: %v", err)
	}

	if err := reg.Delete(first.GetID()); err != nil {
		t.Fatalf("could not delete the first key: %v", err)
	}
	if err := reg.Delete(first.GetID()); !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("second Delete() error = %v, want ErrKeyNotFound", err)
	}
	if _, ok, err := reg.Get(second.GetID()); err != nil {
		t.Errorf("Get() error = %v, want nil", err)
	} else if !ok {
		t.Error("deleting one key removed another")
	}
}

func TestListRejectsInvalidPagination(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	invalid := []struct {
		name   string
		limit  int
		offset int
	}{
		{"zero limit", 0, 0},
		{"negative limit", -1, 0},
		{"limit too large", MaxListLimit + 1, 0},
		{"max int limit", math.MaxInt, 0},
		{"negative offset", 1, -1},
	}

	for _, tc := range invalid {
		if _, err := reg.List(tc.limit, tc.offset); !errors.Is(err, ErrInvalidPagination) {
			t.Errorf("%s: List() error = %v, want ErrInvalidPagination", tc.name, err)
		}
	}

	valid := []struct {
		name   string
		limit  int
		offset int
	}{
		{"minimum limit", 1, 0},
		{"maximum limit", MaxListLimit, 0},
		{"large offset", 1, 1 << 20},
	}

	for _, tc := range valid {
		keys, err := reg.List(tc.limit, tc.offset)
		if err != nil {
			t.Errorf("%s: List() error = %v, want nil", tc.name, err)
			continue
		}
		if keys == nil {
			t.Errorf("%s: List() returned a nil slice, want an empty slice", tc.name)
		}
		if len(keys) != 0 {
			t.Errorf("%s: List() returned %d keys, want 0", tc.name, len(keys))
		}
	}
}

func TestSearch(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	// Keys deliberately share and vary fields to exercise matching. Note the
	// sources "api_v2" and "apixv2": a literal search for "api_v2" must not
	// treat "_" as a single-character wildcard.
	seeds := []Key{
		{IsActive: true, Environment: "prod", Source: "api", Description: "billing service"},
		{IsActive: true, Environment: "prod", Source: "web", Description: "frontend"},
		{IsActive: true, Environment: "dev", Source: "api_v2", Description: "internal"},
		{IsActive: true, Environment: "dev", Source: "apixv2", Description: "internal"},
	}

	sourceToID := make(map[string]uint, len(seeds))
	for _, seed := range seeds {
		created, _, err := reg.Create(seed)
		if err != nil {
			t.Fatalf("could not create key %q: %v", seed.Source, err)
		}
		sourceToID[seed.Source] = created.GetID()
	}

	tests := []struct {
		name     string
		keywords []string
		want     []string // expected sources
	}{
		{"single keyword matches environment", []string{"prod"}, []string{"api", "web"}},
		{"single keyword matches source substring", []string{"api"}, []string{"api", "api_v2", "apixv2"}},
		{"single keyword matches description", []string{"billing"}, []string{"api"}},
		{"case insensitive", []string{"PROD"}, []string{"api", "web"}},
		{"underscore is literal", []string{"api_v2"}, []string{"api_v2"}},
		{"all keywords must match", []string{"prod", "web"}, []string{"web"}},
		{"keywords span different fields", []string{"prod", "billing"}, []string{"api"}},
		{"no match", []string{"missing"}, []string{}},
		{"blank keywords are ignored", []string{"  ", "prod", ""}, []string{"api", "web"}},
		{"no keywords returns all", nil, []string{"api", "web", "api_v2", "apixv2"}},
	}

	for _, tc := range tests {
		keys, total, err := reg.Search(tc.keywords, 0, MaxListLimit)
		if err != nil {
			t.Errorf("%s: Search() error = %v, want nil", tc.name, err)
			continue
		}
		if keys == nil {
			t.Errorf("%s: Search() returned a nil slice, want an empty slice", tc.name)
			continue
		}
		if total != len(tc.want) {
			t.Errorf("%s: Search() total = %d, want %d", tc.name, total, len(tc.want))
		}

		got := make(map[string]bool, len(keys))
		for _, key := range keys {
			got[key.Source] = true
		}
		if len(got) != len(tc.want) {
			t.Errorf("%s: Search() returned %d keys, want %d", tc.name, len(got), len(tc.want))
		}
		for _, want := range tc.want {
			if !got[want] {
				t.Errorf("%s: Search() did not return key %q", tc.name, want)
			}
		}
	}

	// Every returned key must be the stored key, not a zero value.
	keys, _, err := reg.Search([]string{"billing"}, 0, MaxListLimit)
	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}
	if len(keys) != 1 || keys[0].GetID() != sourceToID["api"] {
		t.Errorf("Search() returned wrong key for %q: got %d keys", "billing", len(keys))
	}
}

func TestSearchPagination(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	// All four keys share the "shared" source so a single keyword matches all.
	for i := 0; i < 4; i++ {
		if _, _, err := reg.Create(Key{
			IsActive:    true,
			Environment: "prod",
			Source:      "shared",
			Description: "key",
		}); err != nil {
			t.Fatalf("could not create key: %v", err)
		}
	}

	// A page is limited by limit/offset, but total always reports every match.
	tests := []struct {
		name      string
		limit     int
		offset    int
		wantKeys  int
		wantTotal int
	}{
		{"first page", 2, 0, 2, 4},
		{"second page", 2, 2, 2, 4},
		{"partial last page", 3, 3, 1, 4},
		{"offset past end", 2, 10, 0, 4},
	}

	for _, tc := range tests {
		keys, total, err := reg.Search([]string{"shared"}, tc.offset, tc.limit)
		if err != nil {
			t.Errorf("%s: Search() error = %v, want nil", tc.name, err)
			continue
		}
		if len(keys) != tc.wantKeys {
			t.Errorf("%s: Search() returned %d keys, want %d", tc.name, len(keys), tc.wantKeys)
		}
		if total != tc.wantTotal {
			t.Errorf("%s: Search() total = %d, want %d", tc.name, total, tc.wantTotal)
		}
	}

	// Pages must not overlap.
	first, _, err := reg.Search([]string{"shared"}, 0, 2)
	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}
	second, _, err := reg.Search([]string{"shared"}, 2, 2)
	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}
	seen := make(map[uint]bool, len(first))
	for _, key := range first {
		seen[key.GetID()] = true
	}
	for _, key := range second {
		if seen[key.GetID()] {
			t.Errorf("key %d appeared on both pages", key.GetID())
		}
	}
}

func TestSearchRejectsInvalidPagination(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	invalid := []struct {
		name   string
		limit  int
		offset int
	}{
		{"zero limit", 0, 0},
		{"negative limit", -1, 0},
		{"limit too large", MaxListLimit + 1, 0},
		{"negative offset", 1, -1},
	}

	for _, tc := range invalid {
		if _, _, err := reg.Search(nil, tc.offset, tc.limit); !errors.Is(err, ErrInvalidPagination) {
			t.Errorf("%s: Search() error = %v, want ErrInvalidPagination", tc.name, err)
		}
	}
}

func TestCreateInactiveKey(t *testing.T) {
	reg := newTestRegistry(t)
	defer reg.Close()

	key, token, err := reg.Create(Key{
		IsActive:    false,
		Environment: "env",
		Source:      "src",
	})
	if err != nil {
		t.Fatalf("could not create an inactive key: %v", err)
	}
	if key.GetID() == 0 {
		t.Error("created key has no ID")
	}

	info, err := reg.Authorize(token)
	if err != nil {
		t.Fatalf("could not authorize the created key: %v", err)
	}
	if info.IsAuthorized {
		t.Error("inactive key is authorized")
	}
}
