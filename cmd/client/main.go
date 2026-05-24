// NetInput Share — Client Mode
// Run this on each of the remote laptops that will be controlled.
// Usage: sudo ./client --server 192.168.1.100 --id 1
//        sudo ./client --discover --id 1
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/netinput/netinput-share/config"
	"github.com/netinput/netinput-share/internal/discovery"
	"github.com/netinput/netinput-share/internal/inject"
	"github.com/netinput/netinput-share/internal/network"
	"github.com/netinput/netinput-share/internal/protocol"
)

func main() {
	configPath := flag.String("config", "config.json", "path to config.json")
	serverAddr := flag.String("server", "", "server IP (e.g. 192.168.1.100)")
	screenID := flag.Int("id", 1, "screen ID for this client (1, 2, or 3)")
	doDiscover := flag.Bool("discover", false, "auto-discover server via mDNS")
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

	if *screenID < 1 || *screenID > 3 {
		slog.Error("--id must be 1, 2, or 3 for client mode")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	addr := fmt.Sprintf("%s:%d", *serverAddr, cfg.ServerPort)
	if *doDiscover {
		slog.Info("client: discovering server on LAN...")
		found, err := discovery.DiscoverServer(ctx, 15*time.Second)
		if err != nil {
			slog.Error("discovery failed", "err", err)
			os.Exit(1)
		}
		addr = found
	}

	if *serverAddr == "" && !*doDiscover {
		slog.Error("provide --server IP or use --discover")
		os.Exit(1)
	}

	var width, height int32 = 1920, 1080
	name := fmt.Sprintf("Laptop%d", *screenID)
	for _, s := range cfg.Screens {
		if int(s.ID) == *screenID {
			width = int32(s.Width)
			height = int32(s.Height)
			name = s.Name
			break
		}
	}

	slog.Info("NetInput Share — Client Mode", "server", addr, "screenID", *screenID, "name", name)

	packets := make(chan protocol.Packet, 256)
	netClient := network.NewClient(addr, uint8(*screenID), name, width, height, packets)
	injector := inject.New(packets)

	go func() {
		if err := injector.Start(ctx); err != nil {
			slog.Error("injector error", "err", err)
			cancel()
		}
	}()

	if err := netClient.Run(ctx); err != nil {
		slog.Info("client stopped", "reason", err)
	}
}
