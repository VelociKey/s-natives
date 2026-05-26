package lifecycle

const UdpGo = `package netbench

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"sov.fleet/s-logiclibrary/00200-logic-libraries/gatekeeper"
)

type UDPServer struct {
	ctx    context.Context
	conn   net.PacketConn
	gk     *gatekeeper.Gatekeeper
	secret []byte
	maxAge time.Duration
	Format MsgFormat
	done   chan struct{}
}

func StartUDPServer(ctx context.Context, format MsgFormat, secret []byte) (*UDPServer, error) {
	if ctx == nil {
		return nil, errors.New("pointer safety error: context is nil")
	}
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		slog.Error("Failed to start UDP listener", "err", err)
		return nil, err
	}
	s := &UDPServer{
		ctx:    ctx,
		conn:   c,
		gk:     gatekeeper.NewGatekeeper(),
		secret: secret,
		maxAge: 10 * time.Second,
		Format: format,
		done:   make(chan struct{}),
	}
	slog.Info("UDP Server started", "addr", s.Addr(), "format", int(format))
	
	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()
	
	go s.serve()
	return s, nil
}

func (s *UDPServer) Addr() string {
	if s == nil || s.conn == nil {
		return ""
	}
	return s.conn.LocalAddr().String()
}

func (s *UDPServer) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	slog.Info("Closing UDP Server", "addr", s.Addr())
	err := s.conn.Close()
	select {
	case <-s.done:
	case <-time.After(1 * time.Second):
	}
	return err
}

func (s *UDPServer) serve() {
	if s == nil || s.conn == nil {
		return
	}
	defer close(s.done)
	buf := make([]byte, 8192)
	for {
		n, addr, err := s.conn.ReadFrom(buf)
		if err != nil {
			slog.Debug("UDP listener socket closed")
			return
		}

		var valid bool
		switch s.Format {
		case weBNF:
			_, err = DeserializeText(buf[:n])
			valid = err == nil
			if err != nil {
				slog.Warn("UDP text validation failed", "addr", addr, "err", err)
			}
		case SACPUnsecured:
			_, err = DeserializeSACPUnsecured(buf[:n])
			valid = err == nil
			if err != nil {
				slog.Warn("UDP SACP unsecured validation failed", "addr", addr, "err", err)
			}
		case SACPSecured:
			_, err = DeserializeSACPSecured(buf[:n], s.gk, s.secret, s.maxAge)
			valid = err == nil
			if err != nil {
				slog.Warn("UDP SACP secured validation failed", "addr", addr, "err", err)
			}
		}

		var ack [1]byte
		if valid {
			ack[0] = 0x01
		} else {
			ack[0] = 0x00
		}
		_, err = s.conn.WriteTo(ack[:], addr)
		if err != nil {
			slog.Error("Failed to write UDP ack", "addr", addr, "err", err)
		}
	}
}

func RunUDPCall(conn net.PacketConn, serverAddr net.Addr, payload []byte) (bool, error) {
	if conn == nil || serverAddr == nil {
		return false, errors.New("pointer safety error: UDP connection or server address is nil")
	}
	_, err := conn.WriteTo(payload, serverAddr)
	if err != nil {
		return false, err
	}
	var ack [1]byte
	_, _, err = conn.ReadFrom(ack[:])
	if err != nil {
		return false, err
	}
	return ack[0] == 0x01, nil
}
`
