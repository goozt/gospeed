package server

import (
	"context"
	"encoding/json"
	"net"

	"github.com/goozt/gospeed/internal/protocol"
)

// TestHandler runs a test on the server side and returns metrics.
type TestHandler func(ctx context.Context, conn net.Conn, params json.RawMessage) (any, error)

// handlers maps test types to their server-side handlers.
var handlers = map[protocol.TestType]TestHandler{}

// RegisterHandler adds a server-side test handler.
func RegisterHandler(t protocol.TestType, h TestHandler) {
	handlers[t] = h
}

// RegisteredTests returns the list of test types with registered handlers.
func RegisteredTests() []protocol.TestType {
	tests := make([]protocol.TestType, 0, len(handlers))
	for t := range handlers {
		tests = append(tests, t)
	}
	return tests
}
