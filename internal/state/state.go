package state

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultStateDir = ".armory"
	StateFileName   = "state.json"
)

type SymlinkStatus string

const (
	StatusEquipped SymlinkStatus = "equipped"
	StatusStale    SymlinkStatus = "stale"
	StatusBroken   SymlinkStatus = "broken"
)

type EquippedEntry struct {
	Dir             string    `json:"dir"`
	Team            string    `json:"team"`
	Skills          []string  `json:"skills"`
	ManagedSymlinks []string  `json:"managed_symlinks"`
	EquippedAt      time.Time `json:"equipped_at"`
}

type State struct {
	Equipped []EquippedEntry `json:"equipped"`
}

// StatePath returns the default path to ~/.armory/state.json.
func StatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DefaultStateDir, StateFileName), nil
}

// LoadState reads and parses the default state.json.
// Returns an empty state if the file doesn't exist or contains invalid JSON.
func LoadState() (*State, error) {
	p, err := StatePath()
	if err != nil {
		return nil, err
	}
	return LoadStateFrom(p)
}

// LoadStateFrom reads and parses state.json from a specific path.
// Returns an empty state if the file doesn't exist or contains invalid JSON.
func LoadStateFrom(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		log.Printf("warning: corrupted state file %s: %v; returning empty state", path, err)
		return &State{}, nil
	}
	return &s, nil
}

// Save writes the state to the default ~/.armory/state.json path.
func (s *State) Save() error {
	p, err := StatePath()
	if err != nil {
		return err
	}
	return s.SaveTo(p)
}

// SaveTo writes the state to a specific path, creating directories as needed.
func (s *State) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// AddEquipped adds or updates an entry. If an entry with the same Dir exists, it is replaced.
func (s *State) AddEquipped(entry EquippedEntry) {
	for i, e := range s.Equipped {
		if e.Dir == entry.Dir {
			s.Equipped[i] = entry
			return
		}
	}
	s.Equipped = append(s.Equipped, entry)
}

// RemoveEquipped removes an entry by dir. Returns true if found and removed.
func (s *State) RemoveEquipped(dir string) bool {
	for i, e := range s.Equipped {
		if e.Dir == dir {
			s.Equipped = append(s.Equipped[:i], s.Equipped[i+1:]...)
			return true
		}
	}
	return false
}

// FindByDir returns the entry matching the given dir, or nil if not found.
func (s *State) FindByDir(dir string) *EquippedEntry {
	for i, e := range s.Equipped {
		if e.Dir == dir {
			return &s.Equipped[i]
		}
	}
	return nil
}

// VerifyEquipped checks the health of an equipped entry's symlinks.
// Returns StatusBroken if the directory doesn't exist, StatusStale if some
// symlinks are missing, and StatusEquipped if all symlinks are present.
func VerifyEquipped(entry EquippedEntry) SymlinkStatus {
	info, err := os.Lstat(entry.Dir)
	if err != nil || !info.IsDir() {
		return StatusBroken
	}

	if len(entry.ManagedSymlinks) == 0 {
		return StatusEquipped
	}

	skillsDir := filepath.Join(entry.Dir, ".claude", "skills")
	for _, name := range entry.ManagedSymlinks {
		link := filepath.Join(skillsDir, name)
		fi, err := os.Lstat(link)
		if err != nil || fi.Mode()&os.ModeSymlink == 0 {
			return StatusStale
		}
	}
	return StatusEquipped
}
