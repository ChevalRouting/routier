//go:build !linux

package setup

func nicAddrs(_ string) []string {
	return nil
}
