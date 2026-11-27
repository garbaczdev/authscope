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
	"sync"
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
	// maxTokenLen bounds the work FindKeyByToken performs on untrusted input.
	maxTokenLen = 4096
)

// The purpose of this object is to act as middleware between key store and clients.
// There are 2 types of clients:
// - Auth client that requires fast in-memory lookups to authorize keys via api
// - Admin client to modify the actual stored data
// Why this middleware is needed is because there is both a memory store and an
// in-memory structure.
type Registry struct {
	db *gorm.DB

	// An in-memory index holding all of the keys for quick lookups
	keyForHash map[string]Key
	// Corresponding mutex
	keyMu sync.RWMutex
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

	// Load in all keys to in-memory store
	var krs []keyRow
	if err := db.Find(&krs).Error; err != nil {
		return nil, err
	}
	keyForHash := make(map[string]Key)
	for _, kr := range krs {
		keyForHash[kr.TokenHash] = kr.ToKey()
	}

	return &Registry{
		db: db,
		keyForHash: keyForHash,
	}, nil
}

func (r *Registry) Close() (err error) {
	// Close the connection pool
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// FindKeyByToken returns a key that corresponds to the provided token.
// If a key is found for the token, that key will be returned along with ok=true.
// If no key was found for that token, it returns ok=false.
// If an unexpected error occurred during the check, it returns err!=nil.
func (r *Registry) FindKeyByToken(token string) (key Key, ok bool, err error) {
	if token == "" || len(token) > maxTokenLen {
		return Key{}, false, nil
	}

	// Get the token hash
	tokenHash := hashToken(token)

	// Acquire the read lock
	r.keyMu.RLock()
	defer r.keyMu.RUnlock()

	// Check in the in-memory store
	if key, ok := r.keyForHash[tokenHash]; ok {
		return key, true, nil
	} else {
		return Key{}, false, nil
	}
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
		return fmt.Errorf("%w: id %d", ErrKeyNotFound, keyID)
	}

	// Acquire the write lock
	r.keyMu.Lock()
	defer r.keyMu.Unlock()

	// Lookup a key row by it's ID
	var kr keyRow
	if err := r.db.First(&kr, keyID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: id %d", ErrKeyNotFound, keyID)
		}
		return err
	}

	// Delete key from the database
	res := r.db.Delete(&keyRow{}, keyID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: id %d", ErrKeyNotFound, keyID)
	}

	// Reflect in in-memory store
	delete(r.keyForHash, kr.TokenHash)

	return nil
}

func (r *Registry) Create(key Key) (Key, string, error) {
	// Reject a key that already carries a database ID.
	if key.GetID() != 0 {
		return Key{}, "", fmt.Errorf("%w: key already has an ID", ErrInvalidKey)
	}

	// Reject keys that cannot be persisted before generating or storing anything.
	if err := key.Validate(); err != nil {
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

	// Acquire the write lock
	r.keyMu.Lock()
	defer r.keyMu.Unlock()

	// Ensure no colission
	if _, ok := r.keyForHash[tokenHash]; ok {
		return Key{}, "", errors.New("token hash colission!")
	}
	
	// Persist to DB
	if err := r.db.Create(&kr).Error; err != nil {
		return Key{}, "", err
	}
	key = kr.ToKey()
	
	// Save in in-memory store
	r.keyForHash[tokenHash] = key

	return key, token, nil
}

func (r *Registry) Update(key Key) (Key, error) {
	// Ensure that this key is in DB already
	if key.GetID() == 0 {
		return Key{}, fmt.Errorf("%w: id 0", ErrKeyNotFound)
	}

	// Reject invalid fields before touching the database.
	if err := key.Validate(); err != nil {
		return Key{}, err
	}

	// Acquire the write lock
	r.keyMu.Lock()
	defer r.keyMu.Unlock()

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
	key = kr.ToKey()

	// Persist to in-memory store
	r.keyForHash[kr.TokenHash] = key

	return key, nil
}


func (r *Registry) List(offset int, limit int) ([]Key, int, error) {
	keys, count, err := r.Search([]string{}, offset, limit)
	return keys, count, err
}

// Count returns the number of keys in the database.
func (r *Registry) Count() (int, error) {
	var count int64
	if err := r.db.Model(&Key{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("could not count keys in db: %w", err)
	}
	return int(count), nil
}

// Search returns one page of keys that match all of the provided keywords,
// along with the total number of matching keys across all pages.
// Matching is case-insensitive and checks whether a keyword appears as a
// literal substring of the key's Environment, Source or Description. Keywords
// that are empty or only whitespace are ignored, so an empty keyword list
// returns every key. Results are newest-first, matching List.
func (r *Registry) Search(keywords []string, offset int, limit int) ([]Key, int, error) {
	if err := validatePagination(limit, offset); err != nil {
		return nil, 0, err
	}

	// Build the shared filter once; Session() clones it below so the Count and
	// Find statements do not clobber each other.
	base := r.db.Model(&keyRow{})
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}

		// instr compares literal substrings, so user input such as "%" or "_"
		// is matched as-is instead of acting as a LIKE wildcard.
		// A keyword matches when any of the searchable fields contains it.
		// Passing the nested query to Where makes GORM wrap the ORs in
		// parentheses, so the AND across keywords keeps correct precedence.
		base = base.Where(
			r.db.Where("instr(lower(environment), lower(?)) > 0", keyword).
				Or("instr(lower(source), lower(?)) > 0", keyword).
				Or("instr(lower(description), lower(?)) > 0", keyword),
		)
	}

	// Total matches across all pages (no limit/offset).
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// One page of results.
	var rows []keyRow
	if err := base.Session(&gorm.Session{}).
		Order("created_at DESC, id DESC").
		Offset(offset).
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	keys := make([]Key, len(rows))
	for i, kr := range rows {
		keys[i] = kr.ToKey()
	}
	return keys, int(total), nil
}

// validatePagination checks the bounds shared by List and Search.
func validatePagination(limit, offset int) error {
	if limit < 1 || limit > MaxListLimit {
		return fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalidPagination, MaxListLimit)
	}
	if offset < 0 {
		return fmt.Errorf("%w: offset must not be negative", ErrInvalidPagination)
	}
	return nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:]) // 64 lowercase hex chars
}
