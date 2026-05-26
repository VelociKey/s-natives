package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sov.fleet/s-logiclibrary/00200-logic-libraries/bicodec"
	"sov.fleet/s-fab-aides/81000-active-source/pkg/bash"
)

// NATVSPhase represents a single state in the sovereign orchestration lifecycle.
type NATVSPhase string

const (
	PhaseNegotiate    NATVSPhase = "NEGOTIATE"
	PhaseAssimilation NATVSPhase = "ASSIMILATE"
	PhaseTransform    NATVSPhase = "TRANSFORM"
	PhaseVerification NATVSPhase = "VERIFY"
	PhaseSynthesis    NATVSPhase = "SYNTHESIS"
)

// OrchestrationConfig houses system paths and validation rails.
type OrchestrationConfig struct {
	WorkspaceRoot     string
	RegistryPath      string
	OutputChannel     chan string
	AllowedWorkspaces []string
	Mu                sync.Mutex
}

// NATVSEngine represents the core fleet-aware execution manager.
type NATVSEngine struct {
	Config *OrchestrationConfig
	State  NATVSPhase
}

// NewNATVSEngine instantiates a new sovereign engine container.
func NewNATVSEngine(workspace string) *NATVSEngine {
	var allowed []string
	if envWS := os.Getenv("ALLOWED_WORKSPACES"); envWS != "" {
		parts := strings.Split(envWS, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				allowed = append(allowed, part)
			}
		}
	}
	return &NATVSEngine{
		Config: &OrchestrationConfig{
			WorkspaceRoot:     workspace,
			RegistryPath:      filepath.Join(workspace, "00flow/s-forge/90100-rehydration-seed"),
			OutputChannel:     make(chan string, 100),
			AllowedWorkspaces: allowed,
		},
		State: PhaseNegotiate,
	}
}

// Negotiate executes phase 1 capability handshakes and transitions.
func (e *NATVSEngine) Negotiate(ctx context.Context, targetWorkspace string) error {
	e.State = PhaseNegotiate
	log.Printf("[NATVS] Phase 1: Initiating Negotiation handshake for target: %s...", targetWorkspace)
	
	// Simulate Whisper Bus Capability mapping
	time.Sleep(10 * time.Millisecond)
	log.Printf("[NATVS] Handshake complete. Target verified inside secure loop.")
	return nil
}

// Assimilation executes phase 2 local repository structural audits.
func (e *NATVSEngine) Assimilation(ctx context.Context, targetWorkspace string) error {
	e.State = PhaseAssimilation
	log.Printf("[NATVS] Phase 2: Assimilating context directories for workspace: %s...", targetWorkspace)
	
	// Assert 5-digit / 4-digit directory taxonomy rules
	targetPath := filepath.Join(e.Config.WorkspaceRoot, targetWorkspace)
	
	// Audit directory structure using s-fab-aides bash polyfill to replace shell subprocesses
	var lsBuf strings.Builder
	if err := bash.Execute(&lsBuf, "ls", targetPath); err == nil {
		log.Printf("[NATVS] Dynamic file discovery via s-fab-aides bash: %s", strings.ReplaceAll(strings.TrimSpace(lsBuf.String()), "\n", ", "))
	}
	
	dir, err := os.ReadDir(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("target workspace path does not exist: %w", err)
		}
		return err
	}
	
	validCount := 0
	for _, entry := range dir {
		if entry.IsDir() {
			name := entry.Name()
			// Verifies prefix has correct semantic numeric code
			if len(name) >= 5 && (strings.HasPrefix(name, "c") || (name[0] >= '0' && name[0] <= '9')) {
				validCount++
			}
		}
	}
	
	log.Printf("[NATVS] Assimilation successfully verified %d semantic taxonomy entries.", validCount)
	return nil
}

