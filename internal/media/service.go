package media

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"image"
	// Register standard image decoders so image.Decode recognizes them.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/disintegration/imaging"

	"github.com/GyroZepelix/mithril-cms/internal/audit"
)

const (
	// maxUploadSize is the maximum allowed upload file size (10 MiB).
	maxUploadSize = 10 << 20
)

// allowedMIMETypes is the set of MIME types accepted for upload.
var allowedMIMETypes = map[string]bool{
	"image/jpeg":       true,
	"image/png":        true,
	"image/gif":        true,
	"image/webp":       true,
	"application/pdf":  true,
	"text/plain":       true,
	"text/csv":         true,
	"application/json": true,
}

// imageMIMETypes is the subset of allowed types that are images and support
// variant generation.
var imageMIMETypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// mimeToExtension maps validated MIME types to canonical file extensions.
// Extensions are derived from the MIME type, not user input, to prevent
// extension spoofing attacks.
var mimeToExtension = map[string]string{
	"image/jpeg":       ".jpg",
	"image/png":        ".png",
	"image/gif":        ".gif",
	"image/webp":       ".webp",
	"application/pdf":  ".pdf",
	"text/plain":       ".txt",
	"text/csv":         ".csv",
	"application/json": ".json",
}

// imageVariant defines a resizing target for image variants.
type imageVariant struct {
	Name     string
	MaxWidth int
}

var imageVariants = []imageVariant{
	{Name: "sm", MaxWidth: 480},
	{Name: "md", MaxWidth: 1024},
	{Name: "lg", MaxWidth: 1920},
}

// Service implements the business logic for media upload, processing, and deletion.
type Service struct {
	repo         *Repository
	storage      *LocalStorage
	auditService *audit.Service
}

// NewService creates a new media Service. The audit service is optional;
// if nil, audit events are silently skipped.
func NewService(repo *Repository, storage *LocalStorage, auditService *audit.Service) *Service {
	return &Service{
		repo:         repo,
		storage:      storage,
		auditService: auditService,
	}
}

// logAudit sends an audit event if the audit service is configured.
func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService != nil {
		s.auditService.Log(ctx, event)
	}
}

// UploadError represents a user-facing upload validation error.
type UploadError struct {
	Message string
}

func (e *UploadError) Error() string {
	return e.Message
}

