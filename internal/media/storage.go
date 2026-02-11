package media

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// variants lists the subdirectories used for file storage.
var variants = []string{"original", "sm", "md", "lg"}

// validVariants is a set for O(1) lookup during validation.
var validVariants = map[string]bool{
	"original": true,
	"sm":       true,
	"md":       true,
	"lg":       true,
}

// ErrInsecureFilename is returned when a filename fails security validation.
var ErrInsecureFilename = errors.New("insecure filename")

// ErrInvalidVariant is returned when a variant name is not recognized.
var ErrInvalidVariant = errors.New("invalid variant")

// ErrFileExists is returned when attempting to save a file that already exists.
var ErrFileExists = errors.New("file already exists")

// isSecureFilename validates that a filename is safe for filesystem operations.
// It rejects empty strings, dot-prefixed names, path traversal sequences, and
// any path separator characters.
func isSecureFilename(filename string) bool {
	if filename == "" || strings.HasPrefix(filename, ".") {
		return false
	}
	if strings.Contains(filename, "..") || strings.ContainsAny(filename, "/\\") || filepath.IsAbs(filename) {
		return false
	}
	return true
}

// LocalStorage manages media files on the local filesystem, organized into
// variant subdirectories (original, sm, md, lg).
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a LocalStorage rooted at baseDir and ensures all
// variant subdirectories exist.
func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	for _, v := range variants {
		dir := filepath.Join(baseDir, v)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating media directory %s: %w", dir, err)
		}
	}
	return &LocalStorage{baseDir: baseDir}, nil
}

// Save writes data to {baseDir}/{variant}/{filename}. It validates the
// filename and variant for security, and uses O_EXCL to prevent overwriting
// existing files.
func (s *LocalStorage) Save(variant, filename string, data []byte) error {
	if !isSecureFilename(filename) {
		return fmt.Errorf("%w: %q", ErrInsecureFilename, filename)
	}
	if !validVariants[variant] {
		return fmt.Errorf("%w: %q", ErrInvalidVariant, variant)
	}

	path := filepath.Join(s.baseDir, variant, filename)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: %s", ErrFileExists, path)
		}
		return fmt.Errorf("creating file %s: %w", path, err)
	}

	_, writeErr := f.Write(data)
	closeErr := f.Close()

	if writeErr != nil {
		// Best-effort cleanup on write failure.
		_ = os.Remove(path) // Safe to ignore: we created this file.
		return fmt.Errorf("writing file %s: %w", path, writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing file %s: %w", path, closeErr)
	}

	return nil
}

// Delete removes the file at {baseDir}/{variant}/{filename}. It returns nil
// if the file does not exist (idempotent). The filename and variant are
// validated for security before any filesystem operation.
func (s *LocalStorage) Delete(variant, filename string) error {
	if !isSecureFilename(filename) {
		return fmt.Errorf("%w: %q", ErrInsecureFilename, filename)
	}
	if !validVariants[variant] {
		return fmt.Errorf("%w: %q", ErrInvalidVariant, variant)
	}

	path := filepath.Join(s.baseDir, variant, filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting file %s: %w", path, err)
	}
	return nil
}

// Path returns the absolute filesystem path for the given variant and filename.
// It returns an empty string if the filename or variant fails validation.
func (s *LocalStorage) Path(variant, filename string) string {
	if !isSecureFilename(filename) || !validVariants[variant] {
		return ""
	}
	return filepath.Join(s.baseDir, variant, filename)
}
