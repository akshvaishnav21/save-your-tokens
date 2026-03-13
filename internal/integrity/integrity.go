package integrity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type integrityData struct {
	SHA256 string `json:"sha256"`
}

// HashScript computes the SHA-256 hash of the given script content.
func HashScript(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// Store saves the hash of hookScript to dataDir/hook-integrity.json.
func Store(dataDir, hookScript string) error {
	hash := HashScript(hookScript)
	data := integrityData{SHA256: hash}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling integrity data: %w", err)
	}

	intPath := filepath.Join(dataDir, "hook-integrity.json")
	if err := os.WriteFile(intPath, b, 0644); err != nil {
		return fmt.Errorf("writing integrity file: %w", err)
	}
	return nil
}

// Load reads the stored hash from dataDir/hook-integrity.json.
func Load(dataDir string) (string, error) {
	intPath := filepath.Join(dataDir, "hook-integrity.json")
	b, err := os.ReadFile(intPath)
	if err != nil {
		return "", fmt.Errorf("reading integrity file: %w", err)
	}

	var data integrityData
	if err := json.Unmarshal(b, &data); err != nil {
		return "", fmt.Errorf("parsing integrity file: %w", err)
	}
	return data.SHA256, nil
}

// Verify returns true if the current hook script matches the stored hash.
func Verify(dataDir, hookScript string) bool {
	stored, err := Load(dataDir)
	if err != nil {
		return false
	}
	current := HashScript(hookScript)
	return stored == current
}
