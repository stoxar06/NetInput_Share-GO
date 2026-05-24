// Package hotkey detects modifier+key combinations from a stream of protocol.Packets.
package hotkey

import "github.com/netinput/netinput-share/internal/protocol"

// Linux evdev key codes for common modifier/direction keys.
const (
	KeyLeftCtrl  uint16 = 29
	KeyRightCtrl uint16 = 97
	KeyLeftAlt   uint16 = 56
	KeyRightAlt  uint16 = 100
	KeyLeft      uint16 = 105
	KeyRight     uint16 = 106
)

// Spec describes one hotkey combination.
type Spec struct {
	Mods    []uint16 // modifier codes that must be held
	Key     uint16   // trigger key code (on press)
	Handler func()
}

// Detector tracks live key state and fires handlers on matching combos.
type Detector struct {
	pressed map[uint16]bool
	specs   []Spec
}

// New creates a Detector with the given hotkey specs.
func New(specs []Spec) *Detector {
	return &Detector{
		pressed: make(map[uint16]bool),
		specs:   specs,
	}
}

// Feed processes one key packet.
// Returns true when the packet is consumed by a hotkey and must not be forwarded.
func (d *Detector) Feed(pkt protocol.Packet) bool {
	if pkt.Type != protocol.PacketKeyDown && pkt.Type != protocol.PacketKeyUp {
		return false
	}

	// Update pressed state.
	if pkt.Value == 0 { // release
		delete(d.pressed, pkt.Button)
		return false
	}
	d.pressed[pkt.Button] = true

	if pkt.Value != 1 { // only fire on initial press, not repeat
		return false
	}

	for _, spec := range d.specs {
		if pkt.Button != spec.Key {
			continue
		}
		if d.modsHeld(spec.Mods) {
			spec.Handler()
			return true
		}
	}
	return false
}

func (d *Detector) modsHeld(mods []uint16) bool {
	for _, mod := range mods {
		if !d.pressed[mod] {
			return false
		}
	}
	return true
}
