package subsonic

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// generateSalt returns a random hex string of the given byte length.
func generateSalt(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// generateToken computes the Subsonic authentication token: md5(password + salt).
func generateToken(password, salt string) string {
	sum := md5.Sum([]byte(password + salt))
	return hex.EncodeToString(sum[:])
}
