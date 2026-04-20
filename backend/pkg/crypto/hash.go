package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func SHA256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// GenerateAPIKey returns a key like "sk-" + 48 base62 chars and its SHA-256 hash.
func GenerateAPIKey() (key, hash, prefix string, err error) {
	const length = 48
	b := make([]byte, length)
	for i := range b {
		n, e := rand.Int(rand.Reader, big.NewInt(int64(len(base62Chars))))
		if e != nil {
			return "", "", "", fmt.Errorf("generate api key: %w", e)
		}
		b[i] = base62Chars[n.Int64()]
	}
	key = "sk-" + string(b)
	hash = SHA256Hex(key)
	prefix = key[:10] // "sk-" + 7 chars
	return
}

// GenerateOrderNo generates a payment order number.
func GenerateOrderNo(channel string, userID int64) string {
	rb := make([]byte, 4)
	_, _ = rand.Read(rb)
	return fmt.Sprintf("%s%d%X", channel, userID, rb)
}

// GenerateToken generates a secure random hex token.
func GenerateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
