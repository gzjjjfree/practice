package storageRelated

import "fyne.io/fyne/v2"

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
