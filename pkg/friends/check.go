package friends

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/identity"
	"github.com/ChevalRouting/routier/pkg/types"
)

func Check(ctx context.Context, f *config.Friend) *types.FriendStatus {
	return CheckWith(ctx, f, NewClient(f.URL, f.Token, f.TLSSkipVerify))
}

func CheckWith(ctx context.Context, f *config.Friend, c *Client) *types.FriendStatus {
	st := &types.FriendStatus{Name: f.Name}

	start := time.Now()
	hello, err := c.Hello(ctx)
	if err != nil {
		st.LastError = err.Error()
		return st
	}

	st.RTTms = time.Since(start).Milliseconds()
	st.LastSeen = time.Now().UTC().Format(time.RFC3339)
	st.Hostname = hello.Hostname
	st.Version = hello.Version
	st.OS = hello.OS
	st.Fingerprint = hello.IdentityFingerprint
	st.Encryption = hello.Encryption
	st.ConntrackdRunning = hello.ConntrackdRunning
	st.VRRP = hello.VRRP

	if f.Identity.Fingerprint != "" {
		st.IdentityMatch = identity.FingerprintMatch(f.Identity.Fingerprint, hello.IdentityFingerprint)
		if !st.IdentityMatch {
			st.LastError = "identity fingerprint mismatch (possible MITM)"
			return st
		}

		nonce := randomNonce()
		challenged, cerr := c.HelloChallenge(ctx, nonce)
		if cerr != nil {
			st.LastError = cerr.Error()
			return st
		}

		if !verifyChallenge(challenged, nonce) {
			st.LastError = "identity challenge failed (possible MITM or key theft)"
			return st
		}
	} else {
		st.IdentityMatch = true
	}

	st.Reachable = true
	return st
}

func verifyChallenge(hello *types.FriendsHello, nonce string) bool {
	pub, err := identity.ParsePublicKey(hello.IdentityPublicKey)
	if err != nil {
		return false
	}

	sig, err := base64.StdEncoding.DecodeString(hello.Signature)
	if err != nil {
		return false
	}

	return identity.Verify(pub, []byte(nonce), sig)
}

func randomNonce() string {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		return time.Now().UTC().Format(time.RFC3339Nano)
	}

	return hex.EncodeToString(b[:])
}
