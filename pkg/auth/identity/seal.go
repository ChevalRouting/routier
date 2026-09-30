package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

const (
	sealVersion   = 1
	sealMaxSkew   = 5 * time.Minute
	sealNonceSize = 24
)

var (
	ErrSealAuth    = errors.New("sealed envelope: signature/authentication failed")
	ErrSealStale   = errors.New("sealed envelope: timestamp outside allowed window")
	ErrSealDecrypt = errors.New("sealed envelope: decryption failed")
	ErrSealFormat  = errors.New("sealed envelope: malformed")
)

type sealedEnvelope struct {
	V         int    `json:"v"`
	SenderFpr string `json:"sfpr"`
	EphPub    string `json:"eph"`
	Nonce     string `json:"nonce"`
	TS        int64  `json:"ts"`
	Cipher    string `json:"ct"`
	Sig       string `json:"sig"`
}

func (i *Identity) x25519Private() [32]byte {
	h := sha512.Sum512(i.priv.Seed())

	var priv [32]byte
	copy(priv[:], h[:32])
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	return priv
}

func (i *Identity) X25519PublicBase64() string {
	priv := i.x25519Private()

	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(pub)
}

func signedBytes(env *sealedEnvelope) []byte {
	return []byte(fmt.Sprintf("%d|%s|%s|%s|%d|%s", env.V, env.SenderFpr, env.EphPub, env.Nonce, env.TS, env.Cipher))
}

func (i *Identity) Seal(recipientX25519PubB64 string, plaintext []byte) ([]byte, error) {
	recipient, err := base64.StdEncoding.DecodeString(recipientX25519PubB64)
	if err != nil || len(recipient) != 32 {
		return nil, ErrSealFormat
	}

	var recipientKey [32]byte
	copy(recipientKey[:], recipient)

	ephPub, ephPriv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	var nonce [sealNonceSize]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}

	cipher := box.Seal(nil, plaintext, &nonce, &recipientKey, ephPriv)

	env := sealedEnvelope{
		V:         sealVersion,
		SenderFpr: i.Fingerprint(),
		EphPub:    base64.StdEncoding.EncodeToString(ephPub[:]),
		Nonce:     base64.StdEncoding.EncodeToString(nonce[:]),
		TS:        time.Now().Unix(),
		Cipher:    base64.StdEncoding.EncodeToString(cipher),
	}
	env.Sig = base64.StdEncoding.EncodeToString(i.Sign(signedBytes(&env)))

	return json.Marshal(env)
}

func (i *Identity) Open(envelope []byte, senderEdPubB64 string) ([]byte, error) {
	var env sealedEnvelope
	if err := json.Unmarshal(envelope, &env); err != nil || env.V != sealVersion {
		return nil, ErrSealFormat
	}

	senderPub, err := ParsePublicKey(senderEdPubB64)
	if err != nil {
		return nil, ErrSealFormat
	}

	if Fingerprint(senderPub) != env.SenderFpr {
		return nil, ErrSealAuth
	}

	sig, err := base64.StdEncoding.DecodeString(env.Sig)
	if err != nil || !ed25519.Verify(senderPub, signedBytes(&env), sig) {
		return nil, ErrSealAuth
	}

	if skew := time.Since(time.Unix(env.TS, 0)); skew > sealMaxSkew || skew < -sealMaxSkew {
		return nil, ErrSealStale
	}

	ephRaw, err := base64.StdEncoding.DecodeString(env.EphPub)
	if err != nil || len(ephRaw) != 32 {
		return nil, ErrSealFormat
	}

	nonceRaw, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil || len(nonceRaw) != sealNonceSize {
		return nil, ErrSealFormat
	}

	cipher, err := base64.StdEncoding.DecodeString(env.Cipher)
	if err != nil {
		return nil, ErrSealFormat
	}

	var ephPub [32]byte
	var nonce [sealNonceSize]byte
	copy(ephPub[:], ephRaw)
	copy(nonce[:], nonceRaw)

	priv := i.x25519Private()

	plain, ok := box.Open(nil, cipher, &nonce, &ephPub, &priv)
	if !ok {
		return nil, ErrSealDecrypt
	}

	return plain, nil
}
