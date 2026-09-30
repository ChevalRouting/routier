//go:build !linux

package setup

import "github.com/ChevalRouting/routier/pkg/types"

func systemNics() ([]types.SystemNic, error) {
	return []types.SystemNic{}, nil
}
