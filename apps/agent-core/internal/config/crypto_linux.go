//go:build linux

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
	secretAttrService = "com.envnexus.agent-core"
	secretAttrName    = "config-encryption-key"
	aesKeyLen         = 32
)

func getOrCreateOSKey() ([]byte, error) {
	key, err := secretToolGet()
	if err == nil && len(key) == aesKeyLen {
		return key, nil
	}

	newKey := make([]byte, aesKeyLen)
	if _, err := rand.Read(newKey); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	if err := secretToolSet(newKey); err != nil {
		return nil, fmt.Errorf("store key in Secret Service: %w", err)
	}
	return newKey, nil
}

func secretToolGet() ([]byte, error) {
	cmd := exec.Command("secret-tool", "lookup",
		"service", secretAttrService,
		"name", secretAttrName,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(strings.TrimSpace(string(out)))
}

func secretToolSet(key []byte) error {
	hexKey := hex.EncodeToString(key)

	cmd := exec.Command("secret-tool", "store",
		"--label", "EnvNexus Agent Config Key",
		"service", secretAttrService,
		"name", secretAttrName,
	)
	cmd.Stdin = strings.NewReader(hexKey)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}

func encryptDPAPI(_ []byte) ([]byte, error) {
	return nil, errors.New("DPAPI not available on Linux")
}

func decryptDPAPI(_ []byte, _ byte) ([]byte, error) {
	return nil, errors.New("DPAPI not available on Linux")
}