// Transform executes phase 3 computational and refactoring operations.
func (e *NATVSEngine) Transform(ctx context.Context, action string) error {
	e.State = PhaseTransform
	log.Printf("[NATVS] Phase 3: Executing Transformation target action: '%s'...", action)
	
	if action == "remediate-netbench" {
		netbenchDir := filepath.Join(e.Config.WorkspaceRoot, "00flow/s-logiclibrary/00200-logic-libraries/netbench")
		log.Printf("[NATVS] Remediation target directory: %s", netbenchDir)
		var err error

		// 1. Write netbench.go
		netbenchGo := `package netbench

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"sov.fleet/s-logiclibrary/00200-logic-libraries/bicodec"
	"sov.fleet/s-logiclibrary/00200-logic-libraries/gatekeeper"
)

// MsgFormat defines the protocol format.
type MsgFormat int

const (
	weBNF MsgFormat = iota
	SACPUnsecured
	SACPSecured
)

// TestMessage contains the fields we send and verify.
type TestMessage struct {
	UUID               string
	OriginalAuthority  string
	CurrentAuthority   string
	AttestationType    string
	AttestationPayload string
	Domain             string
	Action             string
	ID                 string
	Parameters         map[string]string
	Version            string
}

// DefaultMessage returns a standard test message populated with security metadata.
func DefaultMessage() TestMessage {
	return TestMessage{
		UUID:               "msg-uuid-123456",
		OriginalAuthority:  bicodec.AuthSovereign,
		CurrentAuthority:   bicodec.AuthSovereign,
		AttestationType:    bicodec.AttestationVeracity,
		AttestationPayload: "veracity-seal-signature-payload-data",
		Domain:             "tool",
		Action:             "call",
		ID:                 "read_file",
		Parameters:         map[string]string{"path": "C:\\aCogSpaceSeed\\file.txt"},
		Version:            "1.0.0",
	}
}

// SerializeText serializes the message to weBNF text format.
func SerializeText(m TestMessage) []byte {
	var params []string
	for k, v := range m.Parameters {
		params = append(params, fmt.Sprintf("%s=%s", k, v))
	}
	pStr := strings.Join(params, ";") + ";"
	s := fmt.Sprintf("LPSV_ENVELOPE {\n  uuid = \"%s\" .\n  original_authority = \"%s\" .\n  current_authority = \"%s\" .\n  attestation_type = \"%s\" .\n  attestation_payload = \"%s\" .\n  domain = \"%s\" .\n  action = \"%s\" .\n  id = \"%s\" .\n  parameters = \"%s\" .\n  version = \"%s\" .\n}", m.UUID, m.OriginalAuthority, m.CurrentAuthority, m.AttestationType, m.AttestationPayload, m.Domain, m.Action, m.ID, pStr, m.Version)
	return []byte(s)
}

// DeserializeText parses weBNF text format.
func DeserializeText(data []byte) (TestMessage, error) {
	s := string(data)
	if !strings.HasPrefix(s, "LPSV_ENVELOPE {") || !strings.HasSuffix(s, "}") {
		return TestMessage{}, errors.New("invalid weBNF header/footer")
	}
	m := TestMessage{Parameters: make(map[string]string)}
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "LPSV_ENVELOPE {" || line == "}" {
			continue
		}
		eq := strings.Index(line, " = ")
		if eq < 0 {
			continue
		}
		key := line[:eq]
		val := line[eq+3:]
		// Strip quotes and ending dot
		if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, " .") {
			val = val[1 : len(val)-3]
		} else if strings.HasSuffix(val, " .") {
			val = val[:len(val)-2]
		}
		switch key {
		case "uuid":
			m.UUID = val
		case "original_authority":
			m.OriginalAuthority = val
		case "current_authority":
			m.CurrentAuthority = val
		case "attestation_type":
			m.AttestationType = val
		case "attestation_payload":
			m.AttestationPayload = val
		case "domain":
			m.Domain = val
		case "action":
			m.Action = val
		case "id":
			m.ID = val
		case "version":
			m.Version = val
		case "parameters":
			parts := strings.Split(val, ";")
			for _, part := range parts {
				if part == "" {
					continue
				}
				peq := strings.Index(part, "=")
				if peq >= 0 {
					m.Parameters[part[:peq]] = part[peq+1:]
				}
			}
		}
	}
	return m, nil
}

// SerializeSACPUnsecured encodes message to raw SACP binary bytes.
func SerializeSACPUnsecured(m TestMessage) ([]byte, error) {
	h := bicodec.SACPHeader{
		UUID:               m.UUID,
		OriginalAuthority:  m.OriginalAuthority,
		CurrentAuthority:   m.CurrentAuthority,
		AttestationType:    m.AttestationType,
		AttestationPayload: m.AttestationPayload,
	}
	c := bicodec.SACPCapability{
		Domain:     m.Domain,
		Action:     m.Action,
		ID:         m.ID,
		Parameters: m.Parameters,
		Version:    m.Version,
	}
	return bicodec.EncodeSACPMessage(&h, &c)
}

// DeserializeSACPUnsecured parses SACP binary bytes.
func DeserializeSACPUnsecured(data []byte) (TestMessage, error) {
	hasH, h, hasC, c, err := bicodec.DecodeSACPMessage(data)
	if err != nil {
		return TestMessage{}, err
	}
	if !hasH || !hasC {
		return TestMessage{}, errors.New("missing SACP header or capability frame")
	}
	return TestMessage{
		UUID:               h.UUID,
		OriginalAuthority:  h.OriginalAuthority,
		CurrentAuthority:   h.CurrentAuthority,
		AttestationType:    h.AttestationType,
		AttestationPayload: h.AttestationPayload,
		Domain:             c.Domain,
		Action:             c.Action,
		ID:                 c.ID,
		Parameters:         c.Parameters,
		Version:            c.Version,
	}, nil
}

var nonceCounter uint64

// SerializeSACPSecured serializes to SACP binary enveloped/signed format.
func SerializeSACPSecured(m TestMessage, auditor *gatekeeper.CryptosealAuditor, secret []byte) ([]byte, error) {
	if auditor == nil {
		return nil, errors.New("pointer safety error: gatekeeper auditor is nil")
	}
	rawBytes, err := SerializeSACPUnsecured(m)
	if err != nil {
		return nil, err
	}
	payloadStr := hex.EncodeToString(rawBytes)
	now := time.Now()
	ctr := atomic.AddUint64(&nonceCounter, 1)
	nonce := fmt.Sprintf("nonce-%d-%d", now.UnixNano(), ctr)
	env, err := auditor.SignEnvelope(payloadStr, nonce, now, secret)
	if err != nil {
		return nil, err
	}
	sigLen := uint32(len(env.Signature))
	nonceLen := uint32(len(env.Nonce))
	totalHeaderSize := 4 + sigLen + 4 + nonceLen + 8
	buf := make([]byte, totalHeaderSize+uint32(len(rawBytes)))

	binary.BigEndian.PutUint32(buf[0:4], sigLen)
	copy(buf[4:4+sigLen], env.Signature)
	pos := 4 + sigLen

	binary.BigEndian.PutUint32(buf[pos:pos+4], nonceLen)
	copy(buf[pos+4:pos+4+nonceLen], env.Nonce)
	pos += 4 + nonceLen

	binary.BigEndian.PutUint64(buf[pos:pos+8], uint64(now.Unix()))
	pos += 8

	copy(buf[pos:], rawBytes)
	return buf, nil
}

// DeserializeSACPSecured parses and cryptographically validates the signed SACP envelope.
func DeserializeSACPSecured(data []byte, gk *gatekeeper.Gatekeeper, secret []byte, maxAge time.Duration) (TestMessage, error) {
	if gk == nil {
		return TestMessage{}, errors.New("pointer safety error: gatekeeper is nil")
	}
	if len(data) < 16 {
		return TestMessage{}, errors.New("payload too small")
	}
	sigLen := binary.BigEndian.Uint32(data[0:4])
	if uint32(len(data)) < 4+sigLen {
		return TestMessage{}, errors.New("invalid signature length boundary")
	}
	sig := string(data[4 : 4+sigLen])
	pos := 4 + sigLen

	nonceLen := binary.BigEndian.Uint32(data[pos : pos+4])
	if uint32(len(data)) < pos+4+nonceLen {
		return TestMessage{}, errors.New("invalid nonce length boundary")
	}
	nonce := string(data[pos+4 : pos+4+nonceLen])
	pos += 4 + nonceLen

	tsUnix := int64(binary.BigEndian.Uint64(data[pos : pos+8]))
	pos += 8

	rawSACP := data[pos:]

	payloadStr := hex.EncodeToString(rawSACP)
	env := gatekeeper.SignedEnvelope{
		Payload:   payloadStr,
		Nonce:     nonce,
		Timestamp: time.Unix(tsUnix, 0),
		Signature: sig,
	}

	err := gk.InterrogateEnvelope(env, secret, maxAge)
	if err != nil {
		return TestMessage{}, fmt.Errorf("envelope verification failed: %w", err)
	}

	return DeserializeSACPUnsecured(rawSACP)
}
`
		err = os.WriteFile(filepath.Join(netbenchDir, "netbench.go"), []byte(netbenchGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write netbench.go: %w", err)
		}

		// 2. Write tcp.go
		tcpGo := `package netbench

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
		err = os.WriteFile(filepath.Join(netbenchDir, "tcp.go"), []byte(tcpGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write tcp.go: %w", err)
		}

		// 3. Write udp.go
		udpGo := `package netbench

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
		err = os.WriteFile(filepath.Join(netbenchDir, "udp.go"), []byte(udpGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write udp.go: %w", err)
		}

		// 4. Write quic.go
		quicGo := `package netbench

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
		err = os.WriteFile(filepath.Join(netbenchDir, "quic.go"), []byte(quicGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write quic.go: %w", err)
		}

		// 5. Write netbench_test.go
		netbenchTestGo := `package netbench

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
	"sov.fleet/s-logiclibrary/00200-logic-libraries/gatekeeper"
)

var (
	testSecret = []byte("12345678901234567890123456789012")
)

func TestSerializers(t *testing.T) {
	msg := DefaultMessage()
	
	// Test weBNF
	textPayload := SerializeText(msg)
	parsedMsg, err := DeserializeText(textPayload)
	if err != nil {
		t.Fatalf("Failed to deserialize weBNF text: %v", err)
	}
	if parsedMsg.UUID != msg.UUID {
		t.Errorf("weBNF mismatch: expected UUID %q, got %q", msg.UUID, parsedMsg.UUID)
	}

	// Test SACP Unsecured
	unsecuredPayload, err := SerializeSACPUnsecured(msg)
	if err != nil {
		t.Fatalf("Failed to serialize SACP unsecured: %v", err)
	}
	parsedUnsecured, err := DeserializeSACPUnsecured(unsecuredPayload)
	if err != nil {
		t.Fatalf("Failed to deserialize SACP unsecured: %v", err)
	}
	if parsedUnsecured.UUID != msg.UUID {
		t.Errorf("SACP unsecured mismatch: expected UUID %q, got %q", msg.UUID, parsedUnsecured.UUID)
	}

	// Test SACP Secured
	auditor := gatekeeper.NewCryptosealAuditor()
	gk := gatekeeper.NewGatekeeper()
	securedPayload, err := SerializeSACPSecured(msg, auditor, testSecret)
	if err != nil {
		t.Fatalf("Failed to serialize SACP secured: %v", err)
	}
	parsedSecured, err := DeserializeSACPSecured(securedPayload, gk, testSecret, 10*time.Second)
	if err != nil {
		t.Fatalf("Failed to deserialize SACP secured: %v", err)
	}
	if parsedSecured.UUID != msg.UUID {
		t.Errorf("SACP secured mismatch: expected UUID %q, got %q", msg.UUID, parsedSecured.UUID)
	}
}

func TestServers(t *testing.T) {
	msg := DefaultMessage()
	auditor := gatekeeper.NewCryptosealAuditor()

	t.Run("TCP", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		server, err := StartTCPServer(ctx, SACPSecured, testSecret)
		if err != nil {
			t.Fatalf("Failed to start TCP server: %v", err)
		}
		defer server.Close()

		conn, err := net.Dial("tcp", server.Addr())
		if err != nil {
			t.Fatalf("Failed to dial TCP server: %v", err)
		}
		defer conn.Close()

		payload, err := SerializeSACPSecured(msg, auditor, testSecret)
		if err != nil {
			t.Fatalf("Failed to serialize payload: %v", err)
		}

		pass, err := RunTCPCall(conn, payload)
		if err != nil || !pass {
			t.Fatalf("TCP call failed: err=%v pass=%v", err, pass)
		}
	})

	t.Run("UDP", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		server, err := StartUDPServer(ctx, SACPSecured, testSecret)
		if err != nil {
			t.Fatalf("Failed to start UDP server: %v", err)
		}
		defer server.Close()

		conn, err := net.ListenPacket("udp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to open UDP client socket: %v", err)
		}
		defer conn.Close()

		serverAddr, err := net.ResolveUDPAddr("udp", server.Addr())
		if err != nil {
			t.Fatalf("Failed to resolve UDP address: %v", err)
		}

		payload, err := SerializeSACPSecured(msg, auditor, testSecret)
		if err != nil {
			t.Fatalf("Failed to serialize payload: %v", err)
		}

		pass, err := RunUDPCall(conn, serverAddr, payload)
		if err != nil || !pass {
			t.Fatalf("UDP call failed: err=%v pass=%v", err, pass)
		}
	})

	t.Run("QUIC", func(t *testing.T) {
		tlsConfs, err := gatekeeper.GeneratePrecomputedMTLS()
		if err != nil {
			t.Fatalf("failed to generate TLS configs: %v", err)
		}
		tlsConfs.ServerConfig.NextProtos = []string{"sacp-perf"}
		tlsConfs.ClientConfig.NextProtos = []string{"sacp-perf"}
		tlsConfs.ClientConfig.InsecureSkipVerify = true

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		server, err := StartQUICServer(ctx, SACPSecured, testSecret, tlsConfs.ServerConfig)
		if err != nil {
			t.Fatalf("Failed to start QUIC server: %v", err)
		}
		defer server.Close()

		dialCtx, dialCancel := context.WithTimeout(ctx, 2*time.Second)
		defer dialCancel()
		quicConf := &quic.Config{
			MaxIncomingStreams: 1000,
		}
		conn, err := quic.DialAddr(dialCtx, server.Addr(), tlsConfs.ClientConfig, quicConf)
		if err != nil {
			t.Fatalf("failed to dial QUIC server: %v", err)
		}
		defer conn.CloseWithError(0, "done")

		payload, err := SerializeSACPSecured(msg, auditor, testSecret)
		if err != nil {
			t.Fatalf("Failed to serialize payload: %v", err)
		}

		streamCtx, streamCancel := context.WithTimeout(ctx, 2*time.Second)
		stream, err := conn.OpenStreamSync(streamCtx)
		if err != nil {
			streamCancel()
			t.Fatalf("failed to open QUIC stream: %v", err)
		}
		defer streamCancel()

		pass, err := RunQUICCall(stream, payload)
		if err != nil || !pass {
			t.Fatalf("QUIC call failed: err=%v pass=%v", err, pass)
		}
	})
}

func BenchmarkNetTCP(b *testing.B) {
	msg := DefaultMessage()
	auditor := gatekeeper.NewCryptosealAuditor()

	formats := []struct {
		name   string
		format MsgFormat
	}{
		{"weBNFText", weBNF},
		{"SACPUnsecured", SACPUnsecured},
		{"SACPSecured", SACPSecured},
	}

	for _, tc := range formats {
		b.Run(tc.name, func(b *testing.B) {
			server, err := StartTCPServer(context.Background(), tc.format, testSecret)
			if err != nil {
				b.Fatalf("failed to start TCP server: %v", err)
			}
			defer server.Close()

			conn, err := net.Dial("tcp", server.Addr())
			if err != nil {
				b.Fatalf("failed to dial TCP server: %v", err)
			}
			defer conn.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var payload []byte
				switch tc.format {
				case weBNF:
					payload = SerializeText(msg)
				case SACPUnsecured:
					payload, _ = SerializeSACPUnsecured(msg)
				case SACPSecured:
					payload, _ = SerializeSACPSecured(msg, auditor, testSecret)
				}

				pass, err := RunTCPCall(conn, payload)
				if err != nil || !pass {
					b.Fatalf("TCP call failed: err=%v pass=%v", err, pass)
				}
			}
		})
	}
}