// Upload processes a multipart file upload. It validates the file, saves the
// original and any image variants, and creates the database record.
func (s *Service) Upload(ctx context.Context, fh *multipart.FileHeader, adminID string) (*Media, error) {
	// Validate file size.
	if fh.Size > maxUploadSize {
		return nil, &UploadError{Message: fmt.Sprintf("file size %d exceeds maximum of %d bytes", fh.Size, maxUploadSize)}
	}

	// Open the uploaded file.
	file, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("opening uploaded file: %w", err)
	}
	defer file.Close()

	// Read the entire file into memory (bounded by maxUploadSize).
	data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading uploaded file: %w", err)
	}
	if int64(len(data)) > maxUploadSize {
		return nil, &UploadError{Message: fmt.Sprintf("file size exceeds maximum of %d bytes", maxUploadSize)}
	}

	// Detect MIME type from file content (first 512 bytes).
	detectedMIME := http.DetectContentType(data[:min(512, len(data))])

	// Normalize: strip parameters (e.g., "text/plain; charset=utf-8" -> "text/plain").
	if idx := strings.IndexByte(detectedMIME, ';'); idx != -1 {
		detectedMIME = strings.TrimSpace(detectedMIME[:idx])
	}

	// Also check the Content-Type from the multipart header.
	headerMIME := fh.Header.Get("Content-Type")
	if idx := strings.IndexByte(headerMIME, ';'); idx != -1 {
		headerMIME = strings.TrimSpace(headerMIME[:idx])
	}

	// Security: the detected MIME type must itself be allowed, OR be the
	// generic "application/octet-stream" fallback (which DetectContentType
	// returns for types it cannot identify). If the detected type is
	// something specific but not in our allowlist (e.g., "text/html"),
	// reject the file regardless of what the client header claims.
	mimeType := detectedMIME
	if detectedMIME == "application/octet-stream" {
		// DetectContentType could not identify the content. Trust the
		// client header only if it is in our allowlist.
		if allowedMIMETypes[headerMIME] {
			mimeType = headerMIME
		}
		// Otherwise mimeType stays "application/octet-stream" and will
		// be rejected by the allowlist check below.
	} else if allowedMIMETypes[detectedMIME] {
		// Detected type is in our allowlist. Prefer the header type when
		// it is also allowed and more specific (e.g., text/csv vs text/plain).
		if headerMIME != "" && allowedMIMETypes[headerMIME] {
			mimeType = headerMIME
		}
	}
	// If detected is something specific but not allowed (e.g., text/html),
	// mimeType == detectedMIME and will fail the check below.

	if !allowedMIMETypes[mimeType] {
		return nil, &UploadError{Message: fmt.Sprintf("MIME type '%s' is not allowed", mimeType)}
	}

	// Derive file extension from the validated MIME type, not user input.
	ext, ok := mimeToExtension[mimeType]
	if !ok {
		ext = ".bin"
	}
	uuidName := generateUUID() + ext

	// Save the original file.
	if err := s.storage.Save("original", uuidName, data); err != nil {
		return nil, fmt.Errorf("saving original file: %w", err)
	}

	// Build the media record.
	m := &Media{
		Filename:     uuidName,
		OriginalName: fh.Filename,
		MimeType:     mimeType,
		Size:         int64(len(data)),
		Variants:     make(map[string]string),
		UploadedBy:   &adminID,
	}

	// Process image variants if this is an image type.
	if imageMIMETypes[mimeType] {
		s.processImageVariants(m, uuidName, data, mimeType)
	}

	// Create the database record.
	if err := s.repo.Create(ctx, m); err != nil {
		// Clean up stored files on DB failure.
		s.cleanupFiles(uuidName, m.Variants)
		return nil, fmt.Errorf("creating media record: %w", err)
	}

	s.logAudit(ctx, audit.Event{
		Action:     "media.upload",
		ActorID:    adminID,
		Resource:   "media",
		ResourceID: m.ID,
	})

	return m, nil
}

// processImageVariants decodes the image, reads dimensions, and generates
// resized variants for each target width smaller than the original.
// It includes a recover guard to prevent panics from malformed images
// propagating to the HTTP handler.
func (s *Service) processImageVariants(m *Media, filename string, data []byte, mimeType string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic during image variant processing",
				"filename", filename, "panic", fmt.Sprintf("%v", r))
		}
	}()

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		slog.Warn("failed to decode image for variant generation", "filename", filename, "error", err)
		return
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	m.Width = &width
	m.Height = &height

	// Determine the encoding format for variants.
	// WebP cannot be encoded by the imaging library, so we use PNG to
	// preserve transparency. All other formats use their native encoding.
	variantFormat := formatFromMIME(mimeType)
	variantExt := variantExtension(mimeType)

	for _, v := range imageVariants {
		if width <= v.MaxWidth {
			// Original is not wider than this variant target; skip.
			continue
		}

		resized := imaging.Resize(img, v.MaxWidth, 0, imaging.Lanczos)
		var buf bytes.Buffer
		if err := imaging.Encode(&buf, resized, variantFormat); err != nil {
			slog.Warn("failed to encode image variant",
				"variant", v.Name, "filename", filename, "error", err)
			continue
		}

		// Build variant filename: replace original extension with variant extension.
		variantFilename := replaceExt(filename, variantExt)

		variantPath := v.Name + "/" + variantFilename
		if err := s.storage.Save(v.Name, variantFilename, buf.Bytes()); err != nil {
			slog.Warn("failed to save image variant",
				"variant", v.Name, "filename", filename, "error", err)
			continue
		}

		m.Variants[v.Name] = variantPath
	}
}

