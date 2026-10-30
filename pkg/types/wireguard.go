package types

type PubKeyResponse struct {
	PublicKey string `json:"public_key"`
}

type KeygenResponse struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

type WGEndpoint struct {
	Interface  string `json:"interface"`
	Address    string `json:"address,omitempty" validate:"optional"`
	ListenPort int    `json:"listen_port,omitempty" validate:"optional"`
}

type DeriveWireguardRequest struct {
	Subnet         string     `json:"subnet" validate:"required"`
	LocalEndpoint  WGEndpoint `json:"local_endpoint"`
	FriendEndpoint WGEndpoint `json:"friend_endpoint"`
}

type DeriveWireguardResult struct {
	Interface string `json:"interface"`
	Friend    string `json:"friend"`
	Pushed    bool   `json:"pushed"`
	PushError string `json:"push_error,omitempty" validate:"optional"`
}

type DeriveTunnelRequest struct {
	Mode           string     `json:"mode" validate:"required"`
	Subnet         string     `json:"subnet" validate:"required"`
	LocalEndpoint  WGEndpoint `json:"local_endpoint"`
	FriendEndpoint WGEndpoint `json:"friend_endpoint"`
}

type DeriveTunnelResult struct {
	Interface string `json:"interface"`
	Friend    string `json:"friend"`
	Pushed    bool   `json:"pushed"`
	PushError string `json:"push_error,omitempty" validate:"optional"`
}

type WireguardPeerStatus struct {
	Name            string   `json:"name,omitempty" validate:"optional"`
	PublicKey       string   `json:"public_key"`
	Endpoint        string   `json:"endpoint,omitempty" validate:"optional"`
	AllowedIPs      []string `json:"allowed_ips"`
	LatestHandshake int64    `json:"latest_handshake"`
	RxBytes         int64    `json:"rx_bytes"`
	TxBytes         int64    `json:"tx_bytes"`
}

type WireguardInterfaceStatus struct {
	Interface  string                `json:"interface"`
	PublicKey  string                `json:"public_key"`
	ListenPort int                   `json:"listen_port"`
	Peers      []WireguardPeerStatus `json:"peers"`
}
