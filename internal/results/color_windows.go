//go:build windows

package results

import "syscall"

func enableWindowsVT100() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("SetConsoleMode")
	handle, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return false
	}
	var mode uint32
	getMode := kernel32.NewProc("GetConsoleMode")
	r, _, _ := getMode.Call(uintptr(handle), uintptr(syscall.Handle(uintptr(0))))
	_ = r
	// Try directly: get current mode and add ENABLE_VIRTUAL_TERMINAL_PROCESSING (0x0004).
	err = syscall.GetConsoleMode(handle, &mode)
	if err != nil {
		return false
	}
	const enableVTP = 0x0004
	r1, _, e := proc.Call(uintptr(handle), uintptr(mode|enableVTP))
	if r1 == 0 {
		_ = e
		return false
	}
	return true
}
