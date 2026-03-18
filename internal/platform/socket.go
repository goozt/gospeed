package platform

import "net"

// TuneConn applies platform-optimal socket options to a connection.
// bufSize is the desired send/receive buffer size in bytes. If 0, defaults are used.
func TuneConn(conn net.Conn, bufSize int) error {
	return tuneConn(conn, bufSize)
}

// SetDontFragment sets the DF bit on a UDP connection for PMTU discovery.
func SetDontFragment(conn *net.UDPConn) error {
	return setDontFragment(conn)
}

// MaxBufferSize returns the largest socket buffer size the OS will accept.
func MaxBufferSize() int {
	return maxBufferSize()
}
