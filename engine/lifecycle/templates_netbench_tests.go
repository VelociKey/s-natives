package lifecycle

const NetbenchTestGo = `package netbench

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
