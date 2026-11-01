package store

import (
	"testing"
	"os"
)

func TestCreateDeleteKey(t *testing.T) {
	// Setup tmp db
	f, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("could not create temp file for db: %v", err)
	}
	defer os.Remove(f.Name())
	reg := Registry{DBPath: f.Name()}

	// Ensure there are no keys at the start
	keyViews, err := reg.List(1, 0)
	if err != nil {
		t.Fatalf("could not list keyViews: %v", err)
	}
	if len(keyViews) != 0 {
		t.Fatalf("there are %d keyViews in a newly initialized db", len(keyViews))
	}

	// Create a key
	keyToCreate := KeyView{
		IsActive: true,
		Environment: "env",
		Source: "src",
		Description: "src",
	}
	createdKey, _, err := reg.Create(keyToCreate)
	if err != nil {
		t.Fatalf("could not create a key: %v", err)
	}

	// Ensure that there is a key in the database
	keyViews, err = reg.List(1, 0)
	if err != nil {
		t.Fatalf("could not list keyViews: %v", err)
	}
	if len(keyViews) != 1 {
		t.Fatalf("there are %d keyViews after creating 1 key", len(keyViews))
	}

	// Delete the created key
	if err = reg.Delete(createdKey.GetID()); err != nil {
		t.Fatalf("could not delete the created key: %v", err)
	}
	// Ensure the deletion was correct
	keyViews, err = reg.List(1, 0)
	if err != nil {
		t.Fatalf("could not list keyViews: %v", err)
	}
	if len(keyViews) != 0 {
		t.Fatalf("there are %d keyViews after deleting created key", len(keyViews))
	}
}

func TestGetKey(t *testing.T) {
	// Setup tmp db
	f, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("could not create temp file for db: %v", err)
	}
	defer os.Remove(f.Name())
	reg := Registry{DBPath: f.Name()}

	// Create a key
	keyToCreate := KeyView{
		IsActive: true,
		Environment: "env",
		Source: "src",
		Description: "src",
	}
	createdKey, _, err := reg.Create(keyToCreate)
	if err != nil {
		t.Fatalf("could not create a key: %v", err)
	}

	// Get the created key
	fetchedKey, err := reg.Get(createdKey.GetID())
	if err != nil {
		t.Fatalf("could not get the created key: %v", err)
	}

	// Compare whether all fields are correct
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
	// Setup tmp db
	f, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("could not create temp file for db: %v", err)
	}
	defer os.Remove(f.Name())
	reg := Registry{DBPath: f.Name()}

	// Create a key
	keyToCreate := KeyView{
		IsActive: true,
		Environment: "env",
		Source: "src",
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
}

func TestUpdateKey(t *testing.T) {
	// Setup tmp db
	f, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("could not create temp file for db: %v", err)
	}
	defer os.Remove(f.Name())
	reg := Registry{DBPath: f.Name()}

	// Create a key
	keyToCreate := KeyView{
		IsActive: true,
		Environment: "env",
		Source: "src",
		Description: "src",
	}
	createdKey, token, err := reg.Create(keyToCreate)
	if err != nil {
		t.Fatalf("could not create a key: %v", err)
	}

	// Ensure key without ID cannot be updated!
	if err = reg.Update(keyToCreate); err == nil {
		t.Fatalf("update for key without id is allowed!")
	}

	// Update the created key
	createdKey.Environment = "changedEnv"
	createdKey.Source = "changedSource"
	if err = reg.Update(createdKey); err != nil {
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
