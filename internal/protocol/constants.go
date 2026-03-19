package protocol

const (
	ProtocolVersion = 1
	DefaultPort     = 9000
	DefaultAddr     = "localhost:9000"
	Magic           = "GOSPEED"

	// Data packet header size: 8-byte sequence + 8-byte timestamp.
	DataHeaderSize = 16
)

// TestType identifies a specific test.
type TestType string

const (
	TestLatency     TestType = "latency"
	TestMTU         TestType = "mtu"
	TestTCP         TestType = "tcp"
	TestUDP         TestType = "udp"
	TestJitter      TestType = "jitter"
	TestBufferbloat TestType = "bufferbloat"
	TestDNS         TestType = "dns"
	TestConnect     TestType = "connect"
	TestBidir       TestType = "bidir"
)

// DefaultTests is the scientifically chosen default test group.
// Order: MTU discovery → baseline latency → TCP throughput → bufferbloat →
// UDP throughput+loss → jitter.
var DefaultTests = []TestType{
	TestMTU,
	TestLatency,
	TestTCP,
	TestBufferbloat,
	TestUDP,
	TestJitter,
}

// AllTests lists every available test type.
var AllTests = []TestType{
	TestLatency,
	TestMTU,
	TestTCP,
	TestUDP,
	TestJitter,
	TestBufferbloat,
	TestDNS,
	TestConnect,
	TestBidir,
}

// MsgType identifies a control-channel message.
type MsgType string

const (
	MsgHello      MsgType = "hello"
	MsgHelloAck   MsgType = "hello_ack"
	MsgTestReq    MsgType = "test_req"
	MsgTestReady  MsgType = "test_ready"
	MsgTestStart  MsgType = "test_start"
	MsgTestResult MsgType = "test_result"
	MsgError      MsgType = "error"
	MsgGoodbye    MsgType = "goodbye"
)