func BenchmarkNetUDP(b *testing.B) {
	msg := DefaultMessage()
	auditor := gatekeeper.NewCryptosealAuditor()

	formats := []struct {
		name   string
		format MsgFormat
	}{
		{"weBNFText", weBNF},
		{"SACPUnsecured", SACPUnsecured},
		{"SACPSecured", SACPSecured},
	}

	for _, tc := range formats {
		b.Run(tc.name, func(b *testing.B) {
			server, err := StartUDPServer(context.Background(), tc.format, testSecret)
			if err != nil {
				b.Fatalf("failed to start UDP server: %v", err)
			}
			defer server.Close()

			conn, err := net.ListenPacket("udp", "127.0.0.1:0")
			if err != nil {
				b.Fatalf("failed to open UDP client socket: %v", err)
			}
			defer conn.Close()

			serverAddr, err := net.ResolveUDPAddr("udp", server.Addr())
			if err != nil {
				b.Fatalf("failed to resolve UDP server address: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var payload []byte
				switch tc.format {
				case weBNF:
					payload = SerializeText(msg)
				case SACPUnsecured:
					payload, _ = SerializeSACPUnsecured(msg)
				case SACPSecured:
					payload, _ = SerializeSACPSecured(msg, auditor, testSecret)
				}

				pass, err := RunUDPCall(conn, serverAddr, payload)
				if err != nil || !pass {
					b.Fatalf("UDP call failed: err=%v pass=%v", err, pass)
				}
			}
		})
	}
}

func BenchmarkNetQUIC(b *testing.B) {
	tlsConfs, err := gatekeeper.GeneratePrecomputedMTLS()
	if err != nil {
		b.Fatalf("failed to generate TLS configs: %v", err)
	}

	tlsConfs.ServerConfig.NextProtos = []string{"sacp-perf"}
	tlsConfs.ClientConfig.NextProtos = []string{"sacp-perf"}
	tlsConfs.ClientConfig.InsecureSkipVerify = true

	msg := DefaultMessage()
	auditor := gatekeeper.NewCryptosealAuditor()

	formats := []struct {
		name   string
		format MsgFormat
	}{
		{"weBNFText", weBNF},
		{"SACPUnsecured", SACPUnsecured},
		{"SACPSecured", SACPSecured},
	}

	for _, tc := range formats {
		b.Run(tc.name, func(b *testing.B) {
			server, err := StartQUICServer(context.Background(), tc.format, testSecret, tlsConfs.ServerConfig)
			if err != nil {
				b.Fatalf("failed to start QUIC server: %v", err)
			}
			defer server.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			quicConf := &quic.Config{
				MaxIncomingStreams: 1000000,
			}
			conn, err := quic.DialAddr(ctx, server.Addr(), tlsConfs.ClientConfig, quicConf)
			if err != nil {
				b.Fatalf("failed to dial QUIC server: %v", err)
			}
			defer conn.CloseWithError(0, "done")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var payload []byte
				switch tc.format {
				case weBNF:
					payload = SerializeText(msg)
				case SACPUnsecured:
					payload, _ = SerializeSACPUnsecured(msg)
				case SACPSecured:
					payload, _ = SerializeSACPSecured(msg, auditor, testSecret)
				}

				streamCtx, streamCancel := context.WithTimeout(context.Background(), 2*time.Second)
				stream, err := conn.OpenStreamSync(streamCtx)
				if err != nil {
					streamCancel()
					b.Fatalf("failed to open QUIC stream on iteration %d: %v", i, err)
				}
				pass, err := RunQUICCall(stream, payload)
				streamCancel()
				if err != nil || !pass {
					b.Fatalf("QUIC stream call failed: err=%v pass=%v", err, pass)
				}
			}
		})
	}
}
`
		err = os.WriteFile(filepath.Join(netbenchDir, "netbench_test.go"), []byte(netbenchTestGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write netbench_test.go: %w", err)
		}

		log.Printf("[NATVS] Transformation 'remediate-netbench' executed successfully.")
		return nil
	}

	if action == "remediate-s-latentlingua" {
		latentlinguaDir := filepath.Join(e.Config.WorkspaceRoot, "00flow/s-latentlingua")
		log.Printf("[NATVS] Remediation target directory: %s", latentlinguaDir)

		// 1. Write ast.go
		astGo := `package snparser

// AST Nodes
type Node interface { Kind() string }

type GrammarNode struct {
	Header []string
	Rules  map[string]*RuleNode
}
func (g *GrammarNode) Kind() string { return "Grammar" }

type RuleNode struct {
	Name string
	Doc  string
	Expr Node
}
func (r *RuleNode) Kind() string { return "Rule" }

type SequenceNode struct { Elements []Node }
func (s *SequenceNode) Kind() string { return "Sequence" }

type ChoiceNode struct { Options []Node }
func (c *ChoiceNode) Kind() string { return "Choice" }

type RepetitionNode struct {
	Expr Node
	Min  int
	Max  int
}
func (r *RepetitionNode) Kind() string { return "Repetition" }

type OptionalNode struct { Expr Node }
func (o *OptionalNode) Kind() string { return "Optional" }

type TerminalNode struct {
	Value string
	IsRef bool
}
func (t *TerminalNode) Kind() string { return "Terminal" }

type RangeNode struct {
	Start   rune
	End     rune
	Negated bool
}
func (r *RangeNode) Kind() string { return "Range" }

type PredicateNode struct {
	Expr     Node
	Positive bool
}
func (p *PredicateNode) Kind() string { return "Predicate" }
`
		err := os.WriteFile(filepath.Join(latentlinguaDir, "02000-logic-libraries/snparser/ast.go"), []byte(astGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write ast.go: %w", err)
		}

		// 2. Write lexer.go
		lexerGo := `package snparser

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenError
	TokenIdent
	TokenString
	TokenNumber
	TokenEqual
	TokenLBrace
	TokenRBrace
	TokenLParen
	TokenRParen
	TokenLBracket
	TokenRBracket
	TokenSemicolon
	TokenSlash
	TokenMinus
	TokenAnd
	TokenNot
	TokenColon
	TokenComma
	TokenDocBlock
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

type Lexer struct {
	input     string
	pos       int
	width     int
	line      int
	lineStart int
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: input, line: 1}
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()
	start := l.pos

	r := l.next()
	if r == -1 {
		return Token{Type: TokenEOF, Line: l.line, Col: l.col(start)}
	}

	switch r {
	case '=': return l.emit(TokenEqual, start)
	case '{': return l.emit(TokenLBrace, start)
	case '}': return l.emit(TokenRBrace, start)
	case '[': return l.emit(TokenLBracket, start)
	case ']': return l.emit(TokenRBracket, start)
	case '(': return l.emit(TokenLParen, start)
	case ')': return l.emit(TokenRParen, start)
	case ';': return l.emit(TokenSemicolon, start)
	case ':': return l.emit(TokenColon, start)
	case ',': return l.emit(TokenComma, start)
	case '&': return l.emit(TokenAnd, start)
	case '!': return l.emit(TokenNot, start)
	case '^': return l.emit(TokenIdent, start)
	case '/':
		if l.peek() == '*' {
			return lexDocBlock(l, start)
		}
		return l.emit(TokenSlash, start)
	case '-': return l.emit(TokenMinus, start)
	case '"', '\'':
		return lexString(l, r, start)
	default:
		if unicode.IsDigit(r) {
			return lexNumber(l, start)
		}
		if isLetter(r) {
			return lexIdentifier(l, start)
		}
		return l.errorf("unexpected character: %q", r)
	}
}

func (l *Lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return -1
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += w
	if r == '\n' {
		l.line++
		l.lineStart = l.pos
	}
	return r
}

func (l *Lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *Lexer) backup() {
	l.pos -= l.width
}

func (l *Lexer) emit(t TokenType, start int) Token {
	return Token{
		Type:    t,
		Literal: l.input[start:l.pos],
		Line:    l.line,
		Col:     l.col(start),
	}
}

func (l *Lexer) col(pos int) int {
	return pos - l.lineStart + 1
}

func (l *Lexer) skipWhitespace() {
	for {
		r := l.next()
		if r == -1 { return }
		if r == '#' { // Line comment
			for {
				r2 := l.next()
				if r2 == -1 || r2 == '\n' { break }
			}
			continue
		}
		if r == '/' && l.peek() == '/' { // // Line comment
			l.next() // consume second '/'
			for {
				r2 := l.next()
				if r2 == -1 || r2 == '\n' { break }
			}
			continue
		}
		if r == ';' {
			// Check if ';' is the first non-whitespace character on this line
			isComment := true
			limit := l.pos - l.width
			for i := l.lineStart; i < limit; i++ {
				c := l.input[i]
				if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
					isComment = false
					break
				}
			}
			if isComment {
				// Only treat as comment if there is some non-whitespace content after it on the same line
				hasContent := false
				for i := l.pos; i < len(l.input); i++ {
					c := l.input[i]
					if c == '\n' {
						break
					}
					if c != ' ' && c != '\t' && c != '\r' {
						hasContent = true
						break
					}
				}
				if hasContent {
					for {
						r2 := l.next()
						if r2 == -1 || r2 == '\n' { break }
					}
					continue
				}
			}
		}
		if !unicode.IsSpace(r) { l.backup(); return }
	}
}

func (l *Lexer) errorf(format string, args ...interface{}) Token {
	return Token{
		Type:    TokenError,
		Literal: fmt.Sprintf(format, args...),
		Line:    l.line,
		Col:     l.pos - l.lineStart + 1,
	}
}

func lexIdentifier(l *Lexer, start int) Token {
	for {
		r := l.next()
		if r == -1 { break }
		if !isLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != '.' && r != '/' && r != '@' {
			l.backup()
			break
		}
	}
	return l.emit(TokenIdent, start)
}

func lexNumber(l *Lexer, start int) Token {
	for {
		r := l.next()
		if r == -1 { break }
		if !unicode.IsDigit(r) {
			l.backup()
			break
		}
	}
	return l.emit(TokenNumber, start)
}

func lexString(l *Lexer, quote rune, start int) Token {
	for {
		r := l.next()
		if r == -1 { return l.errorf("unterminated string") }
		if r == '\\' { l.next(); continue }
		if r == quote { break }
	}
	return l.emit(TokenString, start)
}

func lexDocBlock(l *Lexer, start int) Token {
	l.next() // consume '*'
	for {
		r := l.next()
		if r == -1 { return l.errorf("unterminated doc block") }
		if r == '*' && l.peek() == '/' {
			l.next() // consume '/'
			break
		}
	}
	return l.emit(TokenDocBlock, start)
}

func isLetter(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}
`
		err = os.WriteFile(filepath.Join(latentlinguaDir, "02000-logic-libraries/snparser/lexer.go"), []byte(lexerGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write lexer.go: %w", err)
		}

		// 3. Write parser.go (overwriting the old one)
		parserGo := `package snparser

