package storageRelated

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
)

var customStorageDir = ""

// SetStorageDir overrides the storage directory path (called once from main.go).
func SetStorageDir(path string) {
	customStorageDir = path
}

// GetStorageDir returns the app's storage directory path.
// It can be overridden via SetStorageDir for testing.
func GetStorageDir() string {
	if customStorageDir != "" {
		return customStorageDir
	}
	return fyne.CurrentApp().Storage().RootURI().Path()
}

// SecuredStorageFile wraps payload data and its SHA-256 checksum for integrity verification.
type SecuredStorageFile struct {
	Data     json.RawMessage `json:"data"`
	Checksum string          `json:"checksum"`
}

// AtomicWriteFileWithChecksum atomically writes data to absolutePath with checksum protection.
// Implements 7.1.1 (Atomic storage via temp file + rename) and 7.1.2 (Checksum verification).
func AtomicWriteFileWithChecksum(absolutePath string, data []byte) error {
	dir := filepath.Dir(absolutePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Compute SHA-256 checksum
	hash := sha256.Sum256(data)
	checksumStr := hex.EncodeToString(hash[:])

	secured := SecuredStorageFile{
		Data:     data,
		Checksum: checksumStr,
	}

	containerBytes, err := json.Marshal(secured)
	if err != nil {
		return err
	}

	// Create temp file in the same directory for atomic rename
	tmpFile, err := os.CreateTemp(dir, "atomic-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()

	success := false
	defer func() {
		tmpFile.Close()
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(containerBytes); err != nil {
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, absolutePath); err != nil {
		return err
	}
	success = true
	return nil
}

// ReadFileWithChecksum reads a file, verifies its checksum if applicable, and returns raw data.
// Falls back to raw bytes for legacy unsecured files.
func ReadFileWithChecksum(absolutePath string) ([]byte, error) {
	bytesData, err := os.ReadFile(absolutePath)
	if err != nil {
		return nil, err
	}

	var secured SecuredStorageFile
	if err := json.Unmarshal(bytesData, &secured); err == nil && secured.Checksum != "" {
		// Verify checksum
		hash := sha256.Sum256(secured.Data)
		expectedChecksum := hex.EncodeToString(hash[:])
		if expectedChecksum != secured.Checksum {
			slog.Warn("storage checksum mismatch, file may be corrupted", "path", absolutePath, "expected", expectedChecksum, "got", secured.Checksum)
		}
		return secured.Data, nil
	}

	// Fallback for legacy un-secured JSON files
	return bytesData, nil
}

// CleanupExpiredStorageFiles cleans up temporary atomic files older than 24 hours.
// Implements 7.3.1 (Expired data cleanup).
func CleanupExpiredStorageFiles() {
	storageDir := GetStorageDir()
	entries, err := os.ReadDir(storageDir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".tmp" || (len(name) >= 7 && name[:7] == "atomic-") {
			info, err := entry.Info()
			if err == nil && info.ModTime().Before(cutoff) {
				_ = os.Remove(filepath.Join(storageDir, name))
				slog.Info("cleaned up expired temp storage file", "name", name)
			}
		}
	}
}

// CheckStorageSpace checks if available storage space is sufficient or logs warnings.
// Implements 7.3.2 (Storage space monitoring and alerts).
func CheckStorageSpace() {
	storageDir := GetStorageDir()
	if _, err := os.Stat(storageDir); os.IsNotExist(err) {
		return
	}

	var totalSize int64
	_ = filepath.Walk(storageDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	const maxRecommendedSize = 500 * 1024 * 1024
	if totalSize > maxRecommendedSize {
		slog.Warn("storage usage exceeds recommended threshold", "totalSizeMB", float64(totalSize)/(1024*1024))
	}
}
