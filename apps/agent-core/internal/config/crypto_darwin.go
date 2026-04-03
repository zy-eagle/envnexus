//go:build darwin

package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const (
	keychainService = "com.envnexus.agent-core"
	keychainAccount = "config-encryption-key"
	aesKeyLen       = 32
)

func getOrCreateOSKey() ([]byte, error) {
	key, err := keychainGet()
	if err == nil && len(key) == aesKeyLen {
		return key, nil
	}

	newKey := make([]byte, aesKeyLen)
	if _, err := rand.Read(newKey); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	if err := keychainSet(newKey); err != nil {
		return nil, fmt.Errorf("store key in Keychain: %w", err)
	}
	return newKey, nil
}

func keychainGet() ([]byte, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", keychainService,
		"-a", keychainAccount,
		"-w",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(strings.TrimSpace(string(out)))
}

func keychainSet(key []byte) error {
	hexKey := hex.EncodeToString(key)

	// Delete existing entry if present (ignore error).
	_ = exec.Command("security", "delete-generic-password",
		"-s", keychainService,
		"-a", keychainAccount,
	).Run()

	cmd := exec.Command("security", "add-generic-password",
		"-s", keychainService,
		"-a", keychainAccount,
		"-w", hexKey,
		"-U", // update if exists
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}

func encryptDPAPI(_ []byte) ([]byte, error) {
	return nil, errors.New("DPAPI not available on macOS")
}

func decryptDPAPI(_ []byte, _ byte) ([]byte, error) {
	return nil, errors.New("DPAPI not available on macOS")
}
