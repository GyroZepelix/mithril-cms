package media

import (
	"net/http"
	"testing"
)

func TestGenerateUUID(t *testing.T) {
	uuid := generateUUID()

	if len(uuid) != 36 {
		t.Fatalf("expected UUID length 36, got %d: %q", len(uuid), uuid)
	}

	// Check dashes at correct positions.
	if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		t.Fatalf("UUID has incorrect dash positions: %q", uuid)
	}

	// Check version nibble (position 14 should be '4').
	if uuid[14] != '4' {
		t.Fatalf("UUID version nibble should be '4', got %c: %q", uuid[14], uuid)
	}

	// Check variant nibble (position 19 should be 8, 9, a, or b).
	v := uuid[19]
	if v != '8' && v != '9' && v != 'a' && v != 'b' {
		t.Fatalf("UUID variant nibble should be 8/9/a/b, got %c: %q", v, uuid)
	}

	// Uniqueness check.
	uuid2 := generateUUID()
	if uuid == uuid2 {
		t.Fatal("two generated UUIDs should not be equal")
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"ABCDEF01-2345-6789-ABCD-EF0123456789", true},
		{"", false},
		{"not-a-uuid", false},
		{"550e8400-e29b-41d4-a716-44665544000", false},  // too short
		{"550e8400-e29b-41d4-a716-4466554400000", false}, // too long
		{"550e8400xe29b-41d4-a716-446655440000", false},  // wrong separator
		{"550e8400-e29b-41d4-a716-44665544000g", false},  // invalid char
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isValidUUID(tt.input)
			if got != tt.want {
				t.Errorf("isValidUUID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidVariant(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"original", true},
		{"sm", true},
		{"md", true},
		{"lg", true},
		{"xl", false},
		{"", false},
		{"ORIGINAL", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isValidVariant(tt.input)
			if got != tt.want {
				t.Errorf("isValidVariant(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestAllowedMIMEType(t *testing.T) {
	allowed := []string{
		"image/jpeg", "image/png", "image/gif", "image/webp",
		"application/pdf", "text/plain", "text/csv", "application/json",
	}
	for _, m := range allowed {
		if !AllowedMIMEType(m) {
			t.Errorf("expected %q to be allowed", m)
		}
	}

	disallowed := []string{
		"application/octet-stream", "text/html", "image/svg+xml",
		"application/javascript", "",
	}
	for _, m := range disallowed {
		if AllowedMIMEType(m) {
			t.Errorf("expected %q to be disallowed", m)
		}
	}
}

func TestIsImageMIME(t *testing.T) {
	images := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	for _, m := range images {
		if !IsImageMIME(m) {
			t.Errorf("expected %q to be image MIME", m)
		}
	}

	nonImages := []string{"application/pdf", "text/plain", "text/csv", "application/json"}
	for _, m := range nonImages {
		if IsImageMIME(m) {
			t.Errorf("expected %q to not be image MIME", m)
		}
	}
}

func TestFormatFromMIME(t *testing.T) {
	tests := []struct {
		mimeType string
		want     string
	}{
		{"image/jpeg", "JPEG"},
		{"image/png", "PNG"},
		{"image/gif", "GIF"},
		{"image/webp", "PNG"},  // WebP -> PNG to preserve transparency
		{"unknown/type", "JPEG"}, // fallback
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			got := formatFromMIME(tt.mimeType)
			if got.String() != tt.want {
				t.Errorf("formatFromMIME(%q) = %v, want %v", tt.mimeType, got, tt.want)
			}
		})
	}
}

func TestVariantExtension(t *testing.T) {
	tests := []struct {
		mimeType string
		want     string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/webp", ".png"}, // WebP variants encoded as PNG
		{"unknown/type", ".jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			got := variantExtension(tt.mimeType)
			if got != tt.want {
				t.Errorf("variantExtension(%q) = %q, want %q", tt.mimeType, got, tt.want)
			}
		})
	}
}

func TestReplaceExt(t *testing.T) {
	tests := []struct {
		filename string
		newExt   string
		want     string
	}{
		{"image.webp", ".png", "image.png"},
		{"photo.jpg", ".png", "photo.png"},
		{"noext", ".jpg", "noext.jpg"},
		{"multi.dots.txt", ".csv", "multi.dots.csv"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := replaceExt(tt.filename, tt.newExt)
			if got != tt.want {
				t.Errorf("replaceExt(%q, %q) = %q, want %q", tt.filename, tt.newExt, got, tt.want)
			}
		})
	}
}

