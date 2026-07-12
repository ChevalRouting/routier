package kea

import "encoding/base64"

const (
	LocalSock4 = "/run/kea/kea-dhcp4-ctrl.sock"
	LocalSock6 = "/run/kea/kea-dhcp6-ctrl.sock"
)

func basicAuth(user, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}

func LoadLocal() (*Client, error) {
	return newLocal(), nil
}
