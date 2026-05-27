// Package network handles TCP communication between server and client laptops.
package network

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/netinput/netinput-share/internal/protocol"
)

const keepAliveInterval = time.Second

// ClientConn represents a connected client laptop.
type ClientConn struct {
	ScreenID uint8
	Name     string
	conn     net.Conn
	mu       sync.Mutex
}

// Send encodes and writes a packet to this client.
func (c *ClientConn) Send(pkt protocol.Packet) error {
	data, err := protocol.Encode(pkt)
	if err != nil {
		return fmt.Errorf("clientconn send encode: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(data)))
	if _, err := c.conn.Write(append(length, data...)); err != nil {
		return fmt.Errorf("clientconn send write: %w", err)
	}
	return nil
}

// Server listens for client connections and routes input packets.
type Server struct {
	port     int
	in       <-chan protocol.Packet
	clients  map[uint8]*ClientConn
	mu       sync.RWMutex
	activeID uint8
}

// NewServer creates a Server. in receives captured input packets.
func NewServer(port int, in <-chan protocol.Packet) *Server {
	return &Server{
		port:    port,
		in:      in,
		clients: make(map[uint8]*ClientConn),
	}
}

// Run starts the TCP listener and packet routing loop.
func (s *Server) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("server listen: %w", err)
	}
	defer ln.Close()
	slog.Info("server: listening", "addr", addr)

	go s.acceptLoop(ctx, ln)
	go s.keepAliveLoop(ctx)

	// Close listener when ctx is cancelled to unblock Accept.
	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	return s.routeLoop(ctx)
}

func (s *Server) acceptLoop(ctx context.Context, ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				slog.Warn("server: accept error", "err", err)
				continue
			}
		}
		go s.handleClient(ctx, conn)
	}
}

func (s *Server) handleClient(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	// Read handshake with a generous timeout.
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	pkt, err := readPacket(conn)
	if err != nil {
		slog.Warn("server: handshake read error", "remote", conn.RemoteAddr(), "err", err)
		return
	}
	if pkt.Type != protocol.PacketHandshake {
		slog.Warn("server: expected handshake", "got", pkt.Type)
		return
	}

	var payload protocol.HandshakePayload
	if err := gob.NewDecoder(bytes.NewReader(pkt.Data)).Decode(&payload); err != nil {
		slog.Warn("server: handshake decode error", "err", err)
		return
	}
	conn.SetDeadline(time.Time{}) // clear deadline

	client := &ClientConn{
		ScreenID: payload.ScreenID,
		Name:     payload.Name,
		conn:     conn,
	}

	s.mu.Lock()
	s.clients[payload.ScreenID] = client
	s.mu.Unlock()

	slog.Info("server: client registered",
		"screenID", payload.ScreenID,
		"name", payload.Name,
		"remote", conn.RemoteAddr(),
	)

	defer func() {
		s.mu.Lock()
		delete(s.clients, payload.ScreenID)
		s.mu.Unlock()
		slog.Info("server: client disconnected", "screenID", payload.ScreenID, "name", payload.Name)
	}()

	// Drain any packets the client might send (none in Phase 1), detect disconnect.
	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, err := readPacket(conn)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			return
		}
	}
}

func (s *Server) routeLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case pkt, ok := <-s.in:
			if !ok {
				return fmt.Errorf("server: input channel closed")
			}
			s.mu.RLock()
			client, ok := s.clients[s.activeID]
			s.mu.RUnlock()
			if !ok {
				continue // active screen is local server
			}
			if err := client.Send(pkt); err != nil {
				slog.Warn("server: send error", "screenID", s.activeID, "err", err)
				s.mu.Lock()
				delete(s.clients, s.activeID)
				s.mu.Unlock()
			}
		}
	}
}

func (s *Server) keepAliveLoop(ctx context.Context) {
	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pkt := protocol.Packet{Type: protocol.PacketKeepAlive, Timestamp: time.Now().UnixNano()}
			s.mu.RLock()
			for id, c := range s.clients {
				if err := c.Send(pkt); err != nil {
					slog.Warn("server: keepalive failed", "screenID", id, "err", err)
				}
			}
			s.mu.RUnlock()
		}
	}
}

// SwitchTo switches active screen to the given screenID.
func (s *Server) SwitchTo(screenID uint8) {
	s.mu.Lock()
	old := s.activeID
	s.activeID = screenID
	var oldClient *ClientConn
	if old != screenID {
		oldClient = s.clients[old]
	}
	s.mu.Unlock()

	if oldClient != nil {
		_ = oldClient.Send(protocol.Packet{Type: protocol.PacketReleaseAll})
		slog.Info("server: switched screen", "from", old, "to", screenID)
	}
}

// ActiveID returns the currently active screen ID.
func (s *Server) ActiveID() uint8 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeID
}

// readPacket reads a length-prefixed packet from conn.
func readPacket(conn net.Conn) (protocol.Packet, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return protocol.Packet{}, fmt.Errorf("readPacket length: %w", err)
	}
	length := binary.BigEndian.Uint32(lenBuf)
	if length > 1<<20 {
		return protocol.Packet{}, fmt.Errorf("readPacket: packet too large (%d bytes)", length)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return protocol.Packet{}, fmt.Errorf("readPacket data: %w", err)
	}
	return protocol.Decode(data)
}
