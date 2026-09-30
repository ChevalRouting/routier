package bind

const (
	ConfDir     = "/etc/bind"
	RndcKey     = ConfDir + "/rndc.key"
	ControlAddr = "127.0.0.1"
	ControlPort = 953

	DefaultResolver = "127.0.0.1"
	DefaultPort     = 53
)

func LoadLocal() (*Client, error) {
	return New(RndcKey), nil
}
