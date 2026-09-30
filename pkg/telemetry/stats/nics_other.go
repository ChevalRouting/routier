//go:build !linux

package stats

func PhysicalIfaces() map[string]bool {
	return map[string]bool{}
}
