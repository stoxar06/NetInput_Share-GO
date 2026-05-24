# NetInput Share

> Software KVM over WiFi for Ubuntu Linux — control up to 4 laptops from one keyboard and mouse.

![Go 1.22](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)
![License MIT](https://img.shields.io/badge/license-MIT-D97757?style=flat-square)
![Platform Linux](https://img.shields.io/badge/platform-Linux-1A1A1A?style=flat-square)

---

## What is this?

NetInput Share turns one physical keyboard and mouse into a shared input device for up to 4 laptops on the same LAN — no USB switch, no extra hardware. Move your mouse past the edge of the server screen and input jumps to the next laptop. Works entirely in software using Linux's `evdev` (capture) and `uinput` (injection) kernel interfaces over a plain TCP connection.

```
[ Server Laptop ]  ──TCP──►  [ Client Laptop 1 ]
  Physical KB+Mouse           Virtual KB+Mouse
                    ──TCP──►  [ Client Laptop 2 ]
                    ──TCP──►  [ Client Laptop 3 ]
```

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Usage — Server](#usage--server-mode)
4. [Usage — Client](#usage--client-mode)
5. [Screen Layout](#screen-layout)
6. [Hotkeys](#hotkeys)
7. [Configuration Reference](#configuration-reference)
8. [Architecture](#architecture)
9. [Permissions](#permissions)
10. [Troubleshooting](#troubleshooting)
11. [Development](#development)

---

## Prerequisites

| Requirement | Notes |
|---|---|
| Ubuntu 22.04+ (or any systemd Linux) | Other distros work with manual dependency install |
| Go 1.22+ | Installed automatically by `install.sh` |
| All laptops on the same LAN (WiFi or Ethernet) | No internet required |
| User in `input` group **or** run as root | `install.sh` handles this |

---

## Installation

Run the installer **once on every laptop** (server and all clients):

```bash
git clone https://github.com/stoxar06/NetInput_Share-GO.git
cd NetInput_Share-GO
chmod +x install.sh && ./install.sh
```

The script does five things:

1. Installs Go (via `apt`) if not already present
2. Installs system libraries required by Fyne (OpenGL, X11, DBus, pkg-config)
3. Writes udev rules so `/dev/uinput` and `/dev/input/event*` are accessible without root
4. Adds your user to the `input` group
5. Runs `go mod tidy` and builds `netinput-server` and `netinput-client`

> **Important:** Log out and log back in after the installer runs so the `input` group membership takes effect. Verify with `groups | grep input`.

---

## Usage — Server Mode

Run this on the laptop that has the **physical keyboard and mouse** plugged in.

```bash
# Auto-detect keyboard and mouse, use default config.json
./netinput-server

# Specify config file location
./netinput-server --config /etc/netinput/config.json

# Pin specific evdev devices (useful when multiple keyboards/mice exist)
./netinput-server --keyboard /dev/input/event3 --mouse /dev/input/event5
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--config` | `config.json` | Path to the JSON config file |
| `--keyboard` | auto-detect | evdev device path for the keyboard |
| `--mouse` | auto-detect | evdev device path for the mouse |

### Auto-detection

If `--keyboard` or `--mouse` are not given, the server scans every `/dev/input/event*` device and picks:

- **Keyboard**: first device that reports `EV_KEY` capability with `KEY_A`
- **Mouse**: first device that reports `EV_REL` capability with `REL_X` + `REL_Y`

If multiple matches are found, the first is used and a warning is logged. To list device paths:

```bash
ls -la /dev/input/by-id/       # stable symlinks by device name
cat /proc/bus/input/devices    # all devices with capabilities
```

---

## Usage — Client Mode

Run this on each **remote laptop** that will be controlled.

```bash
# Option A — auto-discover server on LAN (recommended)
./netinput-client --discover --id 1   # Screen 1 (right of server)
./netinput-client --discover --id 2   # Screen 2
./netinput-client --discover --id 3   # Screen 3

# Option B — specify server IP directly
./netinput-client --server 192.168.1.100 --id 1
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--config` | `config.json` | Path to the JSON config file |
| `--server` | — | Server IP address (required unless `--discover`) |
| `--id` | 1 | Screen ID for this client (1, 2, or 3) |
| `--discover` | false | Auto-discover server via mDNS on the LAN |

### Screen IDs

Each client must have a **unique** ID. The server is always Screen 0.

```
[ Screen 0 ]  [ Screen 1 ]  [ Screen 2 ]  [ Screen 3 ]
  Server        --id 1        --id 2        --id 3
```

---

## Screen Layout

Screens are arranged in a horizontal row, left to right, starting from the server.

```
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│          │  │          │  │          │  │          │
│ Screen 0 │  │ Screen 1 │  │ Screen 2 │  │ Screen 3 │
│ (server) │  │ Laptop 2 │  │ Laptop 3 │  │ Laptop 4 │
│          │  │          │  │          │  │          │
└──────────┘  └──────────┘  └──────────┘  └──────────┘
     ▲
  Physical KB+Mouse here
```

### Edge switching

- Mouse past the **right edge** → routes input to the next screen (higher ID)
- Mouse past the **left edge** → routes input to the previous screen (lower ID)
- The cursor wraps to the opposite edge on the new screen

Set `"edge_switch": false` in `config.json` to disable and use only hotkeys.

---

## Hotkeys

These work regardless of which screen is currently active:

| Shortcut | Action |
|---|---|
| `Ctrl + Alt + →` | Switch focus to the **next** screen (right) |
| `Ctrl + Alt + ←` | Switch focus to the **previous** screen (left) |

When a hotkey fires:
1. The hotkey keystrokes are **consumed** — not forwarded to the current client
2. A `ReleaseAll` packet is sent to the old screen so no keys remain stuck
3. The new screen receives all subsequent input

---

## Configuration Reference

`config.json` is optional — all fields have sensible defaults.

```jsonc
{
  "mode":           "server",          // "server" | "client"
  "server_port":    24800,             // TCP port (same on all machines)
  "server_ip":      "auto",            // "auto" = bind all interfaces
  "layout":         "horizontal",      // screen arrangement
  "edge_switch":    true,              // switch when mouse hits screen edge
  "hotkey_switch":  "ctrl+alt+right",  // force-switch right
  "hotkey_back":    "ctrl+alt+left",   // force-switch left
  "clipboard_sync": true,              // clipboard sharing (Phase 3)
  "log_level":      "info",            // "info" | "debug"
  "screens": [
    { "id": 0, "name": "Main",    "ip": "self",          "width": 1920, "height": 1080 },
    { "id": 1, "name": "Laptop2", "ip": "192.168.1.101", "width": 1920, "height": 1080 },
    { "id": 2, "name": "Laptop3", "ip": "192.168.1.102", "width": 1920, "height": 1080 },
    { "id": 3, "name": "Laptop4", "ip": "192.168.1.103", "width": 1920, "height": 1080 }
  ]
}
```

### `screens` fields

| Field | Type | Description |
|---|---|---|
| `id` | int | 0 = server, 1–3 = clients. Must be unique. |
| `name` | string | Human-readable label shown in logs |
| `ip` | string | `"self"` for server screen; client IP for others |
| `width` | int | Screen width in pixels — used for edge detection |
| `height` | int | Screen height in pixels — used for cursor clamping |

---

## Architecture

```
Server laptop (physical KB + mouse)
┌──────────────────────────────────────────────────┐
│  /dev/input/event*                               │
│        │  evdev (exclusive grab)                 │
│        ▼                                         │
│  Evdev Capturer ──chan Packet──► Routing         │
│                                  Goroutine       │
│                                  │               │
│                       ┌──────────┼──────────┐    │
│                       │          │          │    │
│                  Edge det.   Hotkey      Forward │
│                  SwitchTo() SwitchTo()  to client│
│                       └──────────┼──────────┘    │
│                                  │               │
│                            TCP Server :24800     │
│                            gob · keep-alive 1s   │
└──────────────────┬───────────────────────────────┘
                   │  TCP / LAN
       ┌───────────┴───────────┐
       ▼                       ▼
Client (--id 1)          Client (--id 2)  ...
┌─────────────────┐
│  TCP Client     │
│  (reconnects)   │
│       │         │
│  Uinput Inject  │
│       │         │
│ /dev/uinput     │
│ Virtual KB+Mouse│
└─────────────────┘
```

### Wire format

All packets are gob-encoded over a 4-byte length-prefixed TCP stream.

```go
type Packet struct {
    Type      uint8   // PacketMouseMove, PacketKeyDown, …
    ScreenID  uint8   // 0=server, 1–3=clients
    X, Y      int32   // absolute cursor position
    DX, DY    int32   // relative mouse delta
    Button    uint16  // evdev key/button code
    Value     int32   // 1=press, 0=release, 2=repeat
    Timestamp int64   // UnixNano
    Data      []byte  // clipboard payload / handshake
}
```

| Packet type | Value | Purpose |
|---|---|---|
| `PacketMouseMove` | 1 | Relative cursor delta (DX, DY) |
| `PacketMouseButton` | 2 | Mouse button press/release |
| `PacketMouseScroll` | 3 | Scroll wheel delta |
| `PacketKeyDown` | 4 | Key press or repeat |
| `PacketKeyUp` | 5 | Key release |
| `PacketSwitchScreen` | 6 | Notify client it is now active |
| `PacketKeepAlive` | 7 | Heartbeat — sent every 1 s |
| `PacketHandshake` | 8 | Client identifies itself on connect |
| `PacketReleaseAll` | 9 | Release all held keys on old screen |
| `PacketClipboard` | 10 | Clipboard text sync (Phase 3) |

### Package layout

| Package | Responsibility |
|---|---|
| `internal/capture` | Opens evdev devices, grabs exclusively, emits Packets |
| `internal/inject` | Creates virtual KB+mouse via uinput, dispatches Packets |
| `internal/protocol` | Gob encode/decode; Packet struct; all PacketXxx constants |
| `internal/network/server` | TCP listener, per-client goroutines, keep-alive, SwitchTo |
| `internal/network/client` | TCP dial, handshake, reconnect with exponential backoff |
| `internal/screen` | Logical cursor tracking, edge detection, SwitchNext/Prev |
| `internal/hotkey` | Modifier+key detector; intercepts Ctrl+Alt+←/→ |
| `internal/discovery` | mDNS advertise (server) and browse (client) via zeroconf |
| `internal/gui` | Fyne tray + config window (Phase 3 stub) |
| `config` | Load/save/validate config.json |

---

## Permissions

| Device | Required by | How to grant |
|---|---|---|
| `/dev/input/event*` | Server (evdev capture) | Add user to `input` group |
| `/dev/uinput` | Client (uinput injection) | Add user to `input` group |

`install.sh` writes these udev rules:

```
# /etc/udev/rules.d/99-netinput.rules
KERNEL=="uinput",  GROUP="input", MODE="0660"
KERNEL=="event*",  GROUP="input", MODE="0660"
```

And adds you: `sudo usermod -aG input $USER`

**Log out and back in** after running the installer for group changes to take effect.

---

## Troubleshooting

### `no keyboard detected` / `no mouse detected`

```bash
# List all input devices
cat /proc/bus/input/devices | grep -E "Name|Handlers"

# Pass devices explicitly
./netinput-server --keyboard /dev/input/eventX --mouse /dev/input/eventY
```

### `permission denied` on `/dev/input/event*` or `/dev/uinput`

```bash
groups | grep input          # should show "input"
sudo usermod -aG input $USER # add to group
# Log out and back in
```

### Client can't connect to server

```bash
ss -tlnp | grep 24800        # verify server is listening
sudo ufw allow 24800/tcp     # open firewall port
nc -zv 192.168.1.100 24800   # test connectivity from client
```

### mDNS discovery times out

Some WiFi access points block multicast between clients. Use `--server <IP>` directly:

```bash
# Find server IP
ip addr show | grep "inet " | grep -v 127.0.0.1
```

### Cursor doesn't switch at screen edge

- Confirm `"edge_switch": true` in `config.json`
- Check `width`/`height` in `screens[]` match your actual resolution (`xrandr | grep \*`)

### Keys feel stuck after switching

A `ReleaseAll` packet is sent automatically on every switch. If keys stay stuck:

```bash
# Enable debug logging to trace packets
# Set "log_level": "debug" in config.json
```

---

## Development

```bash
# Run all tests
go test ./...

# Test specific packages
go test ./internal/protocol/...
go test ./internal/screen/...
go test ./internal/hotkey/...
go test ./config/...

# Static analysis
go vet ./...

# Build both binaries
go build -o netinput-server ./cmd/server
go build -o netinput-client ./cmd/client
```

### Implementation phases

| Phase | Status | What's included |
|---|---|---|
| 1 — MVP | ✅ Complete | Protocol, capture, inject, TCP server/client, config |
| 2 — Smart switching | ✅ Complete | Edge detection, hotkeys (Ctrl+Alt+←/→), mDNS discovery |
| 3 — Polish | 🔜 Planned | Fyne GUI tray, clipboard sync, TLS encryption, systemd service |

---

## License

MIT
