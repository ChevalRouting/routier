package friends

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"

	"github.com/ChevalRouting/routier/pkg/client"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/identity"
	"github.com/ChevalRouting/routier/pkg/types"
)

type Client struct {
	*client.Client
}

func NewClient(url, token string, tlsSkipVerify bool) *Client {
	return &Client{client.New(&client.Config{URL: url, Token: token, TLSSkipVerify: tlsSkipVerify})}
}

func (c *Client) Pair(ctx context.Context, self *identity.Identity, hostname, scheme string, port int) (*types.FriendPairResponse, error) {
	req := &types.FriendPairRequest{
		Hostname:            hostname,
		IdentityPublicKey:   self.PublicKeyBase64(),
		IdentityFingerprint: self.Fingerprint(),
		X25519PublicKey:     self.X25519PublicBase64(),
		Signature:           base64.StdEncoding.EncodeToString(self.Sign([]byte(self.Fingerprint()))),
		Scheme:              scheme,
		Port:                port,
	}

	return c.Client.Pair(ctx, req)
}

func Index(cfg *config.Config, name string) int {
	for i, f := range cfg.Friends {
		if f.Name == name {
			return i
		}
	}

	return -1
}

func Get(cfg *config.Config, name string) *config.Friend {
	if i := Index(cfg, name); i >= 0 {
		return cfg.Friends[i]
	}

	return nil
}

func Preview(ctx context.Context, c *Client, cfg *config.Config, selfFingerprint string) types.FriendPreview {
	hello, err := c.Client.Hello(ctx)
	if err != nil {
		return types.FriendPreview{Error: err.Error()}
	}

	p := types.FriendPreview{
		Reachable:           true,
		Hostname:            hello.Hostname,
		IdentityFingerprint: hello.IdentityFingerprint,
		IdentityPublicKey:   hello.IdentityPublicKey,
		Version:             hello.Version,
	}
	if selfFingerprint != "" && identity.FingerprintMatch(selfFingerprint, hello.IdentityFingerprint) {
		p.IsSelf = true
	}

	for _, f := range cfg.Friends {
		if f.Identity.Fingerprint != "" && identity.FingerprintMatch(f.Identity.Fingerprint, hello.IdentityFingerprint) {
			p.AlreadyExists = true
		}
	}

	return p
}

func Add(cfg *config.Config, f *config.Friend) error {
	if f.Name == "" {
		return fmt.Errorf("friend name is required")
	}

	if Index(cfg, f.Name) >= 0 {
		return fmt.Errorf("friend %q already exists", f.Name)
	}

	cfg.Friends = append(cfg.Friends, f)

	return nil
}

func Remove(cfg *config.Config, name string) (*config.Friend, error) {
	i := Index(cfg, name)
	if i < 0 {
		return nil, fmt.Errorf("friend %q not found", name)
	}

	f := cfg.Friends[i]
	cfg.Friends = append(cfg.Friends[:i], cfg.Friends[i+1:]...)

	return f, nil
}

func TaggedSections(cfg *config.Config, name string) []types.FriendTaggedSection {
	var out []types.FriendTaggedSection
	for ifName, wg := range cfg.Wireguard {
		if wg.Friend == name {
			out = append(out, types.FriendTaggedSection{Section: "wireguard", Key: ifName})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Section != out[j].Section {
			return out[i].Section < out[j].Section
		}

		return out[i].Key < out[j].Key
	})

	return out
}

func RemoveTagged(cfg *config.Config, name string) {
	for ifName, wg := range cfg.Wireguard {
		if wg.Friend == name {
			delete(cfg.Wireguard, ifName)
		}
	}
}

func PinIdentity(f *config.Friend, fingerprint, publicKey, x25519PublicKey string) error {
	if fingerprint == "" {
		return fmt.Errorf("empty fingerprint")
	}

	if f.Identity.Fingerprint != "" && !identity.FingerprintMatch(f.Identity.Fingerprint, fingerprint) {
		return fmt.Errorf("identity fingerprint mismatch for friend %q (possible MITM)", f.Name)
	}

	f.Identity.Fingerprint = fingerprint
	f.Identity.PublicKey = publicKey
	f.Identity.X25519PublicKey = x25519PublicKey

	return nil
}
