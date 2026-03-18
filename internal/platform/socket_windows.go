//go:build windows

package platform

import (
	"fmt"
	"net"
	"syscall"
)

func tuneConn(conn net.Conn, bufSize int) error {
	sc, ok := conn.(syscall.Conn)
	if !ok {
		return nil
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return err
	}
	var setErr error
	raw.Control(func(fd uintptr) {
		if bufSize > 0 {
			if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, bufSize); err != nil {
				setErr = fmt.Errorf("SO_SNDBUF: %w", err)
				return
			}
			if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, bufSize); err != nil {
				setErr = fmt.Errorf("SO_RCVBUF: %w", err)
				return
			}
		}
		syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1)
	})
	return setErr
}

func setDontFragment(conn *net.UDPConn) error {
	sc, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	var setErr error
	sc.Control(func(fd uintptr) {
		// IP_DONTFRAGMENT = 14 on Windows
		if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IP, 14, 1); err != nil {
			setErr = fmt.Errorf("IP_DONTFRAGMENT: %w", err)
		}
	})
	return setErr
}

func maxBufferSize() int {
	return 4 * 1024 * 1024
}