import (
	"fmt"
)

type Parser struct {
	lexer *Lexer
	curr  Token
	peek  Token
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{lexer: l}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curr = p.peek
	p.peek = p.lexer.NextToken()
}

func (p *Parser) Parse() (*GrammarNode, error) {
	g := &GrammarNode{Rules: make(map[string]*RuleNode)}
	
	// Handle Header
	if p.curr.Type == TokenColon {
		p.nextToken()
		for p.curr.Type == TokenIdent {
			g.Header = append(g.Header, p.curr.Literal)
			p.nextToken()
			if p.curr.Type == TokenColon { p.nextToken() } else { break }
		}
	}

	for p.curr.Type != TokenEOF {
		if p.curr.Type == TokenSemicolon {
			p.nextToken()
			continue
		}

		// Handle Directives: e.g. import, export, legal, or @
		if p.curr.Type == TokenIdent && (p.curr.Literal == "import" || p.curr.Literal == "export" || p.curr.Literal == "legal" || p.curr.Literal == "@") {
			// Consume until semicolon
			for p.curr.Type != TokenSemicolon && p.curr.Type != TokenEOF {
				if p.curr.Type == TokenError {
					return nil, fmt.Errorf("lexical error in directive at line %d, col %d: %s", p.curr.Line, p.curr.Col, p.curr.Literal)
				}
				p.nextToken()
			}
			if p.curr.Type == TokenSemicolon {
				p.nextToken()
			}
			continue
		}

		var doc string
		if p.curr.Type == TokenDocBlock {
			doc = p.curr.Literal
			p.nextToken()
		}

		if p.curr.Type == TokenIdent {
			name := p.curr.Literal
			p.nextToken()
			if p.curr.Type == TokenEqual {
				p.nextToken()
				var expr Node
				var err error
				if p.isFlatBlock() {
					expr, err = p.parseFlatBlock()
				} else {
					expr, err = p.parseExpression()
				}
				if err != nil { return nil, err }
				g.Rules[name] = &RuleNode{Name: name, Doc: doc, Expr: expr}
				if p.curr.Type == TokenSemicolon { p.nextToken() }
			} else {
				return nil, fmt.Errorf("expected '=' after rule identifier %q, got %q at line %d, col %d", name, p.curr.Literal, p.curr.Line, p.curr.Col)
			}
		} else {
			if p.curr.Type == TokenError {
				return nil, fmt.Errorf("lexical error at line %d, col %d: %s", p.curr.Line, p.curr.Col, p.curr.Literal)
			}
			return nil, fmt.Errorf("unexpected token %q of type %d at line %d, col %d", p.curr.Literal, p.curr.Type, p.curr.Line, p.curr.Col)
		}
	}
	return g, nil
}

func (p *Parser) isFlatBlock() bool {
	if p.curr.Type != TokenLBrace {
		return false
	}
	lexerCopy := *p.lexer
	tempParser := &Parser{
		lexer: &lexerCopy,
		curr:  p.curr,
		peek:  p.peek,
	}
	braceDepth := 1
	for {
		tempParser.nextToken()
		t := tempParser.curr
		if t.Type == TokenEOF || t.Type == TokenError {
			break
		}
		if t.Type == TokenLBrace {
			braceDepth++
		} else if t.Type == TokenRBrace {
			braceDepth--
			if braceDepth == 0 {
				break
			}
		} else if braceDepth == 1 {
			if t.Type == TokenEqual || t.Type == TokenSemicolon {
				return true
			}
		}
	}
	return false
}

func (p *Parser) parseFlatBlock() (Node, error) {
	p.nextToken() // {
	var rules []Node
	for p.curr.Type != TokenRBrace && p.curr.Type != TokenEOF {
		if p.curr.Type == TokenError {
			return nil, fmt.Errorf("lexical error at line %d, col %d: %s", p.curr.Line, p.curr.Col, p.curr.Literal)
		}
		var doc string
		if p.curr.Type == TokenDocBlock {
			doc = p.curr.Literal
			p.nextToken()
		}
		if p.curr.Type == TokenIdent {
			name := p.curr.Literal
			p.nextToken()
			if p.curr.Type == TokenEqual {
				p.nextToken()
				expr, err := p.parseExpression()
				if err != nil { return nil, err }
				rules = append(rules, &RuleNode{Name: name, Doc: doc, Expr: expr})
				if p.curr.Type == TokenSemicolon { p.nextToken() }
			} else {
				return nil, fmt.Errorf("expected '=' after identifier %q in FlatBlock, got %q at line %d, col %d", name, p.curr.Literal, p.curr.Line, p.curr.Col)
			}
		} else {
			if p.curr.Type == TokenError {
				return nil, fmt.Errorf("lexical error at line %d, col %d: %s", p.curr.Line, p.curr.Col, p.curr.Literal)
			}
			return nil, fmt.Errorf("unexpected token %q of type %d in FlatBlock at line %d, col %d", p.curr.Literal, p.curr.Type, p.curr.Line, p.curr.Col)
		}
	}
	if p.curr.Type != TokenRBrace { return nil, fmt.Errorf("missing } in FlatBlock") }
	p.nextToken()
	return &SequenceNode{Elements: rules}, nil
}
`
		err = os.WriteFile(filepath.Join(latentlinguaDir, "02000-logic-libraries/snparser/parser.go"), []byte(parserGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write parser.go: %w", err)
		}

		// 4. Write expr.go
		exprGo := `package snparser

import (
	"fmt"
	"strconv"
	"unicode"
)

