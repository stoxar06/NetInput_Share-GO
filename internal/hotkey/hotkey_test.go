package hotkey

import (
	"testing"

	"github.com/netinput/netinput-share/internal/protocol"
)

func keyDown(code uint16) protocol.Packet {
	return protocol.Packet{Type: protocol.PacketKeyDown, Button: code, Value: 1}
}

func keyUp(code uint16) protocol.Packet {
	return protocol.Packet{Type: protocol.PacketKeyUp, Button: code, Value: 0}
}

func TestHotkey_triggered(t *testing.T) {
	fired := false
	d := New([]Spec{
		{Mods: []uint16{KeyLeftCtrl, KeyLeftAlt}, Key: KeyRight, Handler: func() { fired = true }},
	})

	d.Feed(keyDown(KeyLeftCtrl))
	d.Feed(keyDown(KeyLeftAlt))
	consumed := d.Feed(keyDown(KeyRight))

	if !fired {
		t.Error("expected hotkey handler to fire")
	}
	if !consumed {
		t.Error("expected packet to be consumed")
	}
}

func TestHotkey_notTriggered_missingMod(t *testing.T) {
	fired := false
	d := New([]Spec{
		{Mods: []uint16{KeyLeftCtrl, KeyLeftAlt}, Key: KeyRight, Handler: func() { fired = true }},
	})

	d.Feed(keyDown(KeyLeftCtrl)) // only ctrl, no alt
	consumed := d.Feed(keyDown(KeyRight))

	if fired {
		t.Error("expected handler NOT to fire without all mods")
	}
	if consumed {
		t.Error("expected packet NOT to be consumed")
	}
}

func TestHotkey_notConsumed_afterModRelease(t *testing.T) {
	fired := false
	d := New([]Spec{
		{Mods: []uint16{KeyLeftCtrl, KeyLeftAlt}, Key: KeyRight, Handler: func() { fired = true }},
	})

	d.Feed(keyDown(KeyLeftCtrl))
	d.Feed(keyDown(KeyLeftAlt))
	d.Feed(keyUp(KeyLeftAlt)) // release alt
	d.Feed(keyDown(KeyRight))

	if fired {
		t.Error("expected handler NOT to fire after mod released")
	}
}

func TestHotkey_nonKey_passthrough(t *testing.T) {
	d := New([]Spec{
		{Mods: []uint16{KeyLeftCtrl}, Key: KeyRight, Handler: func() {}},
	})
	pkt := protocol.Packet{Type: protocol.PacketMouseMove, DX: 5, DY: 0}
	if d.Feed(pkt) {
		t.Error("mouse packet should never be consumed")
	}
}

func TestHotkey_repeatNotFired(t *testing.T) {
	count := 0
	d := New([]Spec{
		{Mods: []uint16{KeyLeftCtrl}, Key: KeyRight, Handler: func() { count++ }},
	})

	d.Feed(keyDown(KeyLeftCtrl))
	d.Feed(keyDown(KeyRight))
	// repeat event (Value=2)
	repeat := protocol.Packet{Type: protocol.PacketKeyDown, Button: KeyRight, Value: 2}
	d.Feed(repeat)

	if count != 1 {
		t.Errorf("expected handler fired once, got %d", count)
	}
}
