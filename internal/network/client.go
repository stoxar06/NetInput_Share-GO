package network

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/netinput/netinput-share/internal/protocol"
)

const (
	reconnectBaseDelay = time.Second
	reconnectMaxDelay  = 30 * time.Second
)

// Client connects to the server and forwards packets to the injector.
type Client struct {
	serverAddr string
	screenID   uint8
	name       string
	width      int32
	height     int32
	out        chan<- protocol.Packet
}

// NewClient creates a Client.
func NewClient(serverAddr string, screenID uint8, name string, width, height int32, out chan<- protocol.Packet) *Client {
	return &Client{
		serverAddr: serverAddr,
		screenID:   screenID,
		name:       name,
		width:      width,
		height:     height,
		out:        out,
	}
}

// Run connects to server and receives packets. Reconnects on disconnect.
func (c *Client) Run(ctx context.Context) error {
	delay := reconnectBaseDelay
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := c.connect(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Warn("client: connection lost, retrying", "err", err, "delay", delay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			if delay < reconnectMaxDelay {
				delay *= 2
				if delay > reconnectMaxDelay {
					delay = reconnectMaxDelay
				}
			}
			continue
		}
		delay = reconnectBaseDelay
	}
}

func (c *Client) connect(ctx context.Context) error {
	conn, err := net.DialTimeout("tcp", c.serverAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("client dial: %w", err)
	}
	defer conn.Close()
	if tc, ok := conn.(*net.TCPConn); ok {
		tc.SetNoDelay(true)
	}
	slog.Info("client: connected to server", "addr", c.serverAddr)

	// Build and send handshake.
	payload := protocol.HandshakePayload{
		ScreenID: c.screenID,
		Name:     c.name,
		Width:    c.width,
		Height:   c.height,
		Version:  "1.0",
	}
	var payloadBuf bytes.Buffer
	if err := gob.NewEncoder(&payloadBuf).Encode(payload); err != nil {
		return fmt.Errorf("client handshake encode: %w", err)
	}

	hsPkt := protocol.Packet{
		Type: protocol.PacketHandshake,
		Data: payloadBuf.Bytes(),
	}
	hsData, err := protocol.Encode(hsPkt)
	if err != nil {
		return fmt.Errorf("client handshake packet: %w", err)
	}

	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(hsData)))
	if _, err := conn.Write(append(length, hsData...)); err != nil {
		return fmt.Errorf("client handshake send: %w", err)
	}

	// Receive packets from server.
	for {
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		pkt, err := readPacket(conn)
		if err != nil {
			var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					continue
				}
			}
			return fmt.Errorf("client receive: %w", err)
		}
		select {
		case c.out <- pkt:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
