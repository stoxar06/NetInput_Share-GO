package protocol

import (
	"testing"
	"time"
)

func TestEncodeDecode_roundtrip(t *testing.T) {
	original := Packet{
		Type:      PacketMouseMove,
		ScreenID:  2,
		X:         100,
		Y:         200,
		DX:        -5,
		DY:        10,
		Button:    0,
		Value:     0,
		Timestamp: time.Now().UnixNano(),
	}

	data, err := Encode(original)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != original {
		t.Errorf("roundtrip mismatch: got %+v want %+v", got, original)
	}
}

func TestEncodeDecode_allTypes(t *testing.T) {
	types := []uint8{
		PacketMouseMove, PacketMouseButton, PacketMouseScroll,
		PacketKeyDown, PacketKeyUp, PacketSwitchScreen,
		PacketKeepAlive, PacketHandshake, PacketReleaseAll, PacketClipboard,
	}
	for _, typ := range types {
		pkt := Packet{Type: typ, Button: 42, Value: 1}
		data, err := Encode(pkt)
		if err != nil {
			t.Fatalf("type %d Encode: %v", typ, err)
		}
		got, err := Decode(data)
		if err != nil {
			t.Fatalf("type %d Decode: %v", typ, err)
		}
		if got.Type != typ {
			t.Errorf("type %d: got %d", typ, got.Type)
		}
	}
}

func TestEncodeDecode_withData(t *testing.T) {
	pkt := Packet{
		Type: PacketClipboard,
		Data: []byte("hello clipboard"),
	}
	data, err := Encode(pkt)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if string(got.Data) != "hello clipboard" {
		t.Errorf("Data mismatch: got %q", got.Data)
	}
}

func TestDecode_truncated(t *testing.T) {
	_, err := Decode([]byte{0x00, 0x01})
	if err == nil {
		t.Error("expected error for truncated input")
	}
}

func TestDecode_empty(t *testing.T) {
	_, err := Decode([]byte{})
	if err == nil {
		t.Error("expected error for empty input")
	}
}
