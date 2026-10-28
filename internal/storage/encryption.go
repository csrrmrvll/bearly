package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

type EncryptedPayload struct {
	Nonce      []byte
	AuthTag    []byte
	Ciphertext []byte
}

func Encrypt(plaintext []byte, key [32]byte) (EncryptedPayload, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return EncryptedPayload{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return EncryptedPayload{}, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return EncryptedPayload{}, err
	}
	sealed := aead.Seal(nil, nonce, plaintext, nil)
	return EncryptedPayload{
		Nonce:      nonce,
		AuthTag:    sealed[len(sealed)-16:],
		Ciphertext: sealed[:len(sealed)-16],
	}, nil
}

func Decrypt(payload EncryptedPayload, key [32]byte) ([]byte, error) {
	if len(payload.Nonce) != 12 {
		return nil, errors.New("invalid nonce size")
	}
	if len(payload.AuthTag) != 16 {
		return nil, errors.New("invalid auth tag size")
	}
	if len(payload.Ciphertext) == 0 {
		return nil, errors.New("invalid ciphertext size")
	}
	authenticatedCiphertext := append(payload.Ciphertext, payload.AuthTag...)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, payload.Nonce, authenticatedCiphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: " + err.Error())
	}
	if len(plaintext) == 0 {
		return nil, errors.New("decryption failed: plaintext is empty")
	}
	return plaintext, nil
}
