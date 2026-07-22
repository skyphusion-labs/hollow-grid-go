package auth

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const minPassphraseLen = 8
const maxPassphraseLen = 128

// HashPassphrase stores a bcrypt hash of the player's secret phrase.
func HashPassphrase(phrase string) (string, error) {
	phrase = strings.TrimSpace(phrase)
	if len(phrase) < minPassphraseLen {
		return "", errors.New("passphrase too short")
	}
	if len(phrase) > maxPassphraseLen {
		return "", errors.New("passphrase too long")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(phrase), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassphrase checks phrase against a stored bcrypt hash.
func VerifyPassphrase(phrase, hash string) bool {
	if hash == "" {
		return false
	}
	phrase = strings.TrimSpace(phrase)
	if len(phrase) > maxPassphraseLen {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(phrase)) == nil
}
