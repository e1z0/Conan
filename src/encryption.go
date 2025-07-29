package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
)

const magic = "HMACENCv1:" // or anything you choose
var ErrNoMagic = errors.New("missing magic prefix; not encrypted by this scheme")

// Derive 32-byte key from string key using SHA-256
func deriveKey(passphrase string) []byte {
	hash := sha256.Sum256([]byte(passphrase))
	return hash[:]
}

func generatePassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"0123456789" +
		"!@#$%^&*()-_=+[]{}<>?/"

	password := make([]byte, length)
	for i := range password {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[n.Int64()]
	}
	return string(password), nil
}

func decryptString(encrypted, enck string) (string, error) {
	if encrypted == "" {
		return "", nil // Return empty string if plaintext is empty
	}
	key := deriveKey(settings.GlobEncryptKey)
	if enck != "" {
		key = deriveKey(enck)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// enck should be passed normally, only used when changing the encryption key
func encryptString(plaintext, enck string) (string, error) {
	if plaintext == "" {
		return "", nil // Return empty string if plaintext is empty
	}
	key := deriveKey(settings.GlobEncryptKey)
	if enck != "" {
		key = deriveKey(enck)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func encNewKey() (string, error) {
	key, err := generatePassword(32)
	if err != nil {
		return "", err
	}
	return key, nil
}

func isEncrypted(text string) bool {
	return strings.HasPrefix(text, magic)
}

// decryptWithMagic checks for the prefix, strips it, and then calls decryptAES.
func encryptWithMagic(input, passphrase string) (string, error) {
	encrypted, err := encryptAES(input, passphrase)
	return magic + encrypted, err
}

// encryptAES encrypts plaintext using AES-GCM with a key derived from passphrase
func encryptAES(plaintext, passphrase string) (string, error) {
	// derive 32-byte key from passphrase
	h := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(h[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptWithMagic checks for the prefix, strips it, and then calls decryptAES.
func decryptWithMagic(input, passphrase string) (string, error) {
	if !strings.HasPrefix(input, magic) {
		return "", ErrNoMagic
	}
	// strip off the prefix:
	b64 := strings.TrimPrefix(input, magic)
	return decryptAES(b64, passphrase)
}

// decryptAES decrypts base64-encoded ciphertext using AES-GCM and passphrase
func decryptAES(cipherB64, passphrase string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", err
	}
	// derive key
	h := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(h[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
