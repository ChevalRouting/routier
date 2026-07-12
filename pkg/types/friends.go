package types

import "encoding/json"

type FriendVRRPRole struct {
	Instance string `json:"instance"`
	State    string `json:"state"`
}

const SealedContentType = "application/routier-sealed"

const SharedSealedContentType = "application/routier-shared-sealed"

type FriendPoll struct {
	Hello      FriendsHello      `json:"hello"`
	Interfaces []FriendInterface `json:"interfaces,omitempty" validate:"optional"`
	Exports    []FriendVar       `json:"exports,omitempty" validate:"optional"`
	ConfigHash string            `json:"config_hash"`
	Config     json.RawMessage   `json:"config,omitempty" validate:"optional"`
}

type FriendsHello struct {
	Hostname            string           `json:"hostname"`
	IdentityFingerprint string           `json:"identity_fingerprint"`
	IdentityPublicKey   string           `json:"identity_public_key"`
	X25519PublicKey     string           `json:"x25519_public_key,omitempty" validate:"optional"`
	Encryption          bool             `json:"encryption"`
	Version             string           `json:"version"`
	OS                  string           `json:"os,omitempty" validate:"optional"`
	UptimeSec           int64            `json:"uptime_sec"`
	Conntrackd          bool             `json:"conntrackd"`
	ConntrackdRunning   bool             `json:"conntrackd_running"`
	VRRP                []FriendVRRPRole `json:"vrrp,omitempty" validate:"optional"`
	Signature           string           `json:"signature,omitempty" validate:"optional"`
}

type FriendInfo struct {
	Name        string `json:"name"`
	Hostname    string `json:"hostname,omitempty" validate:"optional"`
	URL         string `json:"url"`
	Enabled     bool   `json:"enabled"`
	Fingerprint string `json:"fingerprint,omitempty" validate:"optional"`
	HA          bool   `json:"ha"`
	Paired      bool   `json:"paired,omitempty" validate:"optional"`
}

type FriendPreview struct {
	Reachable           bool   `json:"reachable"`
	Hostname            string `json:"hostname,omitempty" validate:"optional"`
	IdentityFingerprint string `json:"identity_fingerprint,omitempty" validate:"optional"`
	IdentityPublicKey   string `json:"identity_public_key,omitempty" validate:"optional"`
	Version             string `json:"version,omitempty" validate:"optional"`
	IsSelf              bool   `json:"is_self"`
	AlreadyExists       bool   `json:"already_exists"`
	Error               string `json:"error,omitempty" validate:"optional"`
}

type FriendPreviewRequest struct {
	URL           string `json:"url"`
	Token         string `json:"token"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
}

type AddFriendRequest struct {
	Name          string `json:"name" validate:"required"`
	URL           string `json:"url"`
	Token         string `json:"token"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
	Enabled       *bool  `json:"enabled,omitempty" validate:"optional"`
}

type VerifyFriendRequest struct {
	Fingerprint string `json:"fingerprint,omitempty" validate:"optional"`
}

type UpdateFriendRequest struct {
	URL           *string `json:"url,omitempty" validate:"optional"`
	Token         *string `json:"token,omitempty" validate:"optional"`
	TLSSkipVerify *bool   `json:"tls_skip_verify,omitempty" validate:"optional"`
	Enabled       *bool   `json:"enabled,omitempty" validate:"optional"`
}

type FriendPairRequest struct {
	Hostname            string `json:"hostname"`
	IdentityPublicKey   string `json:"identity_public_key"`
	IdentityFingerprint string `json:"identity_fingerprint"`
	X25519PublicKey     string `json:"x25519_public_key,omitempty" validate:"optional"`
	Signature           string `json:"signature"`
	Scheme              string `json:"scheme,omitempty" validate:"optional" enums:"http,https"`
	Port                int    `json:"port,omitempty" validate:"optional"`
}

type FriendPairResponse struct {
	Hostname            string `json:"hostname"`
	IdentityPublicKey   string `json:"identity_public_key"`
	IdentityFingerprint string `json:"identity_fingerprint"`
	X25519PublicKey     string `json:"x25519_public_key,omitempty" validate:"optional"`
	Paired              bool   `json:"paired"`
}

type FriendInterface struct {
	Name      string   `json:"name"`
	Addresses []string `json:"addresses,omitempty" validate:"optional"`
}

type FriendVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ConfigureVRRPRequest struct {
	LocalInterface  string   `json:"local_interface" validate:"required"`
	FriendInterface string   `json:"friend_interface" validate:"required"`
	VRID            int      `json:"vrid" validate:"required"`
	VIPs            []string `json:"vips" validate:"required,min=1"`
	Priority        int      `json:"priority,omitempty" validate:"optional"`
}

type ConfigureVRRPResult struct {
	Status string `json:"status"`
}

type ConfigureConntrackRequest struct {
	LocalInterface  string `json:"local_interface" validate:"required"`
	LocalAddress    string `json:"local_address" validate:"required"`
	FriendInterface string `json:"friend_interface" validate:"required"`
	FriendAddress   string `json:"friend_address" validate:"required"`
	Port            int    `json:"port,omitempty" validate:"optional"`
	AllowInbound    bool   `json:"allow_inbound,omitempty" validate:"optional"`
}

type ConfigureConntrackResult struct {
	Status string `json:"status"`
}

type FriendTaggedSection struct {
	Section string `json:"section"`
	Key     string `json:"key"`
}

type FriendDeleteResult struct {
	Friend   string                `json:"friend"`
	Sections []FriendTaggedSection `json:"sections,omitempty" validate:"optional"`
	Deleted  bool                  `json:"deleted"`
}

type FriendStatus struct {
	Name              string           `json:"name"`
	Reachable         bool             `json:"reachable"`
	RTTms             int64            `json:"rtt_ms,omitempty" validate:"optional"`
	LastSeen          string           `json:"last_seen,omitempty" validate:"optional"`
	LastError         string           `json:"last_error,omitempty" validate:"optional"`
	Hostname          string           `json:"hostname,omitempty" validate:"optional"`
	Version           string           `json:"version,omitempty" validate:"optional"`
	OS                string           `json:"os,omitempty" validate:"optional"`
	Fingerprint       string           `json:"fingerprint,omitempty" validate:"optional"`
	IdentityMatch     bool             `json:"identity_match"`
	Encryption        bool             `json:"encryption"`
	ConntrackdRunning bool             `json:"conntrackd_running,omitempty" validate:"optional"`
	VRRP              []FriendVRRPRole `json:"vrrp,omitempty" validate:"optional"`
}
