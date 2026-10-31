package passwords

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const MaxLength = 128
const MemoryCost = 19 * 1024 // 19 MiB

func Hash(password string) (string, error) {
	if utf8.RuneCountInString(password) > MaxLength {
		return "", fmt.Errorf("password must not exceed %d characters", MaxLength)
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	derivedKey := argon2.IDKey([]byte(password), salt, 2, MemoryCost, 1, 32)
	return encodeArgon2idHash(argon2idHash{
		version:     argon2.Version,
		salt:        salt,
		memoryKiB:   MemoryCost,
		iterations:  2,
		parallelism: 1,
		derivedKey:  derivedKey,
	}), nil
}

func Verify(password, encodedHash string) bool {
	if utf8.RuneCountInString(password) > MaxLength {
		return false
	}
	if len(encodedHash) == 64 {
		// Legacy SHA256 hash
		expectedHash, ok := decodeLegacyHash(encodedHash)
		if !ok {
			return false
		}
		sha256Hash := sha256.Sum256([]byte(password))
		return subtle.ConstantTimeCompare(sha256Hash[:], expectedHash) == 1
	}
	argon2idHash, ok := parseArgon2idHash(encodedHash)
	if !ok || argon2idHash.version != argon2.Version {
		return false
	}
	candidateHash := argon2.IDKey([]byte(password), argon2idHash.salt, argon2idHash.iterations, argon2idHash.memoryKiB, argon2idHash.parallelism, uint32(len(argon2idHash.derivedKey)))
	return subtle.ConstantTimeCompare(candidateHash, argon2idHash.derivedKey) == 1
}

func NeedsRehash(string) bool {
	return false
}
