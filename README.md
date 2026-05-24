# NetInput Share

Share one mouse and keyboard across up to 4 Ubuntu laptops on the same WiFi network.
Move your cursor to the screen edge — control automatically switches to the next laptop.

Built in **Go** for maximum performance and minimal latency.

---

## How It Works

```
[Server Laptop]                         [Client Laptop 1]
  Physical Mouse ──────────────────────► Virtual Mouse
  Physical Keyboard   WiFi (TCP 24800)  Virtual Keyboard
  netinput-server                        netinput-client --id 1

                                        [Client Laptop 2]
                                        ► netinput-client --id 2

                                        [Client Laptop 3]
                                        ► netinput-client --id 3
```

Screen layout (horizontal):
```
┌──────────┬──────────┬──────────┬──────────┐
│ Screen 0 │ Screen 1 │ Screen 2 │ Screen 3 │
│ (Server) │ Laptop 2 │ Laptop 3 │ Laptop 4 │
└──────────┴──────────┴──────────┴──────────┘
```

---

## Requirements

- Ubuntu 20.04 or later
- Go 1.22+
- All laptops on same WiFi network
- User in `input` group (installer handles this)

---

## Quick Start

### 1. Clone & Install (run on ALL laptops)
```bash
git clone https://github.com/netinput/netinput-share
cd netinput-share
chmod +x install.sh
./install.sh
```

Log out and log back in after install (for group permissions).

### 2. Edit Config
Edit `config.json` — set the IP addresses of your 3 client laptops:
```json
"screens": [
  { "id": 0, "name": "Main",    "ip": "self",          "width": 1920, "height": 1080 },
  { "id": 1, "name": "Laptop2", "ip": "192.168.1.101", "width": 1920, "height": 1080 },
  { "id": 2, "name": "Laptop3", "ip": "192.168.1.102", "width": 1920, "height": 1080 },
  { "id": 3, "name": "Laptop4", "ip": "192.168.1.103", "width": 1920, "height": 1080 }
]
```

Find a laptop's IP with: `hostname -I`

### 3. Run Server (laptop with physical mouse+keyboard)
```bash
./netinput-server --config config.json
```

### 4. Run Client (on each of the other 3 laptops)
```bash
# Laptop 2
./netinput-client --server 192.168.1.100 --id 1

# Laptop 3
./netinput-client --server 192.168.1.100 --id 2

# Laptop 4
./netinput-client --server 192.168.1.100 --id 3
```

Or use auto-discovery (no IP needed):
```bash
./netinput-client --discover --id 1
```

---

## Hotkeys

| Hotkey | Action |
|--------|--------|
| `Ctrl+Alt+→` | Switch to next screen |
| `Ctrl+Alt+←` | Switch to previous screen |

Or just move your mouse to the screen edge — it switches automatically!

---

## Project Structure

```
netinput-share/
├── cmd/server/main.go       # Server entry point
├── cmd/client/main.go       # Client entry point
├── internal/
│   ├── capture/             # Read physical mouse+keyboard (evdev)
│   ├── inject/              # Create virtual mouse+keyboard (uinput)
│   ├── protocol/            # Wire format (gob over TCP)
│   ├── network/             # TCP server & client
│   ├── discovery/           # mDNS LAN auto-discovery
│   ├── screen/              # 4-screen layout & edge detection
│   └── gui/                 # Fyne system tray UI
├── config/                  # Config loading
├── config.json              # User configuration
├── install.sh               # Installer script
├── CLAUDE.md                # AI-optimized project spec
└── README.md
```

---

## Development Phases

- **Phase 1** — Core MVP: capture → network → inject (keyboard + mouse)
- **Phase 2** — Smart switching: screen edges, hotkeys, multi-client (4 laptops)
- **Phase 3** — Polish: GUI, clipboard sync, TLS encryption

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.22 |
| Input capture | `golang-evdev` |
| Input injection | `bendahl/uinput` |
| Networking | stdlib TCP |
| GUI | Fyne v2 |
| LAN discovery | `grandcat/zeroconf` (mDNS) |
| Config | JSON (stdlib) |

---

## Troubleshooting

**Permission denied on /dev/input or /dev/uinput**
```bash
sudo usermod -aG input $USER
# Log out and back in
```

**Can't connect to server**
```bash
# Check firewall — open port 24800
sudo ufw allow 24800/tcp
# Verify same WiFi network
ping <server-ip>
```

**Cursor doesn't switch at screen edge**
- Make sure `"edge_switch": true` in config.json
- Check screen width/height values match your actual resolution

---

## License
MIT
