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
