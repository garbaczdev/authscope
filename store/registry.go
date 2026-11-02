package store

import (
	"errors"
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	TokenPrefixLen = 8
)

type AuthInfo struct {
	// true if the token is authorized
	IsAuthorized bool
	// Name of the key's environment
	Environment string
	// Name of the key's source
	Source string
}

// The purpose of this object is to act as middleware between key store and clients.
// There are 2 types of clients:
// - Auth client that requires fast in-memory lookups to authorize keys via api
// - Admin client to modify the actual stored data
// Why this middleware is needed is because there is both a memory store and an
// in-memory structure.
type Registry struct {
	db *gorm.DB
}

func NewRegistry(DBPath string) (*Registry, error) {
	// Open the db connection pool
	db, err := gorm.Open(sqlite.Open(DBPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	// Perform migrations
	err = db.AutoMigrate(&keyRow{})
	if err != nil {
		return nil, err
	}

	r := &Registry{db}

	return r, nil
}

func (r *Registry) Close() (err error) {
	// Close the connection pool
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (r *Registry) Authorize(token string) (authInfo AuthInfo, err error) {
	return AuthInfo{}, nil
}

func (r *Registry) Get(keyID uint) (key Key, ok bool, err error) {
	// Lookup a key row by it's ID
	var kr keyRow
	if err = r.db.First(&kr, keyID).Error; err != nil {
		// A missing key is not a real error — just ok=false like a map lookup.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Key{}, false, nil
		}
		// Anything else is a genuine DB failure.
		return Key{}, false, err
	}
	return kr.ToKey(), true, nil
}

func (r *Registry) Delete(keyID uint) error {
	var kr keyRow
	if err := r.db.Delete(&kr, keyID).Error; err != nil {
		return err
	}
	return nil
}

func (r *Registry) Create(key *Key) (token string, err error) {
	// Generate a token
	token = "aaaaaaaaaaaaaaaaaaaaaaaa"
	tokenPrefix := token[:TokenPrefixLen]

	// Get the hash of a token
	tokenSHA256 := ""

	// Create key in db
	kr := keyRow{
		IsActive:    key.IsActive,
		Environment: key.Environment,
		Source:      key.Source,
		Description: key.Description,
		TokenPrefix: tokenPrefix,
		TokenSHA256: tokenSHA256,
	}
	if err = r.db.Create(&kr).Error; err != nil {
		return "", nil
	}

	// Update the provided key
	*key = kr.ToKey()

	return token, nil
}

func (r *Registry) Update(key *Key) error {
	// Ensure that this key is in DB already
	if key.GetID() == 0 {
		return fmt.Errorf("can't update a key that is not yet in database")
	}

	// Lookup a key row by it's ID
	var kr keyRow
	if err := r.db.First(&kr, key.GetID()).Error; err != nil {
		return err
	}

	// Update the key row values
	kr.IsActive = key.IsActive
	kr.Environment = key.Environment
	kr.Source = key.Source
	kr.Description = key.Description

	// Save the key to the database.
	// Select forces every listed column to be written, so zero values such as
	// IsActive=false are persisted too (Updates with a struct would skip them).
	res := r.db.Model(&kr).
		Select("IsActive", "Environment", "Source", "Description").
		Updates(&kr)

	// Make sure the save actually went through.
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("key %d was not updated", key.GetID())
	}
	return nil
}

func (r *Registry) List(count int, offset int) (keys []Key, err error) {
	return nil, nil
}
