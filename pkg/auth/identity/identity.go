package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

const KeyFilename = "identity_ed25519"

type Identity struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
}

func LoadOrCreate(path string) (*Identity, error) {
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		return parse(data)
	case !os.IsNotExist(err):
		return nil, fmt.Errorf("read identity %s: %w", path, err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate identity: %w", err)
	}

	enc := base64.StdEncoding.EncodeToString(priv)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create identity dir: %w", err)
	}

	if err := os.WriteFile(path, []byte(enc+"\n"), 0600); err != nil {
		return nil, fmt.Errorf("write identity %s: %w", path, err)
	}

	return &Identity{priv: priv, pub: pub}, nil
}

func parse(data []byte) (*Identity, error) {
	raw, err := base64.StdEncoding.DecodeString(string(trimSpace(data)))
	if err != nil {
		return nil, fmt.Errorf("decode identity: %w", err)
	}

	if len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("identity: bad key size %d", len(raw))
	}

	priv := ed25519.PrivateKey(raw)
	pub, ok := priv.Public().(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("identity: invalid private key")
	}

	return &Identity{priv: priv, pub: pub}, nil
}

func trimSpace(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ' || b[len(b)-1] == '\t') {
		b = b[:len(b)-1]
	}

	return b
}

func (i *Identity) PublicKey() ed25519.PublicKey {
	return i.pub
}

func (i *Identity) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(i.pub)
}

func (i *Identity) Fingerprint() string {
	return Fingerprint(i.pub)
}

func (i *Identity) Sign(msg []byte) []byte {
	return ed25519.Sign(i.priv, msg)
}

func Fingerprint(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:])
}

func ParsePublicKey(b64 string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}

	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("bad public key size %d", len(raw))
	}

	return ed25519.PublicKey(raw), nil
}

func Verify(pub ed25519.PublicKey, msg, sig []byte) bool {
	return ed25519.Verify(pub, msg, sig)
}

func FingerprintMatch(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