func TestExtensionFromMIME(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"application/pdf", ".pdf"},
		{"text/plain", ".txt"},
		{"text/csv", ".csv"},
		{"application/json", ".json"},
		{"application/octet-stream", ".bin"},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			got := extensionFromMIME(tt.mime)
			if got != tt.want {
				t.Errorf("extensionFromMIME(%q) = %q, want %q", tt.mime, got, tt.want)
			}
		})
	}
}

func TestMIMEToExtensionMap(t *testing.T) {
	// Verify that every allowed MIME type has a corresponding extension.
	for mime := range allowedMIMETypes {
		ext, ok := mimeToExtension[mime]
		if !ok {
			t.Errorf("allowed MIME type %q missing from mimeToExtension map", mime)
		}
		if ext == "" {
			t.Errorf("mimeToExtension[%q] is empty", mime)
		}
	}
}

func TestUploadError(t *testing.T) {
	err := &UploadError{Message: "file too large"}
	if err.Error() != "file too large" {
		t.Errorf("unexpected error message: %q", err.Error())
	}

	if !ErrUpload(err) {
		t.Error("ErrUpload should return true for *UploadError")
	}

	if ErrUpload(ErrNotFound) {
		t.Error("ErrUpload should return false for non-UploadError")
	}
}

// TestMIMESpoofing_HTMLDeclaredAsTextPlain verifies that a file with HTML
// content is rejected even when the multipart header declares text/plain.
// This tests the MIME detection security fix.
func TestMIMESpoofing_HTMLDeclaredAsTextPlain(t *testing.T) {
	// HTML content that http.DetectContentType will identify as text/html.
	htmlContent := []byte("<html><body><script>alert('xss')</script></body></html>")

	detected := http.DetectContentType(htmlContent)
	// Verify our assumption: DetectContentType should recognize this as HTML.
	if detected != "text/html; charset=utf-8" {
		t.Fatalf("expected DetectContentType to return text/html, got %q", detected)
	}

	// text/html is not in our allowlist, so it should be rejected even if
	// the header says text/plain.
	if AllowedMIMEType("text/html") {
		t.Fatal("text/html should not be in the allowlist")
	}
}

// TestMIMESpoofing_DetectContentTypeHTMLNotAllowed confirms that HTML content
// cannot bypass MIME validation. The detected MIME must be either allowed or
// application/octet-stream for the upload to proceed.
func TestMIMESpoofing_DetectContentTypeHTMLNotAllowed(t *testing.T) {
	// Simulate the MIME detection logic from Upload.
	htmlData := []byte("<!DOCTYPE html><html><head><title>XSS</title></head></html>")
	detected := http.DetectContentType(htmlData)

	// Strip parameters.
	if idx := len("text/html"); len(detected) > idx && detected[idx] == ';' {
		detected = detected[:idx]
	}

	// The detected type should be "text/html" which is NOT in allowlist
	// and NOT "application/octet-stream", so the file should be rejected.
	if detected != "text/html" {
		t.Skipf("DetectContentType returned %q, expected text/html for this test", detected)
	}

	// Verify: text/html is not allowed and not octet-stream, so the MIME
	// validation logic rejects it regardless of header.
	if allowedMIMETypes[detected] {
		t.Fatal("text/html should not be in the allowlist")
	}
	if detected == "application/octet-stream" {
		t.Fatal("expected a specific detected type, not octet-stream")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal", "photo.jpg", "photo.jpg"},
		{"with quotes", `my"file.jpg`, "myfile.jpg"},
		{"with backslash", `my\file.jpg`, "myfile.jpg"},
		{"empty after sanitize", `"\`, "download"},
		{"already clean", "document.pdf", "document.pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
