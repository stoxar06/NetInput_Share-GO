# CLAUDE.md — NetInput Share (Go)
> Token-optimized spec. Read fully before writing any code.

## PROJECT
Software KVM over WiFi for Ubuntu Linux.
One physical mouse+keyboard (server) controls up to 4 laptops (clients) on same LAN.

## LANGUAGE & RUNTIME
- Go 1.22+
- Module: `github.com/netinput/netinput-share`
- Build: `go build ./cmd/server` and `go build ./cmd/client`

## STACK
| Purpose         | Package                              |
|-----------------|--------------------------------------|
| Input capture   | `github.com/gvalkov/golang-evdev`    |
| Input inject    | `github.com/bendahl/uinput`          |
| GUI             | `fyne.io/fyne/v2`                    |
| mDNS discovery  | `github.com/grandcat/zeroconf`       |
| Config          | stdlib `encoding/json`               |
| Networking      | stdlib `net` (TCP)                   |
| Logging         | stdlib `log/slog`                    |
| CLI flags       | stdlib `flag`                        |

## DIRECTORY LAYOUT
```
netinput-share/
├── cmd/
│   ├── server/main.go       # Entry: server mode (has physical KB+mouse)
│   └── client/main.go       # Entry: client mode (receives input)
├── internal/
│   ├── capture/capture.go   # evdev: read mouse+keyboard events
│   ├── inject/inject.go     # uinput: create virtual mouse+keyboard
│   ├── protocol/protocol.go # Wire format: encode/decode events (gob)
│   ├── network/
│   │   ├── server.go        # TCP listener, broadcast to clients
│   │   └── client.go        # TCP dial, receive + apply events
│   ├── discovery/discovery.go # mDNS: advertise server, discover clients
│   ├── screen/layout.go     # 4-screen logical layout, edge detection
│   └── gui/app.go           # fyne: tray icon + config window
├── config/config.go         # Load/save config.json
├── config.json              # User config (IPs, layout, hotkeys)
├── install.sh               # Deps installer + udev rules
├── go.mod
├── go.sum
├── README.md
└── CLAUDE.md                # ← this file
```

## PROTOCOL (Wire Format)
- Transport: TCP port `24800`
- Encoding: Go `encoding/gob`
- Every packet = `Packet` struct (see protocol/protocol.go)

```go
// PacketType constants
const (
    PacketMouseMove    = 1
    PacketMouseButton  = 2
    PacketMouseScroll  = 3
    PacketKeyDown      = 4
    PacketKeyUp        = 5
    PacketSwitchScreen = 6
    PacketKeepAlive    = 7
    PacketHandshake    = 8
)

type Packet struct {
    Type      uint8
    ScreenID  uint8   // 0=server,1,2,3 = clients
    X, Y      int32   // mouse coords (absolute)
    DX, DY    int32   // mouse delta (relative)
    Button    uint16  // mouse button / key code
    Value     int32   // 1=press,0=release,2=repeat
    Timestamp int64   // UnixNano
}
```

## SCREEN LAYOUT
- 4 screens arranged in a row: [0][1][2][3] (configurable in config.json)
- Server is always Screen 0
- Edge detection: if mouse X > screenWidth → send PacketSwitchScreen to next screen
- Active screen tracks which laptop currently "owns" the cursor

## CONFIG (config.json schema)
```json
{
  "mode": "server",
  "server_port": 24800,
  "server_ip": "auto",
  "screens": [
    { "id": 0, "name": "Main",    "ip": "self",          "width": 1920, "height": 1080 },
    { "id": 1, "name": "Laptop2", "ip": "192.168.1.101", "width": 1920, "height": 1080 },
    { "id": 2, "name": "Laptop3", "ip": "192.168.1.102", "width": 1920, "height": 1080 },
    { "id": 3, "name": "Laptop4", "ip": "192.168.1.103", "width": 1920, "height": 1080 }
  ],
  "layout": "horizontal",
  "hotkey_switch": "ctrl+alt+right",
  "hotkey_back":   "ctrl+alt+left",
  "edge_switch": true,
  "clipboard_sync": true,
  "log_level": "info"
}
```

## IMPLEMENTATION PHASES

