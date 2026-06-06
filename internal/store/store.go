// Package store persists the canonical CharSheet for a Hollow Grid world.
//
// CharStore is the seam between standalone and federated operation. The local
// FileStore is the documented offline fallback (federation never blocks play,
// docs/federation.md s8); the federation client will later implement the same
// interface against the Grid (loadCharacter/commitCharacter, protocol.md s3).
// Identity is name-based (protocol.md s1), so the key is the character name,
// normalized case-insensitively.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

// CharStore loads and commits the canonical CharSheet by character name.
type CharStore interface {
	// Load returns the sheet for name and whether one existed.
	Load(name string) (world.CharSheet, bool, error)
	// Commit persists the sheet for name.
	Commit(name string, sheet world.CharSheet) error
}

var keyUnsafe = regexp.MustCompile(`[^a-z0-9_-]+`)

// nameKey normalizes a character name to a safe, case-insensitive storage key.
func nameKey(name string) string {
	k := keyUnsafe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	return strings.Trim(k, "-")
}

// FileStore persists each character as a JSON file in a directory. Simple,
// dependency-free, and human-inspectable; swappable for SQLite or bolt behind
// the CharStore interface without touching the rest of the world.
type FileStore struct {
	dir string
	mu  sync.RWMutex
}

// NewFileStore opens (creating if needed) a character store at dir.
func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &FileStore{dir: dir}, nil
}

func (f *FileStore) path(name string) (string, bool) {
	key := nameKey(name)
	if key == "" {
		return "", false
	}
	return filepath.Join(f.dir, key+".json"), true
}

// Load reads the sheet for name; a missing character is (zero, false, nil).
func (f *FileStore) Load(name string) (world.CharSheet, bool, error) {
	path, ok := f.path(name)
	if !ok {
		return world.CharSheet{}, false, nil
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return world.CharSheet{}, false, nil
	}
	if err != nil {
		return world.CharSheet{}, false, err
	}
	var s world.CharSheet
	if err := json.Unmarshal(b, &s); err != nil {
		return world.CharSheet{}, false, err
	}
	return s, true, nil
}

// Commit writes the sheet for name atomically (temp file + rename).
func (f *FileStore) Commit(name string, sheet world.CharSheet) error {
	path, ok := f.path(name)
	if !ok {
		return errors.New("store: empty character name")
	}
	b, err := json.MarshalIndent(sheet, "", "  ")
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
