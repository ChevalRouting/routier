//go:build !linux

package setup

import "github.com/ChevalRouting/routier/pkg/types"

func currentHost() (types.SetupHost, error) {
	return types.SetupHost{DhcpIfaces: []string{}}, nil
}
