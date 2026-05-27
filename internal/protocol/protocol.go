// Package protocol defines the wire format for NetInput Share.
package protocol

import (
	"encoding/binary"
	"fmt"
)

// Packet type constants.
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
	PacketWarpCursor   uint8 = 11 // server → client: warp cursor to absolute (X, Y)
)

// Packet is the universal message unit sent between server and clients.
type Packet struct {
	Type      uint8
	ScreenID  uint8  // 0=server, 1-3=clients
	X, Y      int32  // absolute position (also used for warp destination)
	DX, DY    int32  // relative mouse delta
	Button    uint16 // mouse button or key code
	Value     int32  // 1=press, 0=release, 2=repeat
	Timestamp int64
	Data      []byte // extra payload (e.g. clipboard, handshake)
}

// Binary layout — 36-byte fixed header followed by variable Data:
//
//	 0     Type
//	 1     ScreenID
//	 2-5   X
//	 6-9   Y
//	10-13  DX
//	14-17  DY
//	18-19  Button
//	20-23  Value
//	24-31  Timestamp
//	32-35  len(Data)
//	36+    Data
const headerSize = 36

// Encode serializes p to a flat byte slice (no reflection, no allocations
// beyond the output buffer).
func Encode(p Packet) ([]byte, error) {
	buf := make([]byte, headerSize+len(p.Data))
	buf[0] = p.Type
	buf[1] = p.ScreenID
	binary.BigEndian.PutUint32(buf[2:], uint32(p.X))
	binary.BigEndian.PutUint32(buf[6:], uint32(p.Y))
	binary.BigEndian.PutUint32(buf[10:], uint32(p.DX))
	binary.BigEndian.PutUint32(buf[14:], uint32(p.DY))
	binary.BigEndian.PutUint16(buf[18:], p.Button)
	binary.BigEndian.PutUint32(buf[20:], uint32(p.Value))
	binary.BigEndian.PutUint64(buf[24:], uint64(p.Timestamp))
	binary.BigEndian.PutUint32(buf[32:], uint32(len(p.Data)))
	copy(buf[headerSize:], p.Data)
	return buf, nil
}

// Decode deserializes a flat byte slice into a Packet.
func Decode(data []byte) (Packet, error) {
	if len(data) < headerSize {
		return Packet{}, fmt.Errorf("protocol: packet too short (%d bytes)", len(data))
	}
	var p Packet
	p.Type = data[0]
	p.ScreenID = data[1]
	p.X = int32(binary.BigEndian.Uint32(data[2:]))
	p.Y = int32(binary.BigEndian.Uint32(data[6:]))
	p.DX = int32(binary.BigEndian.Uint32(data[10:]))
	p.DY = int32(binary.BigEndian.Uint32(data[14:]))
	p.Button = binary.BigEndian.Uint16(data[18:])
	p.Value = int32(binary.BigEndian.Uint32(data[20:]))
	p.Timestamp = int64(binary.BigEndian.Uint64(data[24:]))
	dataLen := binary.BigEndian.Uint32(data[32:])
	end := uint64(headerSize) + uint64(dataLen)
	if uint64(len(data)) < end {
		return Packet{}, fmt.Errorf("protocol: data field truncated (need %d, have %d)", end, len(data))
	}
	if dataLen > 0 {
		p.Data = make([]byte, dataLen)
		copy(p.Data, data[headerSize:end])
	}
	return p, nil
}

// HandshakePayload is sent by the client on connect to identify itself.
// It is gob-encoded into Packet.Data so only used once per connection.
type HandshakePayload struct {
	ScreenID uint8
	Name     string
	Width    int32
	Height   int32
	Version  string
}