// cleanupFiles removes the original and any variant files from storage.
func (s *Service) cleanupFiles(filename string, variantPaths map[string]string) {
	if err := s.storage.Delete("original", filename); err != nil {
		slog.Warn("failed to clean up original file", "filename", filename, "error", err)
	}
	for variant, path := range variantPaths {
		// Extract just the filename portion from the variant path (variant/filename).
		parts := strings.SplitN(path, "/", 2)
		variantFilename := filename
		if len(parts) == 2 {
			variantFilename = parts[1]
		}
		if err := s.storage.Delete(variant, variantFilename); err != nil {
			slog.Warn("failed to clean up variant file", "variant", variant, "filename", variantFilename, "error", err)
		}
	}
}

// Delete removes a media record and all associated files from storage.
// The adminID is used for audit logging.
func (s *Service) Delete(ctx context.Context, id, adminID string) error {
	// Look up the record first so we know which files to delete.
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete from database first.
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Clean up files (best-effort, log failures).
	s.cleanupFiles(m.Filename, m.Variants)

	s.logAudit(ctx, audit.Event{
		Action:     "media.delete",
		ActorID:    adminID,
		Resource:   "media",
		ResourceID: id,
	})

	return nil
}

// List retrieves a paginated list of media records.
func (s *Service) List(ctx context.Context, page, perPage int) ([]*Media, int, error) {
	return s.repo.List(ctx, page, perPage)
}

// GetByFilename retrieves a media record by its generated filename.
func (s *Service) GetByFilename(ctx context.Context, filename string) (*Media, error) {
	return s.repo.GetByFilename(ctx, filename)
}

// generateUUID generates a UUID v4 string using crypto/rand.
func generateUUID() string {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	// Set version (4) and variant (RFC 4122).
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// formatFromMIME returns the imaging format to use for encoding variants.
// WebP is not natively encodable by the imaging library, so PNG is used
// instead to preserve transparency.
func formatFromMIME(mimeType string) imaging.Format {
	switch mimeType {
	case "image/jpeg":
		return imaging.JPEG
	case "image/png":
		return imaging.PNG
	case "image/gif":
		return imaging.GIF
	case "image/webp":
		// imaging cannot encode WebP; use PNG to preserve transparency.
		return imaging.PNG
	default:
		return imaging.JPEG
	}
}

// variantExtension returns the file extension for variant files based on MIME type.
// WebP originals get .png variants since we encode them as PNG.
func variantExtension(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".png" // Cannot encode WebP; variants are PNG.
	default:
		return ".jpg"
	}
}

// replaceExt replaces the file extension on filename with newExt.
func replaceExt(filename, newExt string) string {
	ext := strings.LastIndex(filename, ".")
	if ext == -1 {
		return filename + newExt
	}
	return filename[:ext] + newExt
}

// extensionFromMIME returns a file extension for a given MIME type.
func extensionFromMIME(mimeType string) string {
	if ext, ok := mimeToExtension[mimeType]; ok {
		return ext
	}
	return ".bin"
}

// isValidVariant checks if a variant name is one of the recognized variants.
func isValidVariant(v string) bool {
	return validVariants[v]
}

// isValidUUID validates that s looks like a UUID v4 (8-4-4-4-12 hex format).
func isValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// AllowedMIMEType reports whether the given MIME type is in the allowlist.
// Exported for use in tests.
func AllowedMIMEType(mimeType string) bool {
	return allowedMIMETypes[mimeType]
}

// IsImageMIME reports whether the given MIME type is an image type that
// supports variant generation.
func IsImageMIME(mimeType string) bool {
	return imageMIMETypes[mimeType]
}

// ErrUpload checks if an error is a user-facing upload validation error.
func ErrUpload(err error) bool {
	var ue *UploadError
	return errors.As(err, &ue)
}
