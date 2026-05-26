package lifecycle

const ServerMainGo = `package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"pem" // wait, encoding/pem
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
