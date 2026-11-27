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

func (k Key) Validate() error {
	return errors.Join(k.ValidateAll()...)
}

func (k Key) ValidateAll() []error {
	validationErrors := make([]error, 0)

	// Ensure no empty environment/source
	if len(k.Environment) == 0 {
		validationErrors = append(validationErrors, errors.New(
			"environment field cannot be empty",
		))
	}
	if len(k.Source) == 0 {
		validationErrors = append(validationErrors, errors.New(
			"source field cannot be empty",
		))
	}

	// Ensure no un-safe characters
	for idx := range k.Environment {
		if !isSafeCharacter(k.Environment[idx]) {
			validationErrors = append(validationErrors, errors.New(
				"environment field can contain only: letters/numbers/-/_/.",
			))
			break
		}
	}
	if len(k.Environment) > maxIdentifierLen {
		validationErrors = append(validationErrors, fmt.Errorf(
			"environment field cannot exceed %d characters", maxIdentifierLen,
		))
	}
	for idx := range k.Source {
		if !isSafeCharacter(k.Source[idx]) {
			validationErrors = append(validationErrors, errors.New(
				"source field can contain only: letters/numbers/-/_/.",
			))
			break
		}
	}
	if len(k.Source) > maxIdentifierLen {
		validationErrors = append(validationErrors, fmt.Errorf(
			"source field cannot exceed %d characters", maxIdentifierLen,
		))
	}

	// Ensure description is valid utf-8
	if !utf8.ValidString(k.Description) {
		validationErrors = append(validationErrors, errors.New(
			"description field has to be a valid utf-8 string",
		))
	}
	if len(k.Description) > maxDescriptionLen {
		validationErrors = append(validationErrors, fmt.Errorf(
			"description field cannot exceed %d characters", maxDescriptionLen,
		))
	}

	return validationErrors
}

func isSafeCharacter(char byte) bool {
	// number
	if 48 <= char && char <= 57 {
		return true
	}
	// lowercase letter
	if 97 <= char && char <= 122 {
		return true
	}
	// -
	if char == 45 {
		return true
	}
	// _
	if char == 95 {
		return true
	}
	// .
	if char == 46 {
		return true
	}
	// Anything else is invalid
	return false
}

func (k Key) GetID() uint {
	return k.id
}
func (k Key) GetCreatedAt() time.Time {
	return k.createdAt
}
