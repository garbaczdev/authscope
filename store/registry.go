package store

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	TokenPrefixLen=8
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
	err = db.AutoMigrate(&Key{})
	if err != nil {
		return nil, err
	}

	r := &Registry{ db }

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

func (r *Registry) Get(keyID uint) (keyView KeyView, err error) {
	// Lookup a key by it's ID
	var key Key
	if err = r.db.First(&key, keyID).Error; err != nil {
		return keyView, err
	}
	return key.ToView(), nil
}

func (r *Registry) Delete(keyID uint) (err error) {
	return nil
}

func (r *Registry) Create(keyView KeyView) (
	createdKeyView KeyView,
	token string,
	err error,
) {
	// Generate a token
	token = "aaaaaaaaaaaaaaaaaaaaaaaa"
	tokenPrefix := token[:TokenPrefixLen]

	// Get the hash of a token
	tokenSHA256 := ""

	// Create key in db
	key := &Key{
		IsActive: keyView.IsActive,
		Environment: keyView.Environment,
		Source: keyView.Source,
		Description: keyView.Description,
		TokenPrefix: tokenPrefix,
		TokenSHA256: tokenSHA256,
	}
	if err = r.db.Create(key).Error; err != nil {
		return createdKeyView, "", nil
	}

	return key.ToView(), token, nil
}

func (r *Registry) Update(keyView KeyView) (err error) {
	return nil
}

func (r *Registry) List(count int, offset int) (keyViews []KeyView, err error) {
	return nil, nil
}