### Phase 1 — MVP (do first)
- [ ] `go.mod` with all dependencies
- [ ] `protocol/protocol.go` — Packet struct + gob encode/decode
- [ ] `capture/capture.go` — open evdev devices, emit events to channel
- [ ] `inject/inject.go` — create uinput virtual mouse + keyboard
- [ ] `network/server.go` — TCP listen, manage client connections, broadcast
- [ ] `network/client.go` — TCP dial server, receive packets, inject events
- [ ] `config/config.go` — load/save/validate config.json
- [ ] `cmd/server/main.go` — wire everything for server mode
- [ ] `cmd/client/main.go` — wire everything for client mode

### Phase 2 — Smart Switching
- [ ] `screen/layout.go` — edge detection, screen topology
- [ ] `discovery/discovery.go` — mDNS advertise + browse
- [ ] Hotkey switch support in server
- [ ] Multi-client fan-out (4 laptops)

### Phase 3 — Polish
- [ ] `gui/app.go` — fyne system tray + config UI
- [ ] Clipboard sync (send text clipboard over TCP)
- [ ] TLS encryption option
- [ ] `install.sh` — install deps, udev rules, systemd service

## KEY IMPLEMENTATION NOTES

### Input Capture (Server)
```go
// Must run as root or be in 'input' group
// Devices at: /dev/input/event*
// Use evdev.Open("/dev/input/eventX")
// Filter: EV_KEY (keyboard), EV_REL (mouse rel), EV_ABS (mouse abs), EV_SYN
// Run capture loop in goroutine, send to chan protocol.Packet
```

### Input Injection (Client)
```go
// Requires /dev/uinput access (udev rule or root)
// Create virtual keyboard: uinput.CreateKeyboard()
// Create virtual mouse: uinput.CreateMouse()
// Translate Packet → uinput calls
```

### Network Server Fan-out
```go
// One goroutine per client connection
// Use sync.RWMutex for client map
// Only forward packets to the currently active screen's client
// Keep-alive every 1s, drop dead connections
```

### Screen Switching Logic
```go
// Server tracks: activeScreenID int
// On mouse move: if X >= screenWidth[activeScreenID] → switch to next
// On switch: send PacketSwitchScreen to new client, send "release all" to old
// Hotkey: Ctrl+Alt+→ / Ctrl+Alt+← to force switch
```

### evdev Device Detection
```go
// Auto-detect: scan /dev/input/event* for keyboard+mouse
// Keyboard: has EV_KEY capability with KEY_A
// Mouse: has EV_REL capability with REL_X, REL_Y
// List and let user pick if multiple found
```

## COMMANDS TO RUN APP
```bash
# Server laptop (has physical mouse+keyboard)
sudo ./server --config config.json

# Client laptops (controlled remotely)  
sudo ./client --server 192.168.1.100 --id 1

# Auto-discover server on LAN
sudo ./client --discover --id 1
```

## PERMISSIONS SETUP (udev rules)
```
# /etc/udev/rules.d/99-netinput.rules
KERNEL=="uinput", GROUP="input", MODE="0660"
KERNEL=="event*", GROUP="input", MODE="0660"
```
Add user to input group: `sudo usermod -aG input $USER`

## ERROR HANDLING RULES
- All errors wrapped with context: `fmt.Errorf("capture: %w", err)`
- Network errors → retry with exponential backoff (max 30s)
- Lost client → remove from map, log, continue serving others
- evdev device disconnect → attempt re-open every 5s
- Config parse error → print helpful message, show defaults

## TESTING
```bash
go test ./...                    # all unit tests
go test ./internal/protocol/...  # protocol encode/decode
go test ./internal/screen/...    # layout edge detection
go vet ./...                     # static analysis
go build ./cmd/server ./cmd/client  # build check
```

## DO NOT
- Do NOT use CGO unless required by evdev/uinput (they need it)
- Do NOT use goroutine leaks — always use context.Context for cancellation
- Do NOT hardcode IPs — always read from config
- Do NOT skip error handling
- Do NOT use global mutable state — pass dependencies explicitly

## CURRENT STATUS
> All files are scaffolded with TODOs. Start with Phase 1 in order.
> Begin: `internal/protocol/protocol.go` → then `capture` → `inject` → `network`
