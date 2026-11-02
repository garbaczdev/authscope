package store

import (
	"time"

	"gorm.io/gorm"
)

// Database model for a key
type keyRow struct {
	gorm.Model
	// Whether this key is active
	IsActive bool
	// Name of the key's environment
	Environment string
	// Name of the key's source
	Source string
	// Informative description for the key
	Description string
	// Start of the token so it can be recognized later
	TokenPrefix string
	// Hashed key with a Hashing algorithm
	TokenHash string
}
func (kr keyRow) ToKey() Key {
	return Key{
		IsActive: kr.IsActive,
		Environment: kr.Environment,
		Source: kr.Source,
		Description: kr.Description,
		id: kr.ID,
		createdAt: kr.CreatedAt,
		tokenPrefix: kr.TokenPrefix,
	}
}

// A view structure representing the database state of a key.
type Key struct {
	IsActive    bool
	Environment string
	Source      string
	Description string

	// Read-Only attributes: should not be changed from outside store package
	id          uint
	createdAt   time.Time
	tokenPrefix string
}

func (k Key) GetID() uint {
	return k.id
}
func (k Key) GetCreatedAt() time.Time {
	return k.createdAt
}
func (k Key) GetTokenPrefix() string {
	return k.tokenPrefix
}
