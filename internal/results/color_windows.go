//go:build windows

package results

import (
	"syscall"
	"unsafe"
)

func enableWindowsVT100() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")

	handle, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return false
	}

	var mode uint32
	r, _, _ := getConsoleMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return false
	}

	const enableVTP = 0x0004
	r, _, _ = setConsoleMode.Call(uintptr(handle), uintptr(mode|enableVTP))
	return r != 0
}
