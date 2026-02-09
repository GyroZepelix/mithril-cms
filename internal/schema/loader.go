package schema

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// LoadSchemas reads all *.yaml and *.yml files from the given directory, parses
// each into a ContentType, computes the SHA256 hash of the raw file bytes for
// change detection, and returns the schemas sorted by name for deterministic
// ordering.
//
// An empty directory returns an empty slice with no error.
// A missing directory returns an error.
func LoadSchemas(dir string) ([]ContentType, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading schema directory %q: %w", dir, err)
	}

	var schemas []ContentType

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		ct, err := loadSchemaFile(path)
		if err != nil {
			return nil, fmt.Errorf("loading schema file %q: %w", entry.Name(), err)
		}

		schemas = append(schemas, ct)
	}

	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Name < schemas[j].Name
	})

	return schemas, nil
}

// loadSchemaFile reads a single YAML file, parses it into a ContentType,
// and computes its SHA256 hash. The decoder uses KnownFields(true) so that
// unknown or misspelled keys (e.g., "requred" instead of "required") cause
// a parse error instead of being silently ignored.
func loadSchemaFile(path string) (ContentType, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ContentType{}, fmt.Errorf("reading file: %w", err)
	}

	var ct ContentType
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&ct); err != nil {
		return ContentType{}, fmt.Errorf("parsing YAML: %w", err)
	}

	ct.SchemaHash = fmt.Sprintf("%x", sha256.Sum256(data))

	return ct, nil
}
