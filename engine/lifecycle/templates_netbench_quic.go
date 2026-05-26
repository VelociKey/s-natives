package lifecycle

const QuicGo = `package netbench

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/quic-go/quic-go"
	"sov.fleet/s-logiclibrary/00200-logic-libraries/gatekeeper"
)

type QUICServer struct {
	ctx      context.Context
	listener *quic.Listener
	gk       *gatekeeper.Gatekeeper
	secret   []byte
	maxAge   time.Duration
	Format   MsgFormat
	done     chan struct{}
}

func StartQUICServer(ctx context.Context, format MsgFormat, secret []byte, tlsConf *tls.Config) (*QUICServer, error) {
	if ctx == nil {
		return nil, errors.New("pointer safety error: context is nil")
	}
	if tlsConf == nil {
		return nil, errors.New("pointer safety error: TLS config is nil")
	}
	quicConf := &quic.Config{
		MaxIncomingStreams: 1000000,
	}
	l, err := quic.ListenAddr("127.0.0.1:0", tlsConf, quicConf)
	if err != nil {
		slog.Error("Failed to start QUIC listener", "err", err)
		return nil, err
	}
	s := &QUICServer{
		ctx:      ctx,
		listener: l,
		gk:       gatekeeper.NewGatekeeper(),
		secret:   secret,
		maxAge:   10 * time.Second,
		Format:   format,
		done:     make(chan struct{}),
	}
	slog.Info("QUIC Server started", "addr", s.Addr(), "format", int(format))
	
	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()
	
	go s.serve()
	return s, nil
}

func (s *QUICServer) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *QUICServer) Close() error {
	if s == nil || s.listener == nil {
		return nil
	}
	slog.Info("Closing QUIC Server", "addr", s.Addr())
	err := s.listener.Close()
	select {
	case <-s.done:
	case <-time.After(1 * time.Second):
	}
	return err
}

func (s *QUICServer) serve() {
	if s == nil || s.listener == nil {
		return
	}
	defer close(s.done)
	for {
		conn, err := s.listener.Accept(s.ctx)
		if err != nil {
			slog.Debug("QUIC listener stopped or closed")
			return
		}
		go s.handleConnection(conn)
	}
}

func (s *QUICServer) handleConnection(conn *quic.Conn) {
	if s == nil || conn == nil {
		return
	}
	defer conn.CloseWithError(0, "done")
	slog.Debug("QUIC connection accepted", "remote", conn.RemoteAddr())
	for {
		stream, err := conn.AcceptStream(s.ctx)
		if err != nil {
			slog.Debug("QUIC connection closed or stream accept aborted", "remote", conn.RemoteAddr(), "err", err)
			return
		}
		go s.handleStream(stream)
	}
}

func (s *QUICServer) handleStream(stream *quic.Stream) {
	if s == nil || stream == nil {
		return
	}
	defer func() {
		if err := stream.Close(); err != nil {
			slog.Debug("failed to close stream", "err", err)
		}
	}()
	buf := make([]byte, 8192)

	var lenBuf [4]byte
	_, err := io.ReadFull(stream, lenBuf[:])
	if err != nil {
		slog.Error("Failed to read QUIC stream message length", "err", err)
		return
	}
	length := binary.BigEndian.Uint32(lenBuf[:])
	if length > uint32(len(buf)) {
		buf = make([]byte, length)
	}
	_, err = io.ReadFull(stream, buf[:length])
	if err != nil {
		slog.Error("Failed to read QUIC stream payload", "length", length, "err", err)
		return
	}
	
	if _, err := io.Copy(io.Discard, stream); err != nil {
		slog.Debug("failed to discard stream data", "err", err)
	}

	var valid bool
	switch s.Format {
	case weBNF:
		_, err = DeserializeText(buf[:length])
		valid = err == nil
		if err != nil {
			slog.Warn("QUIC text validation failed", "err", err)
		}
	case SACPUnsecured:
		_, err = DeserializeSACPUnsecured(buf[:length])
		valid = err == nil
		if err != nil {
			slog.Warn("QUIC SACP unsecured validation failed", "err", err)
		}
	case SACPSecured:
		_, err = DeserializeSACPSecured(buf[:length], s.gk, s.secret, s.maxAge)
		valid = err == nil
		if err != nil {
			slog.Warn("QUIC SACP secured validation failed", "err", err)
		}
	}

	var ack [1]byte
	if valid {
		ack[0] = 0x01
	} else {
		ack[0] = 0x00
	}
	_, err = stream.Write(ack[:])
	if err != nil {
		slog.Error("Failed to write QUIC stream ack", "err", err)
	}
}

func RunQUICCall(stream *quic.Stream, payload []byte) (bool, error) {
	if stream == nil {
		return false, errors.New("pointer safety error: QUIC stream is nil")
	}
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(payload)))
	_, err := stream.Write(lenBuf[:])
	if err != nil {
		return false, err
	}
	_, err = stream.Write(payload)
	if err != nil {
		return false, err
	}
	if err := stream.Close(); err != nil {
		slog.Debug("failed to close write stream", "err", err)
	}

	var ack [1]byte
	_, err = io.ReadFull(stream, ack[:])
	if err != nil {
		return false, err
	}
	
	if _, err := io.Copy(io.Discard, stream); err != nil {
		slog.Debug("failed to discard server stream data", "err", err)
	}

	return ack[0] == 0x01, nil
}
`
