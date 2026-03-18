package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

const maxMessageSize = 1 << 20 // 1 MiB

// WriteMsg encodes v as JSON and writes it as a length-prefixed frame.
// Wire format: [4-byte big-endian length][JSON payload].
func WriteMsg(conn net.Conn, msgType MsgType, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	env := Envelope{Type: msgType, Body: body}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	if len(data) > maxMessageSize {
		return fmt.Errorf("message too large: %d bytes", len(data))
	}

	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
	if _, err := conn.Write(hdr[:]); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}
	return nil
}

// ReadMsg reads a length-prefixed frame and decodes the envelope.
func ReadMsg(conn net.Conn) (*Envelope, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	size := binary.BigEndian.Uint32(hdr[:])
	if size > uint32(maxMessageSize) {
		return nil, fmt.Errorf("message too large: %d bytes", size)
	}

	data := make([]byte, size)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	return &env, nil
}

// DecodeBody unmarshals the envelope body into dst.
func DecodeBody(env *Envelope, dst any) error {
	return json.Unmarshal(env.Body, dst)
}

// WriteSessionID writes the 16-byte session ID as the first bytes on a data connection.
func WriteSessionID(conn net.Conn, sessionID [16]byte) error {
	_, err := conn.Write(sessionID[:])
	return err
}

// ReadSessionID reads a 16-byte session ID from a data connection.
func ReadSessionID(conn net.Conn) ([16]byte, error) {
	var id [16]byte
	_, err := io.ReadFull(conn, id[:])
	return id, err
}
