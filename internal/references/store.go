package references

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Reference holds metadata about a saved character reference image.
type Reference struct {
	Name      string    `json:"name"`
	Prompt    string    `json:"prompt"`
	Seed      int64     `json:"seed"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Format    string    `json:"format"`
	ImagePath string    `json:"image_path"`
	CreatedAt time.Time `json:"created_at"`
}

// Store manages named character reference images on disk.
type Store struct {
	baseDir string
	logger  *slog.Logger
}

// NewStore creates a reference store rooted at baseDir/references.
func NewStore(baseDir string, logger *slog.Logger) *Store {
	return &Store{
		baseDir: filepath.Join(baseDir, "references"),
		logger:  logger,
	}
}

// Save writes a reference image and its metadata to disk.
func (s *Store) Save(name string, imgData []byte, ref Reference) error {
	dir := filepath.Join(s.baseDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating reference directory: %w", err)
	}

	imgFile := fmt.Sprintf("reference.%s", ref.Format)
	imgPath := filepath.Join(dir, imgFile)
	if err := os.WriteFile(imgPath, imgData, 0o644); err != nil {
		return fmt.Errorf("writing reference image: %w", err)
	}

	absPath, err := filepath.Abs(imgPath)
	if err != nil {
		absPath = imgPath
	}
	ref.ImagePath = absPath
	ref.Name = name

	metaPath := filepath.Join(dir, "metadata.json")
	metaData, err := json.MarshalIndent(ref, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	if err := os.WriteFile(metaPath, metaData, 0o644); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}

	s.logger.Info("reference saved", "name", name, "path", absPath)
	return nil
}

// Load reads a named reference image and its metadata from disk.
func (s *Store) Load(name string) (*Reference, []byte, error) {
	metaPath := filepath.Join(s.baseDir, name, "metadata.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reference %q not found: %w", name, err)
	}

	var ref Reference
	if err := json.Unmarshal(metaData, &ref); err != nil {
		return nil, nil, fmt.Errorf("parsing reference metadata: %w", err)
	}

	imgFile := fmt.Sprintf("reference.%s", ref.Format)
	imgPath := filepath.Join(s.baseDir, name, imgFile)
	imgData, err := os.ReadFile(imgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading reference image: %w", err)
	}

	return &ref, imgData, nil
}

// List returns all saved references sorted by name.
func (s *Store) List() ([]Reference, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading references directory: %w", err)
	}

	var refs []Reference
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(s.baseDir, entry.Name(), "metadata.json")
		metaData, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var ref Reference
		if err := json.Unmarshal(metaData, &ref); err != nil {
			continue
		}
		refs = append(refs, ref)
	}

	sort.Slice(refs, func(i, j int) bool {
		return refs[i].Name < refs[j].Name
	})

	return refs, nil
}
