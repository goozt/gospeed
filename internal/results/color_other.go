//go:build !windows

package results

func enableWindowsVT100() bool {
	return false
}
