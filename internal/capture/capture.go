// Package capture reads raw mouse and keyboard events from Linux evdev devices.
// Requires user to be in 'input' group or run as root.
package capture

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	evdev "github.com/gvalkov/golang-evdev"

	"github.com/netinput/netinput-share/internal/protocol"
)

// Capturer reads input events from evdev devices and emits protocol.Packets.
type Capturer struct {
	keyboardPath string
	mousePath    string
	out          chan<- protocol.Packet
}

// New creates a Capturer.
func New(keyboardPath, mousePath string, out chan<- protocol.Packet) *Capturer {
	return &Capturer{
		keyboardPath: keyboardPath,
		mousePath:    mousePath,
		out:          out,
	}
}

// Run starts capturing input events. Blocks until ctx is cancelled.
func (c *Capturer) Run(ctx context.Context) error {
	kb, err := openWithRetry(ctx, c.keyboardPath)
	if err != nil {
		return fmt.Errorf("capture: open keyboard: %w", err)
	}
	defer kb.File.Close()

	ms, err := openWithRetry(ctx, c.mousePath)
	if err != nil {
		return fmt.Errorf("capture: open mouse: %w", err)
	}
	defer ms.File.Close()

	if err := kb.Grab(); err != nil {
		return fmt.Errorf("capture: grab keyboard: %w", err)
	}
	defer kb.Release()

	if err := ms.Grab(); err != nil {
		return fmt.Errorf("capture: grab mouse: %w", err)
	}
	defer ms.Release()

	slog.Info("capture: started", "keyboard", c.keyboardPath, "mouse", c.mousePath)

	// Close devices when ctx is cancelled to unblock ReadOne.
	go func() {
		<-ctx.Done()
		kb.File.Close()
		ms.File.Close()
	}()

	errCh := make(chan error, 2)
	go func() { errCh <- c.readKeyboard(ctx, kb) }()
	go func() { errCh <- c.readMouse(ctx, ms) }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
}

func (c *Capturer) readKeyboard(ctx context.Context, dev *evdev.InputDevice) error {
	for {
		ev, err := dev.ReadOne()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("capture keyboard read: %w", err)
		}
		pkt, ok := convertKeyEvent(*ev)
		if !ok {
			continue
		}
		pkt.Timestamp = time.Now().UnixNano()
		select {
		case c.out <- pkt:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *Capturer) readMouse(ctx context.Context, dev *evdev.InputDevice) error {
	var dx, dy int32
	for {
		ev, err := dev.ReadOne()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("capture mouse read: %w", err)
		}

		switch ev.Type {
		case uint16(evdev.EV_REL):
			switch ev.Code {
			case uint16(evdev.REL_X):
				dx += ev.Value
			case uint16(evdev.REL_Y):
				dy += ev.Value
			case uint16(evdev.REL_WHEEL):
				pkt := protocol.Packet{
					Type:      protocol.PacketMouseScroll,
					DY:        ev.Value,
					Timestamp: time.Now().UnixNano(),
				}
				select {
				case c.out <- pkt:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

		case uint16(evdev.EV_SYN):
			if ev.Code == 0 && (dx != 0 || dy != 0) { // SYN_REPORT = 0
				pkt := protocol.Packet{
					Type:      protocol.PacketMouseMove,
					DX:        dx,
					DY:        dy,
					Timestamp: time.Now().UnixNano(),
				}
				dx, dy = 0, 0
				select {
				case c.out <- pkt:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

		case uint16(evdev.EV_KEY):
			pkt := protocol.Packet{
				Type:      protocol.PacketMouseButton,
				Button:    ev.Code,
				Value:     ev.Value,
				Timestamp: time.Now().UnixNano(),
			}
			select {
			case c.out <- pkt:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// ListDevices scans /dev/input/event* and returns detected keyboards and mice.
func ListDevices() (keyboards []string, mice []string, err error) {
	devices, err := evdev.ListInputDevices()
	if err != nil {
		return nil, nil, fmt.Errorf("ListDevices: %w", err)
	}
	for _, dev := range devices {
		caps := dev.Capabilities()
		if isKeyboard(caps) {
			keyboards = append(keyboards, dev.Fn)
		}
		if isMouse(caps) {
			mice = append(mice, dev.Fn)
		}
		dev.File.Close()
	}
	return keyboards, mice, nil
}

func isKeyboard(caps map[evdev.CapabilityType][]evdev.CapabilityCode) bool {
	codes, ok := caps[evdev.EV_KEY]
	if !ok {
		return false
	}
	for _, code := range codes {
		if code == evdev.KEY_A {
			return true
		}
	}
	return false
}

func isMouse(caps map[evdev.CapabilityType][]evdev.CapabilityCode) bool {
	codes, ok := caps[evdev.EV_REL]
	if !ok {
		return false
	}
	hasX, hasY := false, false
	for _, code := range codes {
		switch code {
		case evdev.REL_X:
			hasX = true
		case evdev.REL_Y:
			hasY = true
		}
	}
	return hasX && hasY
}

func convertKeyEvent(ev evdev.InputEvent) (protocol.Packet, bool) {
	if ev.Type != uint16(evdev.EV_KEY) {
		return protocol.Packet{}, false
	}
	// Mouse buttons share EV_KEY; skip BTN_* range (0x100–0x1ff) here,
	// they're handled in the mouse reader.
	if ev.Code >= 0x100 {
		return protocol.Packet{}, false
	}
	pktType := protocol.PacketKeyUp
	if ev.Value == 1 || ev.Value == 2 {
		pktType = protocol.PacketKeyDown
	}
	return protocol.Packet{
		Type:   pktType,
		Button: ev.Code,
		Value:  ev.Value,
	}, true
}

// openWithRetry tries to open an evdev device, retrying every 5s until ctx is done.
func openWithRetry(ctx context.Context, path string) (*evdev.InputDevice, error) {
	for {
		dev, err := evdev.Open(path)
		if err == nil {
			return dev, nil
		}
		slog.Warn("capture: device open failed, retrying", "path", path, "err", err)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("open %s: %w", path, ctx.Err())
		case <-time.After(5 * time.Second):
		}
	}
}