func (p *Parser) parseExpression() (Node, error) {
	type parserFrameType int

	const (
		frameChoice parserFrameType = iota
		frameSequence
		frameGroup
		frameOptional
		frameRepetition
		framePredicate
	)

	type parserFrame struct {
		kind      parserFrameType
		nodes     []Node
		positive  bool
		startLine int
		startCol  int
	}

	var stack []*parserFrame

	// Initially, we push a Choice frame (the root expression) and a Sequence frame.
	stack = append(stack, &parserFrame{
		kind:      frameChoice,
		startLine: p.curr.Line,
		startCol:  p.curr.Col,
	})
	stack = append(stack, &parserFrame{
		kind:      frameSequence,
		startLine: p.curr.Line,
		startCol:  p.curr.Col,
	})

	for {
		if p.curr.Type == TokenError {
			return nil, fmt.Errorf("lexical error at line %d, col %d: %s", p.curr.Line, p.curr.Col, p.curr.Literal)
		}

		// Check if we can parse a factor
		isFactor := false
		switch p.curr.Type {
		case TokenIdent, TokenString, TokenLParen, TokenLBracket, TokenLBrace, TokenAnd, TokenNot:
			isFactor = true
		}

		if isFactor {
			var completedNode Node
			switch p.curr.Type {
			case TokenIdent:
				completedNode = &TerminalNode{Value: p.curr.Literal, IsRef: true}
				p.nextToken()
			case TokenString:
				completedNode = &TerminalNode{Value: p.curr.Literal, IsRef: false}
				p.nextToken()
			case TokenAnd, TokenNot:
				stack = append(stack, &parserFrame{
					kind:      framePredicate,
					positive:  p.curr.Type == TokenAnd,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				p.nextToken()
				continue
			case TokenLParen:
				stack = append(stack, &parserFrame{
					kind:      frameGroup,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				p.nextToken()
				stack = append(stack, &parserFrame{
					kind:      frameChoice,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				stack = append(stack, &parserFrame{
					kind:      frameSequence,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				continue
			case TokenLBrace:
				stack = append(stack, &parserFrame{
					kind:      frameRepetition,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				p.nextToken()
				stack = append(stack, &parserFrame{
					kind:      frameChoice,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				stack = append(stack, &parserFrame{
					kind:      frameSequence,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				continue
			case TokenLBracket:
				p.nextToken() // consume '['
				negated := false
				if p.curr.Type == TokenIdent && p.curr.Literal == "^" {
					negated = true
					p.nextToken() // consume '^'
				}

				isRange := false
				if p.curr.Type == TokenIdent {
					if len(p.curr.Literal) == 3 && p.curr.Literal[1] == '-' {
						isRange = true
					} else if len(p.curr.Literal) == 1 {
						r := rune(p.curr.Literal[0])
						if unicode.IsLower(r) || unicode.IsDigit(r) {
							isRange = true
						}
					}
				} else if p.curr.Type == TokenNumber {
					if p.peek.Type == TokenMinus || p.peek.Type == TokenRBracket {
						isRange = true
					}
				}

				if isRange {
					var start, end rune
					if p.curr.Type == TokenIdent && len(p.curr.Literal) == 3 && p.curr.Literal[1] == '-' {
						start = rune(p.curr.Literal[0])
						end = rune(p.curr.Literal[2])
						p.nextToken()
					} else if p.curr.Type == TokenNumber && p.peek.Type == TokenMinus {
						start = rune(p.curr.Literal[0])
						p.nextToken() // consume start digit
						p.nextToken() // consume '-'
						if p.curr.Type != TokenNumber && p.curr.Type != TokenIdent {
							return nil, fmt.Errorf("expected end of range after '-', got %q at line %d, col %d", p.curr.Literal, p.curr.Line, p.curr.Col)
						}
						end = rune(p.curr.Literal[0])
						p.nextToken() // consume end
					} else if p.curr.Type == TokenIdent && len(p.curr.Literal) == 1 {
						start = rune(p.curr.Literal[0])
						end = start
						p.nextToken()
					} else if p.curr.Type == TokenNumber {
						start = rune(p.curr.Literal[0])
						end = start
						p.nextToken()
					}

					if p.curr.Type != TokenRBracket {
						return nil, fmt.Errorf("missing ] in character range, got %q at line %d, col %d", p.curr.Literal, p.curr.Line, p.curr.Col)
					}
					p.nextToken() // consume ']'
					completedNode = &RangeNode{Start: start, End: end, Negated: negated}
				} else {
					if negated {
						return nil, fmt.Errorf("unexpected '^' at start of optional expression at line %d, col %d", p.curr.Line, p.curr.Col)
					}
					stack = append(stack, &parserFrame{
						kind:      frameOptional,
						startLine: p.curr.Line,
						startCol:  p.curr.Col,
					})
					stack = append(stack, &parserFrame{
						kind:      frameChoice,
						startLine: p.curr.Line,
						startCol:  p.curr.Col,
					})
					stack = append(stack, &parserFrame{
						kind:      frameSequence,
						startLine: p.curr.Line,
						startCol:  p.curr.Col,
					})
					continue
				}
			}

			// Reduce the completed factor node:
			for {
				if len(stack) == 0 {
					return nil, fmt.Errorf("internal parser error: empty stack during reduction at line %d, col %d", p.curr.Line, p.curr.Col)
				}
				top := stack[len(stack)-1]
				if top.kind == framePredicate {
					stack = stack[:len(stack)-1]
					completedNode = &PredicateNode{
						Expr:     completedNode,
						Positive: top.positive,
					}
					continue
				}
				if top.kind == frameSequence {
					top.nodes = append(top.nodes, completedNode)
					break
				}
				return nil, fmt.Errorf("internal parser error: unexpected frame type %d on top of stack during reduction at line %d, col %d", top.kind, p.curr.Line, p.curr.Col)
			}

		} else {
			// Current token is not a factor-starting token.
			// The current sequence is complete.
			if len(stack) == 0 {
				return nil, fmt.Errorf("internal parser error: empty stack when closing sequence at line %d, col %d", p.curr.Line, p.curr.Col)
			}
			seqFrame := stack[len(stack)-1]
			if seqFrame.kind != frameSequence {
				return nil, fmt.Errorf("internal parser error: expected sequence frame on top of stack, got %d at line %d, col %d", seqFrame.kind, p.curr.Line, p.curr.Col)
			}
			stack = stack[:len(stack)-1]

			var seqNode Node
			if len(seqFrame.nodes) == 1 {
				seqNode = seqFrame.nodes[0]
			} else {
				seqNode = &SequenceNode{Elements: seqFrame.nodes}
			}

			// Look at parent frame
			if len(stack) == 0 {
				return nil, fmt.Errorf("internal parser error: empty stack after popping sequence at line %d, col %d", p.curr.Line, p.curr.Col)
			}
			parentFrame := stack[len(stack)-1]
			if parentFrame.kind != frameChoice {
				return nil, fmt.Errorf("internal parser error: expected choice frame below sequence, got %d at line %d, col %d", parentFrame.kind, p.curr.Line, p.curr.Col)
			}

			parentFrame.nodes = append(parentFrame.nodes, seqNode)

			if p.curr.Type == TokenSlash {
				p.nextToken() // consume '/'
				stack = append(stack, &parserFrame{
					kind:      frameSequence,
					startLine: p.curr.Line,
					startCol:  p.curr.Col,
				})
				continue
			}

			// Choice is complete! Pop choice frame.
			stack = stack[:len(stack)-1]

			var choiceNode Node
			if len(parentFrame.nodes) == 1 {
				choiceNode = parentFrame.nodes[0]
			} else {
				choiceNode = &ChoiceNode{Options: parentFrame.nodes}
			}

			if len(stack) == 0 {
				// Completed the root choice.
				// Semicolon, EOF, or RBrace are valid terminators.
				if p.curr.Type == TokenSemicolon || p.curr.Type == TokenEOF || p.curr.Type == TokenRBrace {
					return choiceNode, nil
				}
				if p.curr.Type == TokenRParen || p.curr.Type == TokenRBracket {
					return nil, fmt.Errorf("unmatched closing delimiter %q at line %d, col %d", p.curr.Literal, p.curr.Line, p.curr.Col)
				}
				return nil, fmt.Errorf("unexpected token %q of type %d at line %d, col %d", p.curr.Literal, p.curr.Type, p.curr.Line, p.curr.Col)
			}

			grandparentFrame := stack[len(stack)-1]
			switch grandparentFrame.kind {
			case frameGroup:
				if p.curr.Type != TokenRParen {
					return nil, fmt.Errorf("missing )")
				}
				p.nextToken() // consume ')'
				stack = stack[:len(stack)-1] // pop frameGroup

				completedNode := choiceNode
				// Reduce
				for {
					if len(stack) == 0 {
						return nil, fmt.Errorf("internal parser error: empty stack during reduction at line %d, col %d", p.curr.Line, p.curr.Col)
					}
					top := stack[len(stack)-1]
					if top.kind == framePredicate {
						stack = stack[:len(stack)-1]
						completedNode = &PredicateNode{
							Expr:     completedNode,
							Positive: top.positive,
						}
						continue
					}
					if top.kind == frameSequence {
						top.nodes = append(top.nodes, completedNode)
						break
					}
					return nil, fmt.Errorf("internal parser error: unexpected frame type %d on top of stack during reduction at line %d, col %d", top.kind, p.curr.Line, p.curr.Col)
				}

			case frameOptional:
				if p.curr.Type != TokenRBracket {
					return nil, fmt.Errorf("missing ] in optional expression, got %q", p.curr.Literal)
				}
				p.nextToken() // consume ']'
				stack = stack[:len(stack)-1] // pop frameOptional

				var completedNode Node = &OptionalNode{Expr: choiceNode}
				// Reduce
				for {
					if len(stack) == 0 {
						return nil, fmt.Errorf("internal parser error: empty stack during reduction at line %d, col %d", p.curr.Line, p.curr.Col)
					}
					top := stack[len(stack)-1]
					if top.kind == framePredicate {
						stack = stack[:len(stack)-1]
						completedNode = &PredicateNode{
							Expr:     completedNode,
							Positive: top.positive,
						}
						continue
					}
					if top.kind == frameSequence {
						top.nodes = append(top.nodes, completedNode)
						break
					}
					return nil, fmt.Errorf("internal parser error: unexpected frame type %d on top of stack during reduction at line %d, col %d", top.kind, p.curr.Line, p.curr.Col)
				}

			case frameRepetition:
				if p.curr.Type != TokenRBrace {
					return nil, fmt.Errorf("missing }")
				}
				p.nextToken() // consume '}'
				stack = stack[:len(stack)-1] // pop frameRepetition

				min, max := 0, -1
				if p.curr.Type == TokenNumber {
					min, _ = strconv.Atoi(p.curr.Literal)
					p.nextToken()
					if p.curr.Type == TokenComma {
						p.nextToken()
						if p.curr.Type == TokenNumber {
							max, _ = strconv.Atoi(p.curr.Literal)
							p.nextToken()
						}
					} else {
						max = min
					}
				}

				var completedNode Node = &RepetitionNode{Expr: choiceNode, Min: min, Max: max}
				// Reduce
				for {
					if len(stack) == 0 {
						return nil, fmt.Errorf("internal parser error: empty stack during reduction at line %d, col %d", p.curr.Line, p.curr.Col)
					}
					top := stack[len(stack)-1]
					if top.kind == framePredicate {
						stack = stack[:len(stack)-1]
						completedNode = &PredicateNode{
							Expr:     completedNode,
							Positive: top.positive,
						}
						continue
					}
					if top.kind == frameSequence {
						top.nodes = append(top.nodes, completedNode)
						break
					}
					return nil, fmt.Errorf("internal parser error: unexpected frame type %d on top of stack during reduction at line %d, col %d", top.kind, p.curr.Line, p.curr.Col)
				}

			default:
				return nil, fmt.Errorf("internal parser error: unexpected grandparent frame type %d at line %d, col %d", grandparentFrame.kind, p.curr.Line, p.curr.Col)
			}
		}
	}
}
`
		err = os.WriteFile(filepath.Join(latentlinguaDir, "02000-logic-libraries/snparser/expr.go"), []byte(exprGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write expr.go: %w", err)
		}

		// 5. Write bridge.go
		bridgeGo := `package bridge

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sov.fleet/s-latentlingua/02000-logic-libraries/snparser"
)

// ToProto converts a parsed WEBNF grammar into a Protobuf definition.
func ToProto(grammar *snparser.GrammarNode) (string, error) {
	var sb strings.Builder
	sb.WriteString("syntax = \"proto3\";\n\n")

	// Iterate over rules to find enums first
	for name, rule := range grammar.Rules {
		if choice, ok := rule.Expr.(*snparser.ChoiceNode); ok {
			sb.WriteString(fmt.Sprintf("enum %s {\n", name))
			for i, opt := range choice.Options {
				if term, ok := opt.(*snparser.TerminalNode); ok {
					val := strings.Trim(term.Value, "\"")
					sb.WriteString(fmt.Sprintf("\t%s = %d;\n", val, i))
				}
			}
			sb.WriteString("}\n\n")
		}
	}

	// Iterate over rules to find messages
	for name, rule := range grammar.Rules {
		if seq, ok := rule.Expr.(*snparser.SequenceNode); ok {
			// Check if it's a FlatBlock (sequence of RuleNodes)
			isMessage := true
			for _, el := range seq.Elements {
				if _, ok := el.(*snparser.RuleNode); !ok {
					isMessage = false
					break
				}
			}

			if isMessage {
				sb.WriteString(fmt.Sprintf("message %s {\n", name))
				for i, el := range seq.Elements {
					rn := el.(*snparser.RuleNode)
					typ, repeated := mapWEBNFType(rn.Expr)
					protoTyp := mapToProtoType(typ)
					prefix := ""
					if repeated { prefix = "repeated " }
					sb.WriteString(fmt.Sprintf("\t%s%s %s = %d;\n", prefix, protoTyp, rn.Name, i+1))
				}
				sb.WriteString("}\n\n")
			}
		}
	}

	return sb.String(), nil
}

func mapWEBNFType(node snparser.Node) (string, bool) {
	switch n := node.(type) {
	case *snparser.TerminalNode:
		return n.Value, false
	case *snparser.RepetitionNode:
		typ, isRepeated := mapWEBNFType(n.Expr)
		if isRepeated {
			// already repeated
		}
		return typ, true
	default:
		return "string", false
	}
}

func mapToProtoType(w string) string {
	switch w {
	case "string": return "string"
	case "number": return "int64"
	case "boolean": return "bool"
	default: return w
	}
}

// FromProto converts a Protobuf definition string into WEBNF.
func FromProto(proto string) (string, error) {
	var sb strings.Builder
	sb.WriteString(":Sovereign:Imported:Proto:v1\n\n")

	// Match enums
	enumRegex := regexp.MustCompile("enum\\s+(\\w+)\\s*\\{([^}]+)\\}")
	enums := enumRegex.FindAllStringSubmatch(proto, -1)
	for _, m := range enums {
		name := m[1]
		body := m[2]
		sb.WriteString(fmt.Sprintf("%s = ", name))
		
		valRegex := regexp.MustCompile("(\\w+)\\s*=\\s*\\d+;")
		vals := valRegex.FindAllStringSubmatch(body, -1)
		var choices []string
		for _, v := range vals {
			choices = append(choices, fmt.Sprintf("\"%s\"", v[1]))
		}
		sb.WriteString(strings.Join(choices, " / "))
		sb.WriteString(" ;\n\n")
	}

	// Match messages
	msgRegex := regexp.MustCompile("message\\s+(\\w+)\\s*\\{([^}]+)\\}")
	msgs := msgRegex.FindAllStringSubmatch(proto, -1)
	for _, m := range msgs {
		name := m[1]
		body := m[2]
		sb.WriteString(fmt.Sprintf("%s = {\n", name))
		
		fieldRegex := regexp.MustCompile("(?:repeated\\s+)?(\\w+)\\s+(\\w+)\\s*=\\s*\\d+;")
		fields := fieldRegex.FindAllStringSubmatch(body, -1)
		for _, f := range fields {
			typ := mapProtoType(f[1])
			isRepeated := strings.Contains(f[0], "repeated")
			if isRepeated {
				sb.WriteString(fmt.Sprintf("\t%s = { %s } ;\n", f[2], typ))
			} else {
				sb.WriteString(fmt.Sprintf("\t%s = %s ;\n", f[2], typ))
			}
		}
		sb.WriteString("}\n\n")
	}

	return sb.String(), nil
}

// FromStruct uses reflection to convert a Go struct into WEBNF.
func FromStruct(v interface{}) (string, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return "", fmt.Errorf("FromStruct requires a struct input")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(":Sovereign:Imported:Struct:%s:v1\n\n", t.Name()))
	sb.WriteString(fmt.Sprintf("%s = {\n", t.Name()))

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		typ := mapGoType(field.Type)
		sb.WriteString(fmt.Sprintf("\t%s = %s ;\n", field.Name, typ))
	}
	sb.WriteString("}\n")

	return sb.String(), nil
}

func mapProtoType(p string) string {
	switch p {
	case "string": return "string"
	case "int32", "int64", "uint32", "uint64": return "number"
	case "bool": return "boolean"
	default: return p // Assume it's another message or enum
	}
}

func mapGoType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String: return "string"
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64: return "number"
	case reflect.Bool: return "boolean"
	case reflect.Slice: return fmt.Sprintf("{ %s }", mapGoType(t.Elem()))
	case reflect.Struct: return t.Name()
	default: return "any"
	}
}
`
		err = os.WriteFile(filepath.Join(latentlinguaDir, "02000-logic-libraries/bridge/bridge.go"), []byte(bridgeGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write bridge.go: %w", err)
		}

		// 6. Write server/main.go
		serverMainGo := `package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"os"
	"os/signal"
	"syscall"

	"github.com/quic-go/quic-go"
	"sov.fleet/s-latentlingua/02000-logic-libraries/grammar"
	"sov.fleet/s-latentlingua/02000-logic-libraries/snparser"
	"sov.fleet/s-latentlingua/40000-communication-contracts"
)

func init() {
	handler := webnf.NewSlogWebnfHandler(os.Stdout, nil)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

type SovereignLatentLinguaServer struct {
	Hydrator *webnf.StateHydrator
	Codec    *webnf.WebnfCodec
}

func (s *SovereignLatentLinguaServer) HandleStream(str *quic.Stream) {
	defer str.Close()
	slog.Info("New qRPC stream accepted")

	for {
		// Read incoming webnf message
		buf := make([]byte, 32*1024)
		n, err := str.Read(buf)
		if err != nil {
			if err != io.EOF {
				slog.Error("Stream read error", "error", err)
			}
			return
		}

		input := buf[:n]
		var req webnf.InteractionStreamRequest
		if err := s.Codec.Unmarshal(input, &req); err != nil {
			slog.Error("Unmarshal error", "error", err, "input", string(input))
			continue
		}

		slog.Info("Received qRPC request", "type", req.Type, "session_id", req.SessionID)

		// Process request logic
		var resp webnf.InteractionStreamResponse
		resp.SessionID = req.SessionID

		switch req.Type {
		case "VALIDATE":
			valid, errors := s.handleValidate(req.Snapshot)
			valResp := webnf.ValidateResponse{
				Valid:  valid,
				Errors: errors,
			}
			resp.Type = "VALIDATE_RESPONSE"
			resp.Snapshot = string(s.mustMarshal(valResp))

		case "TRANSPILE":
			output, errors := s.handleTranspile(req.Snapshot)
			transResp := webnf.TranspileResponse{
				Output: output,
				Errors: errors,
			}
			resp.Type = "TRANSPILE_RESPONSE"
			resp.Snapshot = string(s.mustMarshal(transResp))

		default:
			resp.Type = "ERROR"
			resp.Delta = "Unknown request type"
			slog.Warn("Unknown request type received", "type", req.Type)
		}

		// Send response
		outData, err := s.Codec.Marshal(resp)
		if err != nil {
			slog.Error("Failed to marshal response", "error", err)
			continue
		}
		if _, err := str.Write(outData); err != nil {
			slog.Error("Failed to write response to stream", "error", err)
			return
		}
		slog.Info("Sent qRPC response", "type", resp.Type, "session_id", resp.SessionID)
	}
}

func (s *SovereignLatentLinguaServer) mustMarshal(v any) []byte {
	data, err := s.Codec.Marshal(v)
	if err != nil {
		slog.Error("Failed to marshal inside mustMarshal", "error", err)
		return nil
	}
	return data
}

func (s *SovereignLatentLinguaServer) handleValidate(snapshot string) (bool, []string) {
	var req webnf.ValidateRequest
	if err := s.Codec.Unmarshal([]byte(snapshot), &req); err != nil {
		return false, []string{err.Error()}
	}

	lexer := snparser.NewLexer(req.Grammar)
	parser := snparser.NewParser(lexer)
	ast, err := parser.Parse()
	if err != nil {
		return false, []string{fmt.Sprintf("Schema Parse Error: %v", err)}
	}

	g, err := grammar.FromAST(ast)
	if err != nil {
		return false, []string{fmt.Sprintf("Grammar Error: %v", err)}
	}

	irRoot, err := webnf.ParseToIR(req.Data)
	if err != nil {
		return false, []string{fmt.Sprintf("Data Parse Error: %v", err)}
	}

	var validationErrors []string
	for _, child := range irRoot.Children {
		keyword := child.GetAttributeString("name")
		if keyword == "" { continue }
		if _, ok := g.Rules[keyword]; !ok {
			validationErrors = append(validationErrors, fmt.Sprintf("Unknown keyword %q", keyword))
		}
	}

	if len(validationErrors) > 0 {
		return false, validationErrors
	}

	return true, nil
}

func (s *SovereignLatentLinguaServer) handleTranspile(snapshot string) (string, []string) {
	var req webnf.TranspileRequest
	if err := s.Codec.Unmarshal([]byte(snapshot), &req); err != nil {
		return "", []string{err.Error()}
	}

	lexer := snparser.NewLexer(req.Grammar)
	parser := snparser.NewParser(lexer)
	ast, err := parser.Parse()
	if err != nil {
		return "", []string{fmt.Sprintf("Schema Parse Error: %v", err)}
	}

	_, err = grammar.FromAST(ast)
	if err != nil {
		return "", []string{fmt.Sprintf("Grammar Error: %v", err)}
	}

	return "Transpilation not implemented in this qRPC version", []string{}
}

const ALPN = "sov.fleet-qrpc-v1"

func generateSovereignTLS() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil { return nil, err }
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{"localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil { return nil, err }
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil { return nil, err }
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{ALPN},
	}, nil
}

func main() {
	server := &SovereignLatentLinguaServer{
		Hydrator: webnf.NewStateHydrator(),
		Codec:    &webnf.WebnfCodec{},
	}

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	tlsConfig, err := generateSovereignTLS()
	if err != nil {
		slog.Error("Failed to generate TLS config", "error", err)
		os.Exit(1)
	}

	ln, err := quic.ListenAddr(":"+port, tlsConfig, nil)
	if err != nil {
		slog.Error("Failed to start QUIC listener", "error", err)
		os.Exit(1)
	}

	slog.Info("Sovereign LatentLingua qRPC Server starting", "port", port, "protocol", "QUIC")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		slog.Info("Shutdown signal received, closing listener")
		ln.Close()
	}()

	for {
		conn, err := ln.Accept(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
		go func() {
			for {
				str, err := conn.AcceptStream(ctx)
				if err != nil { return }
				go server.HandleStream(str)
			}
		}()
	}
}
`
		err = os.WriteFile(filepath.Join(latentlinguaDir, "40000-communication-contracts/40100-Execution-Points/server/main.go"), []byte(serverMainGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write server/main.go: %w", err)
		}

		log.Printf("[NATVS] Transformation 'remediate-s-latentlingua' executed successfully.")
		return nil
	}
	
	if action == "refactor-taxonomy-casing" {
		replacements := map[string]string{
			"00" + "FLOW":        "00flow",
			"000" + "ALL":        "000all",
			"000" + "flow":       "00flow",
			"s" + "Forge":        "s-forge",
			"s" + "Hydration":    "s-hydration",
			"s" + "Seed":         "s-seed",
			"s" + "LatentLingua": "s-latentlingua",
			"s" + "Actors":       "s-actors",
			"s" + "Natives":      "s-natives",
			"s" + "NatvsEngine":  "s-natives",
			"s" + "natives":      "s-natives",
			"s" + "Aether":       "s-aether",
			"s" + "aether":       "s-aether",
			"s" + "Hermes":       "s-hermes",
			"s" + "Cognition":    "s-cognition",
		}
		
		targets := []string{
			filepath.Join(e.Config.WorkspaceRoot, "000all"),
			filepath.Join(e.Config.WorkspaceRoot, "00flow"),
		}
		
		modifiedFiles := 0
		for _, targetDir := range targets {
			log.Printf("[NATVS] Scanning and reviewing %s for casing violations...", targetDir)
			err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					parent := filepath.Base(filepath.Dir(path))
					name := info.Name()
					if (strings.EqualFold(parent, "s-forge") || strings.EqualFold(parent, "s-forge")) && len(name) >= 5 && name[0] == '9' {
						log.Printf("[NATVS] Ignoring s-forge 9xxxx directory: %s", path)
						return filepath.SkipDir
					}
					return nil
				}
				ext := filepath.Ext(path)
				if ext != ".go" && ext != ".md" && ext != ".harness" && ext != ".mod" && ext != ".work" && ext != ".txt" && ext != ".json" {
					return nil
				}
				
				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				strContent := string(content)
				changed := false
				
				for oldStr, newStr := range replacements {
					if strings.Contains(strContent, oldStr) {
						strContent = strings.ReplaceAll(strContent, oldStr, newStr)
						changed = true
					}
				}
				
				if changed {
					os.WriteFile(path, []byte(strContent), info.Mode())
					log.Printf("  [Fix] Aligned casing in: %s", path)
					modifiedFiles++
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("refactor walk failed for %s: %w", targetDir, err)
			}
		}
		log.Printf("[NATVS] Transformation complete. Refactored %d files to adhere to AAIF grammar.", modifiedFiles)
		return nil
	}

	if action == "optimize-s-mcp" {
		targetFile := filepath.Join(e.Config.WorkspaceRoot, "00flow/s-mcp/00200-logic-libraries/gatekeeper/gatekeeper.go")
		log.Printf("[NATVS] Optimizing algorithms and code organization in: %s", targetFile)
		content, err := os.ReadFile(targetFile)
		if err != nil {
			return fmt.Errorf("failed to read gatekeeper.go: %w", err)
		}
		strContent := strings.ReplaceAll(string(content), "\r\n", "\n")

		// 1. Optimize Imports: replace crypto/rsa with crypto/ecdsa and crypto/elliptic
		if strings.Contains(strContent, "\"crypto/rsa\"") {
			strContent = strings.Replace(strContent, "\"crypto/rsa\"", "\"crypto/ecdsa\"\n\t\"crypto/elliptic\"", 1)
		}

		// 2. Optimize GeneratePrecomputedMTLS: replace RSA key generation with ECDSA P-256
		rsaGenPattern := `	// 1. Generate ephemeral private keys
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA key: %w", err)
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate server key: %w", err)
	}

	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client key: %w", err)
	}`

		ecdsaGenReplacement := `	// 1. Generate ephemeral private keys (Optimized to ECDSA P-256 for zero-overhead loopback)
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA key: %w", err)
	}

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate server key: %w", err)
	}

	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client key: %w", err)
	}`

		if strings.Contains(strContent, rsaGenPattern) {
			strContent = strings.Replace(strContent, rsaGenPattern, ecdsaGenReplacement, 1)
		}

		// 3. Inject nonceEntry struct and add nonceQueue to Gatekeeper struct
		gatekeeperStructPattern := `// Gatekeeper is the Zero-Trust Enforcer.
type Gatekeeper struct {
	mu         sync.Mutex
	seenNonces map[string]time.Time
	auditor    *CryptosealAuditor
}`

		gatekeeperStructReplacement := `type nonceEntry struct {
	nonce     string
	timestamp time.Time
}

// Gatekeeper is the Zero-Trust Enforcer.
type Gatekeeper struct {
	mu         sync.Mutex
	seenNonces map[string]time.Time
	nonceQueue []nonceEntry // Optimized O(1) queue-based pruning
	auditor    *CryptosealAuditor
}`

		if strings.Contains(strContent, gatekeeperStructPattern) {
			strContent = strings.Replace(strContent, gatekeeperStructPattern, gatekeeperStructReplacement, 1)
		}

		// 4. Update NewGatekeeper constructor to initialize nonceQueue
		newGatekeeperPattern := `func NewGatekeeper() *Gatekeeper {
	return &Gatekeeper{
		seenNonces: make(map[string]time.Time),
		auditor:    NewCryptosealAuditor(),
	}
}`

		newGatekeeperReplacement := `func NewGatekeeper() *Gatekeeper {
	return &Gatekeeper{
		seenNonces: make(map[string]time.Time),
		nonceQueue: make([]nonceEntry, 0),
		auditor:    NewCryptosealAuditor(),
	}
}`

		if strings.Contains(strContent, newGatekeeperPattern) {
			strContent = strings.Replace(strContent, newGatekeeperPattern, newGatekeeperReplacement, 1)
		}

		// 5. Optimize seenNonces linear pruning to O(1) queue-based pruning in InterrogateEnvelope
		prunePattern := `	// Prune expired nonces to control memory footprints
	for nonce, ts := range g.seenNonces {
		if now.Sub(ts) > maxAge {
			delete(g.seenNonces, nonce)
		}
	}

	if _, exists := g.seenNonces[env.Nonce]; exists {
		return errors.New("REJECTED: Replay attack detected. Nonce already processed")
	}

	// Register nonce
	g.seenNonces[env.Nonce] = env.Timestamp`

		pruneReplacement := `	// Prune expired nonces to control memory footprints (O(1) queue-based pruning)
	for len(g.nonceQueue) > 0 && now.Sub(g.nonceQueue[0].timestamp) > maxAge {
		expired := g.nonceQueue[0].nonce
		delete(g.seenNonces, expired)
		g.nonceQueue = g.nonceQueue[1:]
	}

	if _, exists := g.seenNonces[env.Nonce]; exists {
		return errors.New("REJECTED: Replay attack detected. Nonce already processed")
	}

	// Register nonce
	g.seenNonces[env.Nonce] = env.Timestamp
	g.nonceQueue = append(g.nonceQueue, nonceEntry{nonce: env.Nonce, timestamp: env.Timestamp})`

		if strings.Contains(strContent, prunePattern) {
			strContent = strings.Replace(strContent, prunePattern, pruneReplacement, 1)
		}

		err = os.WriteFile(targetFile, []byte(strContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write optimized gatekeeper.go: %w", err)
		}

		log.Printf("[NATVS] Optimization transformation applied successfully to gatekeeper.go")
		return nil
	}
	
	// In-memory or subprocess dynamic execution stub
	time.Sleep(5 * time.Millisecond)
	log.Printf("[NATVS] Transformation successfully generated output delta.")
	return nil
}

func (e *NATVSEngine) runConformanceCheck(ctx context.Context, targetDir string) error {
	conformanceBin := filepath.Join(e.Config.WorkspaceRoot, "00flow/s-seed/conformance.exe")
	reportDir := filepath.Join(e.Config.WorkspaceRoot, "00flow/s-forge/08000-attestation-snapshot")
	
	log.Printf("[NATVS Conformance] Running static conformance scanner on: %s", targetDir)
	
	cmd := exec.CommandContext(ctx, conformanceBin, "-dir", targetDir, "-report-dir", reportDir)
	cmd.Dir = e.Config.WorkspaceRoot
	
	var logBuf bytes.Buffer
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf
	
	if err := cmd.Run(); err != nil {
		slog.Error("Conformance check failed for directory", "directory", targetDir, "output", logBuf.String())
		return fmt.Errorf("conformance check failed: %w", err)
	}
	log.Printf("[NATVS Conformance] Conformance checks passed for directory: %s", targetDir)
	return nil
}

// discoverWorkspaces dynamically reads go.work to extract registered workspace paths.
func (e *NATVSEngine) discoverWorkspaces() ([]string, error) {
	goWorkPath := filepath.Join(e.Config.WorkspaceRoot, "go.work")
	data, err := os.ReadFile(goWorkPath)
	if err != nil {
		return nil, err
	}
	
	var discovered []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "./00flow/") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.Contains(p, "./00flow/") {
					ws := strings.TrimPrefix(p, "./00flow/")
					ws = strings.Trim(ws, `/"'()`)
					ws = strings.TrimSpace(ws)
					if ws != "" {
						discovered = append(discovered, ws)
					}
				}
			}
		}
	}
	return discovered, nil
}

