// Package protocol defines the wire format for NetInput Share.
// All packets are encoded/decoded using encoding/gob over TCP.
package protocol

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

// Packet type constants
const (
	PacketMouseMove    uint8 = 1
	PacketMouseButton  uint8 = 2
	PacketMouseScroll  uint8 = 3
	PacketKeyDown      uint8 = 4
	PacketKeyUp        uint8 = 5
	PacketSwitchScreen uint8 = 6
	PacketKeepAlive    uint8 = 7
	PacketHandshake    uint8 = 8
	PacketReleaseAll   uint8 = 9
	PacketClipboard    uint8 = 10
)

// Packet is the universal message unit sent between server and clients.
type Packet struct {
	Type      uint8
	ScreenID  uint8  // 0=server, 1,2,3 = clients
	X, Y      int32  // absolute mouse position
	DX, DY    int32  // relative mouse delta
	Button    uint16 // mouse button or key code
	Value     int32  // 1=press, 0=release, 2=repeat
	Timestamp int64  // UnixNano
	Data      []byte // extra payload (e.g. clipboard text)
}

// Encode serializes a Packet to bytes using gob.
func Encode(p Packet) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(p); err != nil {
		return nil, fmt.Errorf("protocol encode: %w", err)
	}
	return buf.Bytes(), nil
}

// Decode deserializes bytes into a Packet.
func Decode(data []byte) (Packet, error) {
	var p Packet
	buf := bytes.NewReader(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&p); err != nil {
		return Packet{}, fmt.Errorf("protocol decode: %w", err)
	}
	return p, nil
}

// HandshakePayload is sent by client on connect to identify itself.
type HandshakePayload struct {
	ScreenID uint8
	Name     string
	Width    int32
	Height   int32
	Version  string
}
