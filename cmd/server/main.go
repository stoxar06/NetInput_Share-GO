// NetInput Share — Server Mode
// Run this on the laptop that has the physical mouse and keyboard connected.
// Usage: sudo ./server --config config.json
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/netinput/netinput-share/config"
	"github.com/netinput/netinput-share/internal/capture"
	"github.com/netinput/netinput-share/internal/discovery"
	"github.com/netinput/netinput-share/internal/hotkey"
	"github.com/netinput/netinput-share/internal/network"
	"github.com/netinput/netinput-share/internal/protocol"
	"github.com/netinput/netinput-share/internal/screen"
)

func main() {
	configPath := flag.String("config", "config.json", "path to config.json")
	keyboard := flag.String("keyboard", "", "evdev keyboard device (auto-detect if empty)")
	mouse := flag.String("mouse", "", "evdev mouse device (auto-detect if empty)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	level := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	slog.Info("NetInput Share — Server Mode", "port", cfg.ServerPort, "screens", len(cfg.Screens))

	// Resolve keyboard and mouse device paths.
	keyboardPath, mousePath, err := resolveDevices(*keyboard, *mouse)
	if err != nil {
		slog.Error("device detection failed", "err", err)
		os.Exit(1)
	}
	slog.Info("devices", "keyboard", keyboardPath, "mouse", mousePath)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Build screen layout.
	var screens []screen.Screen
	for _, s := range cfg.Screens {
		screens = append(screens, screen.Screen{
			ID:     s.ID,
			Name:   s.Name,
			IP:     s.IP,
			Width:  s.Width,
			Height: s.Height,
		})
	}
	layout := screen.NewLayout(screens, cfg.Layout)

	packets := make(chan protocol.Packet, 256)
	routed := make(chan protocol.Packet, 256)

	capturer := capture.New(keyboardPath, mousePath, packets)
	srv := network.NewServer(cfg.ServerPort, routed)

	switchNext := func() {
		newID, err := layout.SwitchNext()
		if err != nil {
			slog.Warn("hotkey: already at last screen")
			return
		}
		srv.SwitchTo(newID)
		slog.Info("hotkey: switched to next screen", "screenID", newID)
	}
	switchPrev := func() {
		newID, err := layout.SwitchPrev()
		if err != nil {
			slog.Warn("hotkey: already at first screen")
			return
		}
		srv.SwitchTo(newID)
		slog.Info("hotkey: switched to previous screen", "screenID", newID)
	}

	ctrlAlt := []uint16{hotkey.KeyLeftCtrl, hotkey.KeyLeftAlt}
	hk := hotkey.New([]hotkey.Spec{
		{Mods: ctrlAlt, Key: hotkey.KeyRight, Handler: switchNext},
		{Mods: ctrlAlt, Key: hotkey.KeyLeft, Handler: switchPrev},
	})

	// Routing goroutine: edge detection + hotkey interception + fan-out.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case pkt, ok := <-packets:
				if !ok {
					return
				}
				if pkt.Type == protocol.PacketMouseMove {
					newID, switched := layout.UpdateCursor(int(pkt.DX), int(pkt.DY))
					if switched {
						srv.SwitchTo(newID)
					}
				}
				// Hotkey check: consume matching key packets, don't forward them.
				if hk.Feed(pkt) {
					continue
				}
				// Only forward to routed if not on server screen (screen 0).
				if srv.ActiveID() != 0 {
					select {
					case routed <- pkt:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	go func() {
		if err := discovery.Advertise(ctx, "NetInput-Server"); err != nil {
			slog.Warn("discovery advertise error", "err", err)
		}
	}()

	go func() {
		if err := capturer.Run(ctx); err != nil {
			slog.Error("capture error", "err", err)
			cancel()
		}
	}()

	if err := srv.Run(ctx); err != nil {
		slog.Info("server stopped", "reason", err)
	}
}

func resolveDevices(keyboardFlag, mouseFlag string) (string, string, error) {
	if keyboardFlag != "" && mouseFlag != "" {
		return keyboardFlag, mouseFlag, nil
	}

	keyboards, mice, err := capture.ListDevices()
	if err != nil {
		return "", "", fmt.Errorf("list devices: %w", err)
	}

	kbPath := keyboardFlag
	if kbPath == "" {
		if len(keyboards) == 0 {
			return "", "", fmt.Errorf("no keyboard detected; use --keyboard /dev/input/eventX")
		}
		kbPath = keyboards[0]
		if len(keyboards) > 1 {
			slog.Warn("multiple keyboards detected, using first", "path", kbPath, "all", keyboards)
		}
	}

	mPath := mouseFlag
	if mPath == "" {
		if len(mice) == 0 {
			return "", "", fmt.Errorf("no mouse detected; use --mouse /dev/input/eventX")
		}
		mPath = mice[0]
		if len(mice) > 1 {
			slog.Warn("multiple mice detected, using first", "path", mPath, "all", mice)
		}
	}

	return kbPath, mPath, nil
}