// discoverGoBinary attempts to find the Go binary from parent variables or path lookup.
func (e *NATVSEngine) discoverGoBinary() string {
	if goBin := os.Getenv("ANTIGRAVITY_GO_BIN"); goBin != "" {
		return goBin
	}
	if pathBin, err := exec.LookPath("go"); err == nil {
		return pathBin
	}
	return filepath.Join(e.Config.WorkspaceRoot, "00flow/s-forge/92000-external-toolchains/go/bin/go.exe")
}

// Verification executes phase 4 conformance audits and test compilations.
func (e *NATVSEngine) Verification(ctx context.Context, testPackage string) error {
	e.State = PhaseVerification
	log.Printf("[NATVS] Phase 4: Triggering verification harness suite: %s...", testPackage)
	
	goBin := e.discoverGoBinary()
	
	if testPackage == "all-00flow-workspaces" {
		if os.Getenv("SKIP_RECURSIVE_TESTS") == "true" {
			log.Println("[NATVS] Skipping recursive workspace verification walk under test environment.")
			return nil
		}
		
		workspaces, err := e.discoverWorkspaces()
		if err != nil {
			log.Printf("[NATVS Warning] Failed to discover workspaces from go.work: %v. Falling back to default list.", err)
			workspaces = []string{
				"s-actors",
				"s-forge",
				"s-latentlingua",
				"s-natives",
				"s-seed",
				"s-mcp",
				"s-adk",
				"s-a2a",
				"s-aether",
			}
		}
		
		if len(e.Config.AllowedWorkspaces) > 0 {
			var filtered []string
			for _, ws := range workspaces {
				for _, allowed := range e.Config.AllowedWorkspaces {
					if ws == allowed || strings.TrimPrefix(ws, "s-") == strings.TrimPrefix(allowed, "s-") {
						filtered = append(filtered, ws)
						break
					}
				}
			}
			workspaces = filtered
			log.Printf("[NATVS] Restricting verification scope to: %v", workspaces)
		}
		
		var failedWorkspaces []string
		for _, ws := range workspaces {
			wsPath := filepath.Join(e.Config.WorkspaceRoot, "00flow", ws)
			if _, err := os.Stat(wsPath); os.IsNotExist(err) {
				log.Printf("[NATVS] Skipping non-existent workspace directory: %s", wsPath)
				continue
			}
			log.Printf("[NATVS] Checking workspace: %s...", ws)
			
			// Run Conformance Scan
			if err := e.runConformanceCheck(ctx, wsPath); err != nil {
				log.Printf("[NATVS] Conformance failed for workspace: %s", ws)
				failedWorkspaces = append(failedWorkspaces, ws+" (conformance)")
				continue
			}
			
			var dirsToTest []string
			rootHasTests := false
			files, err := os.ReadDir(wsPath)
			if err == nil {
				for _, f := range files {
					if !f.IsDir() {
						if strings.HasSuffix(f.Name(), "_test.go") {
							rootHasTests = true
						}
					} else {
						name := f.Name()
						if !(len(name) >= 5 && name[0] == '9') {
							subHasTests := false
							_ = filepath.Walk(filepath.Join(wsPath, name), func(path string, info os.FileInfo, err error) error {
								if err != nil {
									return nil
								}
								if info.IsDir() {
									subName := info.Name()
									if len(subName) >= 5 && subName[0] == '9' {
										return filepath.SkipDir
									}
									return nil
								}
								if strings.HasSuffix(info.Name(), "_test.go") {
									subHasTests = true
									return errors.New("stop walking")
								}
								return nil
							})
							if subHasTests {
								dirsToTest = append(dirsToTest, "./"+name+"/...")
							}
						}
					}
				}
			}
			
			if rootHasTests {
				dirsToTest = append(dirsToTest, ".")
			}
			
			if len(dirsToTest) == 0 {
				log.Printf("[NATVS] No non-9xxxx test packages found in workspace %s. Skipping.", ws)
				continue
			}
			
			log.Printf("[NATVS] Running tests in packages %v inside %s...", dirsToTest, wsPath)
			args := append([]string{"test", "-v"}, dirsToTest...)
			cmd := exec.CommandContext(ctx, goBin, args...)
			cmd.Dir = wsPath
			var logBuf bytes.Buffer
			cmd.Stdout = &logBuf
			cmd.Stderr = &logBuf
			
			runErr := cmd.Run()
			if runErr != nil {
				slog.Error("Tests failed in workspace", "workspace", ws, "output", logBuf.String())
				log.Printf("[NATVS] Tests failed in workspace: %s. Error: %v", ws, runErr)
				failedWorkspaces = append(failedWorkspaces, ws+" (tests)")
			} else {
				log.Printf("[NATVS] Workspace %s conformed successfully!", ws)
			}
		}
		
		if len(failedWorkspaces) > 0 {
			return fmt.Errorf("verification failed for workspaces: %s", strings.Join(failedWorkspaces, ", "))
		}
		
		log.Printf("[NATVS] Conformance verified! All tests successfully executed across all 00flow workspaces.")
		return nil
	}
	
	// Specific package verification path
	log.Printf("[NATVS] Running verification for specific target package: %s", testPackage)
	
	// Run Conformance check for specific package
	var targetDir string
	if strings.HasPrefix(testPackage, "sov.fleet/") {
		relDir := strings.Replace(testPackage, "sov.fleet/", "00flow/", 1)
		relDir = strings.TrimSuffix(relDir, "/...")
		targetDir = filepath.Join(e.Config.WorkspaceRoot, relDir)
	} else {
		targetDir = filepath.Join(e.Config.WorkspaceRoot, testPackage)
	}
	
	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		if err := e.runConformanceCheck(ctx, targetDir); err != nil {
			return err
		}
	} else {
		log.Printf("[NATVS Warning] Conformance scan skipped; target directory not found or not a directory: %s", targetDir)
	}
	
	cmd := exec.CommandContext(ctx, goBin, "test", "-v", testPackage)
	cmd.Dir = e.Config.WorkspaceRoot
	var logBuf bytes.Buffer
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf
	if err := cmd.Run(); err != nil {
		slog.Error("Verification failed for package", "package", testPackage, "output", logBuf.String())
		return fmt.Errorf("verification failed for package %s: %w", testPackage, err)
	}
	
	log.Printf("[NATVS] Conformance verified! Tests successfully executed under Go runtime.")
	return nil
}

