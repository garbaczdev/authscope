package store

import (
	"time"

	"gorm.io/gorm"
)

// Database model for a key
type Key struct {
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
	// Hashed key with SHA256 algorithm
	TokenSHA256 string
}
func (k Key) ToView() KeyView {
	return KeyView{
		IsActive: k.IsActive,
		Environment: k.Environment,
		Source: k.Source,
		Description: k.Description,
		id: k.ID,
		createdAt: k.CreatedAt,
		tokenPrefix: k.TokenPrefix,
	}
}

// A view structure representing the database state of a key.
type KeyView struct {
	IsActive    bool
	Environment string
	Source      string
	Description string

	// Read-Only attributes: should not be changed from outside store package
	id          uint
	createdAt   time.Time
	tokenPrefix string
}

func (k KeyView) GetID() uint {
	return k.id
}
func (k KeyView) GetCreatedAt() time.Time {
	return k.createdAt
}
func (k KeyView) GetTokenPrefix() string {
	return k.tokenPrefix
}
