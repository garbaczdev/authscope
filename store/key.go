package store

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

// ErrInvalidKey is returned when a key has fields that cannot be stored.
var ErrInvalidKey = errors.New("store: invalid key")

const (
	// maxIdentifierLen is the maximum size of an Environment or Source value.
	maxIdentifierLen = 255
	// maxDescriptionLen is the maximum size of a Description value.
	maxDescriptionLen = 1024
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
	// Hashed key with a Hashing algorithm
	TokenHash string `gorm:"uniqueIndex;not null"`
}

func (kr keyRow) ToKey() Key {
	return Key{
		IsActive:    kr.IsActive,
		Environment: kr.Environment,
		Source:      kr.Source,
		Description: kr.Description,
		id:          kr.ID,
		createdAt:   kr.CreatedAt,
	}
}

// A view structure representing the database state of a key.
type Key struct {
	IsActive    bool
	Environment string
	Source      string
	Description string

	// Read-Only attributes: should not be changed from outside store package
	id        uint
	createdAt time.Time
}

// validate reports whether the key's mutable fields satisfy the store's
// persistence rules. Environment and Source must be non-empty and contain only
// visible ASCII characters (no spaces, control characters, or non-ASCII text).
// Description may contain any valid UTF-8 text.
func (k Key) validate() error {
	if err := validateIdentifier("environment", k.Environment); err != nil {
		return err
	}
	if err := validateIdentifier("source", k.Source); err != nil {
		return err
	}
	if len(k.Description) > maxDescriptionLen {
		return fmt.Errorf(
			"%w: description must be at most %d bytes",
			ErrInvalidKey,
			maxDescriptionLen,
		)
	}
	if !utf8.ValidString(k.Description) {
		return fmt.Errorf("%w: description must be valid UTF-8", ErrInvalidKey)
	}
	return nil
}

// validateIdentifier checks that value is a non-empty, bounded string of
// visible ASCII characters without spaces. These values become HTTP headers
// (X-Env and X-Source), so control characters must never reach the database.
func validateIdentifier(field, value string) error {
	if value == "" {
		return fmt.Errorf("%w: %s must not be empty", ErrInvalidKey, field)
	}
	if len(value) > maxIdentifierLen {
		return fmt.Errorf(
			"%w: %s must be at most %d bytes",
			ErrInvalidKey,
			field,
			maxIdentifierLen,
		)
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '!' || value[i] > '~' {
			return fmt.Errorf(
				"%w: %s must contain only visible ASCII characters without spaces",
				ErrInvalidKey,
				field,
			)
		}
	}
	return nil
}

func (k Key) GetID() uint {
	return k.id
}
func (k Key) GetCreatedAt() time.Time {
	return k.createdAt
}
