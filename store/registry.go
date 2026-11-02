package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// ErrKeyNotFound is returned when a key does not exist or was already deleted.
	ErrKeyNotFound = errors.New("store: key not found")
	// ErrInvalidPagination is returned when List is called with invalid bounds.
	ErrInvalidPagination = errors.New("store: invalid pagination")
)

const (
	// MaxListLimit is the maximum number of keys List returns in one call.
	MaxListLimit = 1000
	// maxTokenLen bounds the work Authorize performs on untrusted input.
	maxTokenLen = 4096
)

type AuthInfo struct {
	KeyID uint
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

func NewRegistry(dbPath string) (*Registry, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, errors.New("store: database path must not be empty")
	}
	// Open the db connection pool
	// IgnoreRecordNotFoundError silences the "record not found" log from
	// First() when a key is absent — Get intentionally handles that case as
	// ok=false, so the log is just noise.
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				IgnoreRecordNotFoundError: true,
				LogLevel:                  logger.Warn,
			},
		),
	})
	if err != nil {
		return nil, err
	}
	// Perform migrations
	if err = db.AutoMigrate(&keyRow{}); err != nil {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}

	return &Registry{db: db}, nil
}

func (r *Registry) Close() (err error) {
	// Close the connection pool
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (r *Registry) Authorize(token string) (AuthInfo, error) {
	if token == "" || len(token) > maxTokenLen {
		return AuthInfo{}, nil
	}

	tokenHash := hashToken(token)

	// Find the key row in the database for that token hash.
	// TODO: Make this optimal by having an in-memory store.
	var kr keyRow
	if err := r.db.First(&kr, "token_hash = ?", tokenHash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No key found for that token
			return AuthInfo{IsAuthorized: false}, nil
		}
		// Anything else is a genuine DB failure.
		return AuthInfo{}, err
	}

	if !kr.IsActive {
		// Key is not active
		return AuthInfo{IsAuthorized: false}, nil
	}

	// Key found for that token
	return AuthInfo{
		KeyID:        kr.ID,
		IsAuthorized: true,
		Environment:  kr.Environment,
		Source:       kr.Source,
	}, nil
}

func (r *Registry) Get(keyID uint) (Key, bool, error) {
	if keyID == 0 {
		return Key{}, false, nil
	}

	// Lookup a key row by it's ID
	var kr keyRow
	if err := r.db.First(&kr, keyID).Error; err != nil {
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
	if keyID == 0 {
		return fmt.Errorf("%w: id 0", ErrKeyNotFound)
	}

	// Delete key from the database
	res := r.db.Delete(&keyRow{}, keyID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: id %d", ErrKeyNotFound, keyID)
	}
	return nil
}

func (r *Registry) Create(key Key) (Key, string, error) {
	// Reject a key that already carries a database ID.
	if key.GetID() != 0 {
		return Key{}, "", fmt.Errorf("%w: key already has an ID", ErrInvalidKey)
	}

	// Reject keys that cannot be persisted before generating or storing anything.
	if err := key.validate(); err != nil {
		return Key{}, "", err
	}

	// Generate a token
	token := "authscope-" + rand.Text()

	// Get the hash of a token
	tokenHash := hashToken(token)

	// Create key in db
	kr := keyRow{
		IsActive:    key.IsActive,
		Environment: key.Environment,
		Source:      key.Source,
		Description: key.Description,
		TokenHash:   tokenHash,
	}
	if err := r.db.Create(&kr).Error; err != nil {
		return Key{}, "", err
	}

	return kr.ToKey(), token, nil
}

func (r *Registry) Update(key Key) (Key, error) {
	// Ensure that this key is in DB already
	if key.GetID() == 0 {
		return Key{}, fmt.Errorf("%w: id 0", ErrKeyNotFound)
	}

	// Reject invalid fields before touching the database.
	if err := key.validate(); err != nil {
		return Key{}, err
	}

	// Lookup a key row by it's ID
	var kr keyRow
	if err := r.db.First(&kr, key.GetID()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Key{}, fmt.Errorf("%w: id %d", ErrKeyNotFound, key.GetID())
		}
		return Key{}, err
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
		return Key{}, res.Error
	}
	if res.RowsAffected == 0 {
		return Key{}, fmt.Errorf("%w: id %d", ErrKeyNotFound, key.GetID())
	}
	return kr.ToKey(), nil
}

func (r *Registry) List(limit, offset int) ([]Key, error) {
	if limit < 1 || limit > MaxListLimit {
		return nil, fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalidPagination, MaxListLimit)
	}
	if offset < 0 {
		return nil, fmt.Errorf("%w: offset must not be negative", ErrInvalidPagination)
	}

	var rows []keyRow
	if err := r.db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	keys := make([]Key, len(rows))
	for i, kr := range rows {
		keys[i] = kr.ToKey()
	}
	return keys, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:]) // 64 lowercase hex chars
}