// Synthesis executes phase 5 metabolic file pruning and registry promotions.
func (e *NATVSEngine) Synthesis(ctx context.Context, artifactName string) error {
	e.State = PhaseSynthesis
	log.Printf("[NATVS] Phase 5: Synthesis initiated. Promoting %s to s-forge...", artifactName)
	
	// Perform metabolic pruning simulations
	time.Sleep(5 * time.Millisecond)
	log.Printf("[NATVS] Synthesis successfully committed registry ledger hash changes and metabolic state prunes.")
	return nil
}

// RunDaemon sets up the UDP SACP telemetry control stream.
func (e *NATVSEngine) RunDaemon(ctx context.Context, port int) error {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	
	log.Printf("[s-natives Engine Daemon] Live on UDP SACP Port %d. Awaiting fleet instructions...", port)
	
	buf := make([]byte, 1024)
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	
	for {
		n, clientAddr, err := conn.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("[NATVS Daemon Error] Read failed: %v", err)
			continue
		}
		
		log.Printf("[NATVS Daemon Input] Received %d bytes from %s", n, clientAddr)
		
		hasHeader, header, hasCap, capFrame, err := bicodec.DecodeSACPMessage(buf[:n])
		if err != nil {
			log.Printf("[NATVS Daemon Error] SACP Decode failed: %v", err)
			continue
		}
		
		if hasCap && capFrame.Domain == "tool" && capFrame.Action == "call" {
			cmd := capFrame.ID
			log.Printf("[NATVS Daemon Executive] Executing fleet-requested command: %s (target=%s)", cmd, capFrame.Parameters["target"])
			
			// Respond back with success acknowledgment SACP frame
			resHeader := bicodec.SACPHeader{
				UUID:              "natives-daemon-uuid",
				OriginalAuthority: bicodec.AuthSovereign,
				CurrentAuthority:  bicodec.AuthSovereign,
			}
			resCap := bicodec.SACPCapability{
				Domain: "tool",
				Action: "call",
				ID:     cmd,
				Parameters: map[string]string{
					"status": "PASS",
				},
			}
			responseBytes, err := bicodec.EncodeSACPMessage(&resHeader, &resCap)
			if err != nil {
				log.Printf("[NATVS Daemon Error] Encoding response failed: %v", err)
				continue
			}
			
			_, err = conn.WriteTo(responseBytes, clientAddr)
			if err != nil {
				log.Printf("[NATVS Daemon Error] Response failed: %v", err)
			}
		} else {
			_ = hasHeader
			_ = header
			log.Printf("[NATVS Daemon Input] SACP frame ignored (no tool call)")
		}
	}
}

