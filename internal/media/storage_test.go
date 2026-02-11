package media

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLocalStorage(t *testing.T) {
	dir := t.TempDir()

	storage, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	if storage == nil {
		t.Fatal("expected non-nil storage")
	}

	// Verify all variant directories were created.
	for _, v := range variants {
		path := filepath.Join(dir, v)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("variant directory %q not created: %v", v, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("variant %q is not a directory", v)
		}
	}
}

func TestLocalStorage_SaveAndPath(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	data := []byte("hello, world")
	filename := "test.txt"

	if err := storage.Save("original", filename, data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify the file exists at the expected path.
	expectedPath := filepath.Join(dir, "original", filename)
	gotPath := storage.Path("original", filename)
	if gotPath != expectedPath {
		t.Errorf("Path() = %q, want %q", gotPath, expectedPath)
	}

	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	if string(content) != "hello, world" {
		t.Errorf("file content = %q, want %q", string(content), "hello, world")
	}
}

func TestLocalStorage_SaveExclusive(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	filename := "unique.txt"
	if err := storage.Save("original", filename, []byte("first")); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	// Second save to the same filename should fail with ErrFileExists.
	err = storage.Save("original", filename, []byte("second"))
	if err == nil {
		t.Fatal("expected error on duplicate save, got nil")
	}
	if !errors.Is(err, ErrFileExists) {
		t.Errorf("expected ErrFileExists, got: %v", err)
	}

	// Original content should be preserved.
	content, err := os.ReadFile(storage.Path("original", filename))
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(content) != "first" {
		t.Errorf("file content = %q, want %q", string(content), "first")
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	filename := "deleteme.txt"
	if err := storage.Save("original", filename, []byte("data")); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Delete should succeed.
	if err := storage.Delete("original", filename); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// File should be gone.
	path := storage.Path("original", filename)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted, got err = %v", err)
	}

	// Deleting a non-existent file should be idempotent (no error).
	if err := storage.Delete("original", "nonexistent.txt"); err != nil {
		t.Errorf("Delete() of non-existent file should not error, got %v", err)
	}
}

func TestLocalStorage_SaveVariants(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	for _, v := range []string{"original", "sm", "md", "lg"} {
		data := []byte("data-" + v)
		if err := storage.Save(v, "test.jpg", data); err != nil {
			t.Errorf("Save(%q) error = %v", v, err)
		}

		content, err := os.ReadFile(storage.Path(v, "test.jpg"))
		if err != nil {
			t.Errorf("reading %q variant: %v", v, err)
			continue
		}
		if string(content) != "data-"+v {
			t.Errorf("%q variant content = %q, want %q", v, string(content), "data-"+v)
		}
	}
}

func TestIsSecureFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"valid simple", "image.jpg", true},
		{"valid uuid", "550e8400-e29b-41d4-a716-446655440000.jpg", true},
		{"valid with dash", "my-file.png", true},
		{"valid with underscore", "my_file.png", true},
		{"empty", "", false},
		{"dot prefix", ".hidden", false},
		{"dot prefix with ext", ".htaccess", false},
		{"dotdot", "..", false},
		{"path traversal unix", "../etc/passwd", false},
		{"path traversal windows", `..\..\windows\system32`, false},
		{"forward slash", "sub/file.jpg", false},
		{"backslash", `sub\file.jpg`, false},
		{"embedded dotdot", "foo..bar", false},
		{"absolute unix", "/etc/passwd", false},
		{"dot current dir", "./file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSecureFilename(tt.filename)
			if got != tt.want {
				t.Errorf("isSecureFilename(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestLocalStorage_PathTraversal_Save(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	maliciousFilenames := []string{
		"../../../etc/passwd",
		"..\\..\\windows\\system32\\config",
		".hidden",
		"sub/file.jpg",
		`sub\file.jpg`,
		"",
	}

	for _, name := range maliciousFilenames {
		err := storage.Save("original", name, []byte("evil"))
		if err == nil {
			t.Errorf("Save(%q) should have been rejected", name)
		}
		if !errors.Is(err, ErrInsecureFilename) {
			t.Errorf("Save(%q) error = %v, want ErrInsecureFilename", name, err)
		}
	}
}

func TestLocalStorage_PathTraversal_Delete(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	maliciousFilenames := []string{
		"../../../etc/passwd",
		".hidden",
		"",
	}

	for _, name := range maliciousFilenames {
		err := storage.Delete("original", name)
		if err == nil {
			t.Errorf("Delete(%q) should have been rejected", name)
		}
		if !errors.Is(err, ErrInsecureFilename) {
			t.Errorf("Delete(%q) error = %v, want ErrInsecureFilename", name, err)
		}
	}
}

func TestLocalStorage_InvalidVariant(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	// Save with invalid variant.
	err = storage.Save("xl", "file.jpg", []byte("data"))
	if err == nil {
		t.Error("Save with invalid variant should fail")
	}
	if !errors.Is(err, ErrInvalidVariant) {
		t.Errorf("expected ErrInvalidVariant, got: %v", err)
	}

	// Delete with invalid variant.
	err = storage.Delete("xl", "file.jpg")
	if err == nil {
		t.Error("Delete with invalid variant should fail")
	}
	if !errors.Is(err, ErrInvalidVariant) {
		t.Errorf("expected ErrInvalidVariant, got: %v", err)
	}

	// Path with invalid variant returns empty string.
	path := storage.Path("xl", "file.jpg")
	if path != "" {
		t.Errorf("Path with invalid variant should return empty, got %q", path)
	}

	// Path with insecure filename returns empty string.
	path = storage.Path("original", "../../../etc/passwd")
	if path != "" {
		t.Errorf("Path with traversal filename should return empty, got %q", path)
	}
}
