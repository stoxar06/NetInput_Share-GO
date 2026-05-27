#!/bin/bash
# NetInput Share — Installer for Ubuntu
# Run once on ALL laptops (server + clients)
set -e

echo "=== NetInput Share Installer ==="
echo ""

# 1. Install Go (if not installed)
if ! command -v go &>/dev/null; then
  echo "[1/5] Installing Go..."
  sudo apt-get update -q
  sudo apt-get install -y golang-go
else
  echo "[1/5] Go already installed: $(go version)"
fi

# 2. Install system dependencies
echo "[2/5] Installing system dependencies..."
sudo apt-get install -y \
  libx11-dev \
  libxrandr-dev \
  libxinerama-dev \
  libxcursor-dev \
  libxi-dev \
  libgl1-mesa-dev \
  libegl1-mesa-dev \
  libdbus-1-dev \
  pkg-config

# 3. Setup udev rules (allow /dev/uinput and /dev/input access without root)
echo "[3/5] Setting up udev rules..."
sudo tee /etc/udev/rules.d/99-netinput.rules > /dev/null <<'EOF'
# NetInput Share — allow input group access
KERNEL=="uinput",  GROUP="input", MODE="0660"
KERNEL=="event*",  GROUP="input", MODE="0660"
EOF
sudo udevadm control --reload-rules
sudo udevadm trigger

# 4. Add current user to 'input' group
echo "[4/5] Adding $USER to 'input' group..."
sudo usermod -aG input "$USER"
echo "  ⚠  You must log out and log back in for group changes to take effect."

# 5. Build binaries
echo "[5/5] Building NetInput Share..."
go mod tidy
go build -o netinput-server ./cmd/server
go build -o netinput-client ./cmd/client
echo ""
echo "✅ Build complete!"
echo ""
echo "=== HOW TO USE ==="
echo ""
echo "On the SERVER laptop (has physical mouse+keyboard):"
echo "  ./netinput-server --config config.json"
echo ""
echo "On each CLIENT laptop (controlled remotely):"
echo "  ./netinput-client --server <SERVER_IP> --id 1   # Laptop 2"
echo "  ./netinput-client --server <SERVER_IP> --id 2   # Laptop 3"
echo "  ./netinput-client --server <SERVER_IP> --id 3   # Laptop 4"
echo ""
echo "Or use auto-discovery:"
echo "  ./netinput-client --discover --id 1"
echo ""
echo "Edit config.json to set your screen IPs and layout."
echo ""
echo "Hotkeys:"
echo "  Ctrl+Alt+Right  → switch to next screen"
echo "  Ctrl+Alt+Left   → switch to previous screen"
echo ""
echo "=== DONE ==="
