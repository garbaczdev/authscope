package store

import (
	"path/filepath"
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
	createdKey := keyToCreate

	token, err := reg.Create(&createdKey)
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

	if _, err := reg.Get(createdKey.GetID()); err == nil {
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
	createdKey := keyToCreate

	_, err := reg.Create(&createdKey)
	if err != nil {
		t.Fatalf("could not create a key: %v", err)
	}

	// Get the created key
	fetchedKey, err := reg.Get(createdKey.GetID())
	if err != nil {
		t.Fatalf("could not get the created key: %v", err)
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
	createdKey := keyToCreate
	token, err := reg.Create(&createdKey)
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
	createdKey := keyToCreate
	token, err := reg.Create(&createdKey)
	if err != nil {
		t.Fatalf("could not create a key: %v", err)
	}

	// Ensure key without ID cannot be updated!
	if err = reg.Update(&keyToCreate); err == nil {
		t.Fatalf("update for key without id is allowed!")
	}

	// Update the created key
	createdKey.Environment = "changedEnv"
	createdKey.Source = "changedSource"
	if err = reg.Update(&createdKey); err != nil {
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
}
