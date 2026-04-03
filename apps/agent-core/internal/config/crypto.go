package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
)

const (
	encMagic     = "ENX_ENC:"
	encVersion   = byte(2) // v2 = OS-level encryption
	encHeaderLen = len(encMagic) + 1
	gcmNonceLen  = 12
)

// Encrypt encrypts plaintext using the OS credential store.
// Windows: DPAPI (entire blob). macOS/Linux: AES-256-GCM with key from Keychain/Secret Service.
func Encrypt(plaintext []byte) ([]byte, error) {
	if runtime.GOOS == "windows" {
		return encryptDPAPI(plaintext)
	}
	return encryptAESWithOSKey(plaintext)
}

// Decrypt decrypts data produced by Encrypt.
func Decrypt(data []byte) ([]byte, error) {
	if !IsEncrypted(data) {
		return nil, errors.New("not an encrypted file")
	}
	ver := data[len(encMagic)]
	payload := data[encHeaderLen:]

	if runtime.GOOS == "windows" {
		return decryptDPAPI(payload, ver)
	}
	return decryptAESWithOSKey(payload, ver)
}

// IsEncrypted returns true if data starts with the encryption magic header.
func IsEncrypted(data []byte) bool {
	return len(data) > encHeaderLen && string(data[:len(encMagic)]) == encMagic
}

// ReadFileAutoDecrypt reads a file; decrypts if encrypted, otherwise returns plaintext.
// If the file has the encryption header but decryption fails (e.g. different user/machine),
// the error is returned so the caller can fall back to other config sources.
func ReadFileAutoDecrypt(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if IsEncrypted(data) {
		decrypted, err := Decrypt(data)
		if err != nil {
			slog.Warn("[config] Encrypted file could not be decrypted", "path", path, "error", err)
			return nil, fmt.Errorf("decrypt %s: %w", path, err)
		}
		return decrypted, nil
	}
	return data, nil
}

// WriteFileEncrypted encrypts and writes data. Falls back to plaintext on failure.
func WriteFileEncrypted(path string, plaintext []byte, perm os.FileMode) error {
	encrypted, err := Encrypt(plaintext)
	if err != nil {
		slog.Warn("[config] Encryption failed, writing plaintext", "path", path, "error", err)
		return os.WriteFile(path, plaintext, perm)
	}
	return os.WriteFile(path, encrypted, perm)
}

func makeHeader() []byte {
	h := make([]byte, encHeaderLen)
	copy(h, []byte(encMagic))
	h[len(encMagic)] = encVersion
	return h
}

// ── AES-256-GCM with OS-managed key (macOS / Linux) ─────────────────────────

func encryptAESWithOSKey(plaintext []byte) ([]byte, error) {
	key, err := getOrCreateOSKey()
	if err != nil {
		return nil, fmt.Errorf("get OS key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcmNonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	out := makeHeader()
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func decryptAESWithOSKey(payload []byte, _ byte) ([]byte, error) {
	if len(payload) < gcmNonceLen+1 {
		return nil, errors.New("payload too short")
	}
	key, err := getOrCreateOSKey()
	if err != nil {
		return nil, fmt.Errorf("get OS key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := payload[:gcmNonceLen]
	ciphertext := payload[gcmNonceLen:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
