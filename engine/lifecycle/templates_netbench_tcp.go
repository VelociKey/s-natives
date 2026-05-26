package lifecycle

const TcpGo = `package netbench

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"net"
	"time"

	"sov.fleet/s-logiclibrary/00200-logic-libraries/gatekeeper"
)

type TCPServer struct {
	ctx      context.Context
	listener net.Listener
	gk       *gatekeeper.Gatekeeper
	secret   []byte
	maxAge   time.Duration
	Format   MsgFormat
	done     chan struct{}
}

func StartTCPServer(ctx context.Context, format MsgFormat, secret []byte) (*TCPServer, error) {
	if ctx == nil {
		return nil, errors.New("pointer safety error: context is nil")
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		slog.Error("Failed to start TCP listener", "err", err)
		return nil, err
	}
	s := &TCPServer{
		ctx:      ctx,
		listener: l,
		gk:       gatekeeper.NewGatekeeper(),
		secret:   secret,
		maxAge:   10 * time.Second,
		Format:   format,
		done:     make(chan struct{}),
	}
	slog.Info("TCP Server started", "addr", s.Addr(), "format", int(format))
	
	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()
	
	go s.serve()
	return s, nil
}

func (s *TCPServer) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *TCPServer) Close() error {
	if s == nil || s.listener == nil {
		return nil
	}
	slog.Info("Closing TCP Server", "addr", s.Addr())
	err := s.listener.Close()
	select {
	case <-s.done:
	case <-time.After(1 * time.Second):
	}
	return err
}

func (s *TCPServer) serve() {
	if s == nil || s.listener == nil {
		return
	}
	defer close(s.done)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			slog.Debug("TCP listener stopped or closed")
			return
		}
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	if s == nil || conn == nil {
		return
	}
	defer conn.Close()

	go func() {
		<-s.ctx.Done()
		conn.Close()
	}()

	slog.Debug("TCP connection accepted", "remote", conn.RemoteAddr())
	buf := make([]byte, 8192)
	for {
		var lenBuf [4]byte
		_, err := io.ReadFull(conn, lenBuf[:])
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed) {
				slog.Debug("TCP client disconnected", "remote", conn.RemoteAddr())
			} else {
				slog.Error("Failed to read TCP message length", "remote", conn.RemoteAddr(), "err", err)
			}
			return
		}
		length := binary.BigEndian.Uint32(lenBuf[:])
		if length > uint32(len(buf)) {
			buf = make([]byte, length)
		}
		_, err = io.ReadFull(conn, buf[:length])
		if err != nil {
			slog.Error("Failed to read TCP message payload", "remote", conn.RemoteAddr(), "length", length, "err", err)
			return
		}

		var valid bool
		switch s.Format {
		case weBNF:
			_, err = DeserializeText(buf[:length])
			valid = err == nil
			if err != nil {
				slog.Warn("TCP text validation failed", "err", err)
			}
		case SACPUnsecured:
			_, err = DeserializeSACPUnsecured(buf[:length])
			valid = err == nil
			if err != nil {
				slog.Warn("TCP SACP unsecured validation failed", "err", err)
			}
		case SACPSecured:
			_, err = DeserializeSACPSecured(buf[:length], s.gk, s.secret, s.maxAge)
			valid = err == nil
			if err != nil {
				slog.Warn("TCP SACP secured validation failed", "err", err)
			}
		}

		var ack [1]byte
		if valid {
			ack[0] = 0x01
		} else {
			ack[0] = 0x00
		}
		_, err = conn.Write(ack[:])
		if err != nil {
			slog.Error("Failed to write TCP ack", "remote", conn.RemoteAddr(), "err", err)
			return
		}
	}
}

func RunTCPCall(conn net.Conn, payload []byte) (bool, error) {
	if conn == nil {
		return false, errors.New("pointer safety error: TCP connection is nil")
	}
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(payload)))
	_, err := conn.Write(lenBuf[:])
	if err != nil {
		return false, err
	}
	_, err = conn.Write(payload)
	if err != nil {
		return false, err
	}
	var ack [1]byte
	_, err = io.ReadFull(conn, ack[:])
	if err != nil {
		return false, err
	}
	return ack[0] == 0x01, nil
}
`
