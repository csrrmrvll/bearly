package passwords

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	MaxLength  = 128
	saltLength = 16
)

func Hash(password string) (string, error) {
	if utf8.RuneCountInString(password) > MaxLength {
		return "", fmt.Errorf("password must not exceed %d characters", MaxLength)
	}

	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	derivedKey := argon2.IDKey([]byte(password), salt, 2, 19*1024, 1, 32)

	return encodeArgon2idHash(argon2idHash{
		version:     argon2.Version,
		memoryKiB:   19 * 1024,
		iterations:  2,
		parallelism: 1,
		salt:        salt,
		derivedKey:  derivedKey,
	}), nil
}

func Verify(password, encodedHash string) bool {
	if utf8.RuneCountInString(password) > MaxLength {
		return false
	}
	if legacyHash, ok := decodeLegacyHash(encodedHash); ok {
		candidateHash := sha256.Sum256([]byte(password))
		return subtle.ConstantTimeCompare(candidateHash[:], legacyHash) == 1
	}

	parsedHash, ok := parseArgon2idHash(encodedHash)
	if !ok || parsedHash.version != argon2.Version {
		return false
	}
	candidateHash := argon2.IDKey(
		[]byte(password),
		parsedHash.salt,
		parsedHash.iterations,
		parsedHash.memoryKiB,
		parsedHash.parallelism,
		uint32(len(parsedHash.derivedKey)),
	)
	return subtle.ConstantTimeCompare(candidateHash, parsedHash.derivedKey) == 1
}

func NeedsRehash(string) bool {
	return false
}
