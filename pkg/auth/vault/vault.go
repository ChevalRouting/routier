package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const KeySize = 32

var ErrSealedTooShort = errors.New("sealed value is too short")

type Vault struct {
	aead cipher.AEAD
}

func New(key []byte) (*Vault, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("vault key must be %d bytes, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Vault{aead: aead}, nil
}

func LoadOrCreate(path string) (*Vault, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		key, decodeErr := hex.DecodeString(strings.TrimSpace(string(data)))
		if decodeErr != nil {
			return nil, fmt.Errorf("decode vault key %s: %w", path, decodeErr)
		}

		return New(key)
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate vault key: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create vault key dir: %w", err)
	}

	if err := os.WriteFile(path, []byte(hex.EncodeToString(key)), 0o600); err != nil {
		return nil, fmt.Errorf("write vault key: %w", err)
	}

	return New(key)
}

func (v *Vault) Seal(plaintext, aad []byte) ([]byte, error) {
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	return v.aead.Seal(nonce, nonce, plaintext, aad), nil
}

func (v *Vault) Open(sealed, aad []byte) ([]byte, error) {
	size := v.aead.NonceSize()
	if len(sealed) < size+v.aead.Overhead() {
		return nil, ErrSealedTooShort
	}

	return v.aead.Open(nil, sealed[:size], sealed[size:], aad)
}
