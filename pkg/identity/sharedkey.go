package identity

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

const sharedInfo = "routier-friend-poll-v1"

type sharedEnvelope struct {
	V     int    `json:"v"`
	Nonce string `json:"nonce"`
	TS    int64  `json:"ts"`
	CT    string `json:"ct"`
}

func (i *Identity) SharedKey(peerX25519PubB64 string) ([32]byte, error) {
	var key [32]byte

	peer, err := base64.StdEncoding.DecodeString(peerX25519PubB64)
	if err != nil || len(peer) != 32 {
		return key, ErrSealFormat
	}

	priv := i.x25519Private()
	secret, err := curve25519.X25519(priv[:], peer)
	if err != nil {
		return key, err
	}

	self, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return key, err
	}

	r := hkdf.New(sha256.New, secret, sharedSalt(self, peer), []byte(sharedInfo))
	if _, err := io.ReadFull(r, key[:]); err != nil {
		return key, err
	}

	return key, nil
}

func sharedSalt(a, b []byte) []byte {
	if bytes.Compare(a, b) > 0 {
		a, b = b, a
	}

	return append(append([]byte{}, a...), b...)
}

func SealShared(key [32]byte, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ts := time.Now().Unix()
	cipher := aead.Seal(nil, nonce, plaintext, sharedAAD(ts))

	return json.Marshal(sharedEnvelope{
		V:     sealVersion,
		Nonce: base64.StdEncoding.EncodeToString(nonce),
		TS:    ts,
		CT:    base64.StdEncoding.EncodeToString(cipher),
	})
}

func OpenShared(key [32]byte, envelope []byte) ([]byte, error) {
	var env sharedEnvelope
	if err := json.Unmarshal(envelope, &env); err != nil || env.V != sealVersion {
		return nil, ErrSealFormat
	}

	if skew := time.Since(time.Unix(env.TS, 0)); skew > sealMaxSkew || skew < -sealMaxSkew {
		return nil, ErrSealStale
	}

	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil || len(nonce) != chacha20poly1305.NonceSizeX {
		return nil, ErrSealFormat
	}

	cipher, err := base64.StdEncoding.DecodeString(env.CT)
	if err != nil {
		return nil, ErrSealFormat
	}

	aead, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return nil, err
	}

	plain, err := aead.Open(nil, nonce, cipher, sharedAAD(env.TS))
	if err != nil {
		return nil, ErrSealDecrypt
	}

	return plain, nil
}

func sharedAAD(ts int64) []byte {
	return []byte(fmt.Sprintf("%d|%d", sealVersion, ts))
}
