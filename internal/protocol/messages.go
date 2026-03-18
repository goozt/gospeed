package protocol

import "encoding/json"

// Envelope wraps every control message for uniform decoding.
type Envelope struct {
	Type MsgType         `json:"type"`
	Body json.RawMessage `json:"body,omitempty"`
}

// Hello is sent by the client to initiate a session.
type Hello struct {
	Version  int    `json:"version"`
	ClientID string `json:"client_id"`
}

// HelloAck is the server's reply to Hello.
type HelloAck struct {
	Version   int    `json:"version"`
	SessionID string `json:"session_id"`
}

// TestRequest asks the server to prepare a specific test.
type TestRequest struct {
	Test   TestType        `json:"test"`
	Params json.RawMessage `json:"params,omitempty"`
}

// TCPParams configures a TCP throughput test.
type TCPParams struct {
	Streams  int  `json:"streams"`
	Duration int  `json:"duration"` // seconds
	Reverse  bool `json:"reverse"`  // server→client
}

// UDPParams configures a UDP throughput test.
type UDPParams struct {
	Bandwidth  int64 `json:"bandwidth"`   // target bits/sec, 0 = unlimited
	Duration   int   `json:"duration"`    // seconds
	PacketSize int   `json:"packet_size"` // bytes
}

// JitterParams configures a jitter test.
type JitterParams struct {
	IntervalMs int `json:"interval_ms"` // ms between packets
	Count      int `json:"count"`       // number of packets
}

// MTUParams configures a path MTU discovery test.
type MTUParams struct {
	MinSize int `json:"min_size"`
	MaxSize int `json:"max_size"`
}

// BufferbloatParams configures a bufferbloat/responsiveness test.
type BufferbloatParams struct {
	Duration int `json:"duration"` // seconds of load
	Streams  int `json:"streams"`
}

// LatencyParams configures a latency test.
type LatencyParams struct {
	Count int `json:"count"` // number of samples
}

// TestReady tells the client the server is prepared.
type TestReady struct {
	DataPort int `json:"data_port,omitempty"`
}

// TestStart signals the client to begin sending/receiving data.
type TestStart struct{}

// TestResultMsg carries metrics back to the client.
type TestResultMsg struct {
	Test    TestType        `json:"test"`
	Metrics json.RawMessage `json:"metrics"`
}

// ErrorMsg carries an error from server to client.
type ErrorMsg struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Goodbye terminates the session.
type Goodbye struct{}