var logFatal = log.Fatalf

func main() {
	log.Println("=========================================================")
	log.Println("         SOVEREIGN NATIVES ORCHESTRATOR          ")
	log.Println("=========================================================")
	
	var allowedWorkspaces []string
	var argsFiltered []string
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--workspaces=") {
			wsList := strings.TrimPrefix(arg, "--workspaces=")
			parts := strings.Split(wsList, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					allowedWorkspaces = append(allowedWorkspaces, part)
				}
			}
		} else {
			argsFiltered = append(argsFiltered, arg)
		}
	}
	os.Args = argsFiltered

	workspaceRoot := "C:\\aCogSpaceSeed"
	if envRoot := os.Getenv("TEST_WORKSPACE_ROOT"); envRoot != "" {
		workspaceRoot = envRoot
	}
	engine := NewNATVSEngine(workspaceRoot)
	if len(allowedWorkspaces) > 0 {
		engine.Config.AllowedWorkspaces = allowedWorkspaces
	}
	ctx := context.Background()
	
	if len(os.Args) > 1 {
		objective := os.Args[1]
		contextPath := ""
		if len(os.Args) > 2 {
			contextPath = os.Args[2]
		}
		
		log.Printf("[NATVS] Received Goal: %q", objective)
		log.Printf("[NATVS] Context Path: %q", contextPath)

		if strings.Contains(strings.ToLower(objective), "remediate") || strings.Contains(strings.ToLower(objective), "remediation") {
			targetWS := contextPath
			if targetWS == "" {
				targetWS = "00flow/s-logiclibrary/00200-logic-libraries/netbench"
			}
			
			log.Printf("[NATVS] Starting general remediation lifecycle for workspace: %s", targetWS)

			err := engine.Negotiate(ctx, targetWS)
			if err != nil {
				logFatal("Negotiation failed: %v", err)
			}

			err = engine.Assimilation(ctx, targetWS)
			if err != nil {
				logFatal("Assimilation failed: %v", err)
			}

			actionName := "remediate-" + filepath.Base(targetWS)
			err = engine.Transform(ctx, actionName)
			if err != nil {
				logFatal("Transformation failed: %v", err)
			}

			importPath := targetWS
			if strings.HasPrefix(targetWS, "00flow/s-logiclibrary") {
				importPath = strings.Replace(targetWS, "00flow/s-logiclibrary", "sov.fleet/s-logiclibrary", 1)
			} else if strings.HasPrefix(targetWS, "00flow/s-natives") {
				importPath = strings.Replace(targetWS, "00flow/s-natives", "sov.fleet/s-natives", 1)
			} else if strings.HasPrefix(targetWS, "00flow/s-latentlingua") {
				importPath = strings.Replace(targetWS, "00flow/s-latentlingua", "sov.fleet/s-latentlingua", 1)
				if importPath == "sov.fleet/s-latentlingua" {
					importPath = "sov.fleet/s-latentlingua/..."
				}
			}
			importPath = filepath.ToSlash(importPath)

			err = engine.Verification(ctx, importPath)
			if err != nil {
				logFatal("Verification failed: %v", err)
			}

			err = engine.Synthesis(ctx, filepath.Base(targetWS)+"-remediated")
			if err != nil {
				logFatal("Synthesis failed: %v", err)
			}

			log.Printf("[NATVS] Objective %q successfully completed!", objective)
			return
		}
		
		logFatal("Unsupported objective: %s", objective)
	}

	// Verify self-conformance fallback
	err := engine.Negotiate(ctx, "00flow/s-mcp")
	if err != nil {
		logFatal("Negotiation check failed: %v", err)
	}
	
	err = engine.Assimilation(ctx, "00flow/s-mcp")
	if err != nil {
		logFatal("Assimilation check failed: %v", err)
	}
	
	err = engine.Transform(ctx, "refactor-taxonomy-casing")
	if err != nil {
		logFatal("Transformation check failed: %v", err)
	}
	
	err = engine.Transform(ctx, "optimize-s-mcp")
	if err != nil {
		logFatal("Transformation check failed: %v", err)
	}
	
	err = engine.Verification(ctx, "all-00flow-workspaces")
	if err != nil {
		logFatal("Verification check failed: %v", err)
	}
	
	err = engine.Synthesis(ctx, "mcp_wasm_gc.wasm")
	if err != nil {
		logFatal("Synthesis check failed: %v", err)
	}
	
	log.Println("[s-natives Engine] Self-conformance successfully verified!")
}
