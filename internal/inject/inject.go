// Package inject creates virtual Linux input devices using uinput
// and injects received protocol.Packets as real input events.
// Requires /dev/uinput access (udev rule or root).
package inject

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/bendahl/uinput"

	"github.com/netinput/netinput-share/internal/protocol"
)

// evdev BTN_* codes used for mouse buttons.
const (
	btnLeft   = 0x110
	btnRight  = 0x111
	btnMiddle = 0x112
)

// Injector holds virtual keyboard and mouse devices.
type Injector struct {
	keyboard    uinput.Keyboard
	mouse       uinput.Mouse
	in          <-chan protocol.Packet
	mu          sync.Mutex
	pressedKeys map[int]bool
}

// New creates an Injector that reads from in channel.
func New(in <-chan protocol.Packet) *Injector {
	return &Injector{
		in:          in,
		pressedKeys: make(map[int]bool),
	}
}

// Start creates the virtual devices and begins injecting events.
func (inj *Injector) Start(ctx context.Context) error {
	kb, err := uinput.CreateKeyboard("/dev/uinput", []byte("NetInput Virtual KB"))
	if err != nil {
		return fmt.Errorf("inject: create keyboard: %w", err)
	}
	defer kb.Close()

	m, err := uinput.CreateMouse("/dev/uinput", []byte("NetInput Virtual Mouse"))
	if err != nil {
		return fmt.Errorf("inject: create mouse: %w", err)
	}
	defer m.Close()

	inj.keyboard = kb
	inj.mouse = m

	slog.Info("inject: virtual devices created")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case pkt, ok := <-inj.in:
			if !ok {
				return fmt.Errorf("inject: input channel closed")
			}
			if err := inj.dispatch(pkt); err != nil {
				slog.Warn("inject: dispatch error", "type", pkt.Type, "err", err)
			}
		}
	}
}

// dispatch routes a packet to the correct virtual device call.
func (inj *Injector) dispatch(pkt protocol.Packet) error {
	switch pkt.Type {
	case protocol.PacketMouseMove:
		return inj.mouse.MoveRelative(pkt.DX, pkt.DY)

	case protocol.PacketMouseButton:
		return inj.mouseButton(int(pkt.Button), pkt.Value)

	case protocol.PacketMouseScroll:
		return inj.mouse.Wheel(false, pkt.DY)

	case protocol.PacketKeyDown:
		key := int(pkt.Button)
		inj.mu.Lock()
		inj.pressedKeys[key] = true
		inj.mu.Unlock()
		return inj.keyboard.KeyDown(key)

	case protocol.PacketKeyUp:
		key := int(pkt.Button)
		inj.mu.Lock()
		delete(inj.pressedKeys, key)
		inj.mu.Unlock()
		return inj.keyboard.KeyUp(key)

	case protocol.PacketReleaseAll:
		return inj.releaseAll()

	case protocol.PacketKeepAlive:
		return nil

	default:
		slog.Warn("inject: unknown packet type", "type", pkt.Type)
	}
	return nil
}

func (inj *Injector) mouseButton(code int, value int32) error {
	switch code {
	case btnLeft:
		if value == 1 {
			return inj.mouse.LeftPress()
		}
		return inj.mouse.LeftRelease()
	case btnRight:
		if value == 1 {
			return inj.mouse.RightPress()
		}
		return inj.mouse.RightRelease()
	case btnMiddle:
		if value == 1 {
			return inj.mouse.MiddlePress()
		}
		return inj.mouse.MiddleRelease()
	}
	return nil
}

func (inj *Injector) releaseAll() error {
	inj.mu.Lock()
	keys := make([]int, 0, len(inj.pressedKeys))
	for k := range inj.pressedKeys {
		keys = append(keys, k)
	}
	inj.pressedKeys = make(map[int]bool)
	inj.mu.Unlock()

	for _, k := range keys {
		if err := inj.keyboard.KeyUp(k); err != nil {
			slog.Warn("inject: releaseAll key error", "key", k, "err", err)
		}
	}

	// Release all mouse buttons defensively.
	_ = inj.mouse.LeftRelease()
	_ = inj.mouse.RightRelease()
	_ = inj.mouse.MiddleRelease()

	slog.Info("inject: release all")
	return nil
}
