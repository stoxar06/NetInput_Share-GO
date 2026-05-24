// Package discovery handles mDNS-based LAN service advertisement and discovery.
package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	serviceType   = "_netinput._tcp"
	serviceDomain = "local."
	servicePort   = 24800
)

// Advertise announces this node as a NetInput server on the LAN.
// Blocks until ctx is cancelled.
func Advertise(ctx context.Context, name string) error {
	server, err := zeroconf.Register(
		name,
		serviceType,
		serviceDomain,
		servicePort,
		[]string{"version=1.0"},
		nil,
	)
	if err != nil {
		return fmt.Errorf("discovery advertise: %w", err)
	}
	defer server.Shutdown()
	slog.Info("discovery: advertising as NetInput server", "name", name)
	<-ctx.Done()
	return ctx.Err()
}

// DiscoverServer browses the LAN for a NetInput server and returns its address.
// Times out after timeout duration if no server found.
func DiscoverServer(ctx context.Context, timeout time.Duration) (string, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return "", fmt.Errorf("discovery resolver: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry, 4)
	browseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := resolver.Browse(browseCtx, serviceType, serviceDomain, entries); err != nil {
		return "", fmt.Errorf("discovery browse: %w", err)
	}

	slog.Info("discovery: scanning LAN for NetInput server...")

	for {
		select {
		case <-browseCtx.Done():
			return "", fmt.Errorf("discovery: no server found within %v", timeout)
		case entry, ok := <-entries:
			if !ok {
				return "", fmt.Errorf("discovery: browse channel closed")
			}
			if len(entry.AddrIPv4) == 0 {
				continue
			}
			addr := fmt.Sprintf("%s:%d", entry.AddrIPv4[0], entry.Port)
			slog.Info("discovery: found server", "name", entry.Instance, "addr", addr)
			return addr, nil
		}
	}
}
