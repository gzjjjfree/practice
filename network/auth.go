package network

import (
	"encoding/json"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
)

const tokenFileName = "user_token.json"

// TokenStorage handles persistent storage of authentication tokens.
type TokenStorage struct {
	storageDir string
}

// TokenFileData is the structure saved to user_token.json.
type TokenFileData struct {
	Token   string `json:"token"`
	Refresh string `json:"refresh"`
}

// obfuscate simple XOR obfuscation to prevent plaintext reading
func obfuscate(data []byte) []byte {
	key := byte(0xA5)
	result := make([]byte, len(data))
	for i, b := range data {
		result[i] = b ^ key
	}
	return result
}

// NewTokenStorage creates a new token storage instance.
func NewTokenStorage() *TokenStorage {
	rootURI := fyne.CurrentApp().Storage().RootURI()
	path := rootURI.Path()
	if path == "" || path == "." {
		return &TokenStorage{
			storageDir: ".",
		}
	}
	return &TokenStorage{
		storageDir: path,
	}
}

// loadFileData loads the token file data from disk.
// Returns zero-value TokenFileData if file doesn't exist or is invalid.
func (s *TokenStorage) loadFileData() TokenFileData {
	filePath := filepath.Join(s.storageDir, tokenFileName)
	bytesData, err := os.ReadFile(filePath)
	if err != nil {
		return TokenFileData{}
	}
	// Try unmarshaling directly first (for backward compatibility), then obfuscated
	var data TokenFileData
	if err := json.Unmarshal(bytesData, &data); err == nil && (data.Token != "" || data.Refresh != "") {
		return data
	}

	plainData := obfuscate(bytesData)
	if err := json.Unmarshal(plainData, &data); err != nil {
		return TokenFileData{}
	}
	return data
}

// SaveToken saves the authentication token to local storage with 0600 permissions and obfuscation.
// It also saves the refresh token if provided (non-empty).
func (s *TokenStorage) SaveToken(token string) error {
	fileData := s.loadFileData()
	fileData.Token = token

	jsonData, err := json.Marshal(fileData)
	if err != nil {
		return err
	}

	obfuscated := obfuscate(jsonData)
	filePath := filepath.Join(s.storageDir, tokenFileName)
	return os.WriteFile(filePath, obfuscated, 0600)
}

// SaveRefreshToken saves the refresh token to local storage with 0600 permissions and obfuscation.
// It preserves the existing access token.
func (s *TokenStorage) SaveRefreshToken(refresh string) error {
	fileData := s.loadFileData()
	if refresh != "" {
		fileData.Refresh = refresh
	}

	jsonData, err := json.Marshal(fileData)
	if err != nil {
		return err
	}

	obfuscated := obfuscate(jsonData)
	filePath := filepath.Join(s.storageDir, tokenFileName)
	return os.WriteFile(filePath, obfuscated, 0600)
}

// LoadToken loads the authentication token from local storage.
func (s *TokenStorage) LoadToken() string {
	fileData := s.loadFileData()
	return fileData.Token
}

// LoadRefreshToken loads the refresh token from local storage.
func (s *TokenStorage) LoadRefreshToken() string {
	fileData := s.loadFileData()
	return fileData.Refresh
}

// ClearToken removes the authentication token from local storage.
// It tolerates "file does not exist" errors — clearing a missing file is idempotent.
func (s *TokenStorage) ClearToken() error {
	filePath := filepath.Join(s.storageDir, tokenFileName)
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ClearRefreshToken clears the refresh token from local storage.
func (s *TokenStorage) ClearRefreshToken() error {
	fileData := s.loadFileData()
	fileData.Refresh = ""
	if fileData.Token == "" {
		return s.ClearToken()
	}
	jsonData, err := json.Marshal(fileData)
	if err != nil {
		return err
	}
	obfuscated := obfuscate(jsonData)
	filePath := filepath.Join(s.storageDir, tokenFileName)
	return os.WriteFile(filePath, obfuscated, 0600)
}
