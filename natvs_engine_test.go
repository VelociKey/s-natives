package main

import (
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sov.fleet/s-logiclibrary/00200-logic-libraries/bicodec"
)

func TestNATVSComprehensive(t *testing.T) {
	// Create hermetic temp workspace
	tempDir := t.TempDir()
	
	// Create required directories
	dirs := []string{
		filepath.Join(tempDir, "000all/s-cognition"),
		filepath.Join(tempDir, "00flow/s-forge/90000-authority"),
		filepath.Join(tempDir, "00flow/s-forge/90100-rehydration-seed"),
		filepath.Join(tempDir, "00flow/s-forge/80200-rehydration-seed"),
	}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
	}

	workspaces := []string{
		"s-aether",
		"s-actors",
		"s-forge",
		"s-latentlingua",
		"s-natives",
		"s-seed",
		"s-mcp",
		"s-adk",
		"s-a2a",
	}
	for _, ws := range workspaces {
		wsDir := filepath.Join(tempDir, "00flow", ws)
		err := os.MkdirAll(wsDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create workspace directory %s: %v", ws, err)
		}
		// Write dummy go.mod
		err = os.WriteFile(filepath.Join(wsDir, "go.mod"), []byte("module sov.fleet/"+ws+"\ngo 1.26.3\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to write dummy go.mod for %s: %v", ws, err)
		}
	}

	// Copy conformance.exe to tempDir/00flow/s-seed/conformance.exe
	srcConf := "C:\\aCogSpaceSeed\\00flow\\s-seed\\conformance.exe"
	destConf := filepath.Join(tempDir, "00flow/s-seed/conformance.exe")
	errCopy := copyFile(srcConf, destConf)
	if errCopy != nil {
		t.Fatalf("Failed to copy conformance.exe: %v", errCopy)
	}
	
	gkFile := writeMockGatekeeper(t, tempDir)
	
	// Write dummy files for casing refactoring tests
	dummyGoFile := filepath.Join(tempDir, "00flow/s-forge/80200-rehydration-seed/test.go")
	err := os.WriteFile(dummyGoFile, []byte("package main\nimport \"s" + "Seed\"\n// s" + "Forge and s" + "Natives"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy Go file: %v", err)
	}
	
	dummyIgnoredFile := filepath.Join(tempDir, "00flow/s-forge/90100-rehydration-seed/ignored.txt")
	err = os.WriteFile(dummyIgnoredFile, []byte("s" + "Seed and s" + "Forge"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy ignored file: %v", err)
	}
	
	dummyMDFile := filepath.Join(tempDir, "000all/s-cognition/test.md")
	err = os.WriteFile(dummyMDFile, []byte("# s" + "Natives and s" + "Aether"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy MD file: %v", err)
	}
	
	engine := NewNATVSEngine(tempDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// 1. Negotiation Phase
	err = engine.Negotiate(ctx, "00flow/s-mcp")
	if err != nil {
		t.Fatalf("Negotiation phase failed: %v", err)
	}
	if engine.State != PhaseNegotiate {
		t.Errorf("Expected state %s, got %s", PhaseNegotiate, engine.State)
	}
	
	// 2. Assimilation Phase
	// (a) Valid workspace
	err = engine.Assimilation(ctx, "00flow/s-forge")
	if err != nil {
		t.Fatalf("Assimilation phase failed: %v", err)
	}
	if engine.State != PhaseAssimilation {
		t.Errorf("Expected state %s, got %s", PhaseAssimilation, engine.State)
	}
	// (b) Non-existent workspace (error check)
	err = engine.Assimilation(ctx, "00flow/nonexistent")
	if err == nil {
		t.Errorf("Expected assimilation failure for non-existent workspace")
	}
	
	// 3. Transformation Phase (Refactoring)
	err = engine.Transform(ctx, "refactor-taxonomy-casing")
	if err != nil {
		t.Fatalf("Transformation refactoring failed: %v", err)
	}
	
	// Verify that conformed Go file was refactored
	conformedContent, err := os.ReadFile(dummyGoFile)
	if err != nil {
		t.Fatalf("Failed to read conformed file: %v", err)
	}
	strContent := string(conformedContent)
	if strings.Contains(strContent, "s" + "Seed") || strings.Contains(strContent, "s" + "Forge") || strings.Contains(strContent, "s" + "Natives") {
		t.Errorf("Conformed Go file was not refactored correctly: %q", strContent)
	}
	
	// Verify that conformed MD file was refactored
	conformedMDContent, err := os.ReadFile(dummyMDFile)
	if err != nil {
		t.Fatalf("Failed to read conformed MD file: %v", err)
	}
	strMD := string(conformedMDContent)
	if strings.Contains(strMD, "s" + "Natives") || strings.Contains(strMD, "s" + "Aether") {
		t.Errorf("Conformed MD file was not refactored correctly: %q", strMD)
	}
	
	// Verify that ignored file in 9xxxx directory was NOT modified
	ignoredContent, err := os.ReadFile(dummyIgnoredFile)
	if err != nil {
		t.Fatalf("Failed to read ignored file: %v", err)
	}
	strIgnored := string(ignoredContent)
	if !strings.Contains(strIgnored, "s" + "Seed") {
		t.Errorf("Ignored 9xxxx file was modified: %q", strIgnored)
	}
	
	// Verify that optimize-s-mcp refactoring works
	err = engine.Transform(ctx, "optimize-s-mcp")
	if err != nil {
		t.Fatalf("Transformation optimize-s-mcp failed: %v", err)
	}

	optContent, err := os.ReadFile(gkFile)
	if err != nil {
		t.Fatalf("Failed to read conformed gatekeeper: %v", err)
	}
	optStr := string(optContent)
	if !strings.Contains(optStr, "ecdsa.GenerateKey") {
		t.Error("Expected conformed gatekeeper to contain ECDSA key generation")
	}
	if !strings.Contains(optStr, "nonceQueue []nonceEntry") {
		t.Error("Expected conformed gatekeeper to track nonceQueue")
	}

	// Check coverage of Transformation's unrecognized action fallback
	err = engine.Transform(ctx, "unknown-action")
	if err != nil {
		t.Fatalf("Transformation fallback failed: %v", err)
	}
	
	// 4. Verification Phase
	// All workspaces package walk (no test files case)
	err = engine.Verification(ctx, "all-00flow-workspaces")
	if err != nil {
		t.Fatalf("Verification phase all workspaces failed: %v", err)
	}
	
	// 5. Synthesis Phase
	err = engine.Synthesis(ctx, "aether_wasm_gc.wasm")
	if err != nil {
		t.Fatalf("Synthesis phase failed: %v", err)
	}
	if engine.State != PhaseSynthesis {
		t.Errorf("Expected state %s, got %s", PhaseSynthesis, engine.State)
	}
}

func TestNATVSListenDaemonAndSACP(t *testing.T) {
	engine := NewNATVSEngine("C:\\aCogSpaceSeed")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	port := 51989
	
	// Start UDP server daemon
	go func() {
		_ = engine.RunDaemon(ctx, port)
	}()
	
	// Allow UDP socket start
	time.Sleep(100 * time.Millisecond)
	
	// Dial UDP client port
	raddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:51989")
	if err != nil {
		t.Fatalf("Failed to resolve RAddr: %v", err)
	}
	
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		t.Fatalf("Failed to dial UDP server: %v", err)
	}
	defer conn.Close()
	
	// Dispatch SACP command frame (constructed via bicodec)
	h := bicodec.SACPHeader{
		UUID:              "test-client-uuid",
		OriginalAuthority: bicodec.AuthSovereign,
		CurrentAuthority:  bicodec.AuthSovereign,
	}
	c := bicodec.SACPCapability{
		Domain: "tool",
		Action: "call",
		ID:     "orchestrate-workspace-migration",
		Parameters: map[string]string{
			"target": "s-aether",
		},
	}
	msgBytes, err := bicodec.EncodeSACPMessage(&h, &c)
	if err != nil {
		t.Fatalf("Failed to encode SACP message: %v", err)
	}
	
	_, err = conn.Write(msgBytes)
	if err != nil {
		t.Fatalf("Failed to write to UDP socket: %v", err)
	}
	
	// Read Response SACP frame
	buf := make([]byte, 1024)
	err = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}
	
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from UDP socket: %v", err)
	}
	
	hasHeader, _, hasCap, resC, err := bicodec.DecodeSACPMessage(buf[:n])
	if err != nil {
		t.Fatalf("Failed to decode UDP response: %v", err)
	}
	
	if !hasCap {
		t.Errorf("Expected capability frame in response, got none (hasHeader=%v)", hasHeader)
	} else {
		if resC.ID != "orchestrate-workspace-migration" {
			t.Errorf("Expected capability ID 'orchestrate-workspace-migration', got: %q", resC.ID)
		}
		if resC.Parameters["status"] != "PASS" {
			t.Errorf("Expected status 'PASS', got: %q", resC.Parameters["status"])
		}
	}
	
	// Test sending invalid SACP frame to server to cover fallback paths
	_, _ = conn.Write([]byte("INVALID_FRAME"))
	time.Sleep(50 * time.Millisecond)
	
	// Cancel context to shut down RunDaemon cleanly
	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestMainFunc(t *testing.T) {
	t.Setenv("SKIP_RECURSIVE_TESTS", "true")
	
	// Save and restore os.Args so main() doesn't interpret test flags as CLI goals
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"natvs-engine"}
	
	// Create hermetic temp workspace for main() self-conformance checks
	tempDir := t.TempDir()
	dirs := []string{
		filepath.Join(tempDir, "000all/s-cognition"),
		filepath.Join(tempDir, "00flow/s-aether"),
		filepath.Join(tempDir, "00flow/s-forge/90100-rehydration-seed"),
		filepath.Join(tempDir, "00flow/s-mcp/00200-logic-libraries/gatekeeper"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create temp directory for main check: %v", err)
		}
	}
	
	_ = writeMockGatekeeper(t, tempDir)
	
	t.Setenv("TEST_WORKSPACE_ROOT", tempDir)
	main()
}

func TestNATVSVerificationFailure(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create required directories
	dirs := []string{
		filepath.Join(tempDir, "00flow/s-aether"),
	}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
	}
	
	// Create a dummy test file in s-aether to make it look like it has tests
	dummyTestGoFile := filepath.Join(tempDir, "00flow/s-aether/dummy_test.go")
	err := os.WriteFile(dummyTestGoFile, []byte("package main\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy test file: %v", err)
	}
	
	engine := NewNATVSEngine(tempDir)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	// Ensure SKIP_RECURSIVE_TESTS is NOT set
	t.Setenv("SKIP_RECURSIVE_TESTS", "")
	
	err = engine.Verification(ctx, "all-00flow-workspaces")
	if err == nil {
		t.Errorf("Expected Verification to fail because of missing/invalid goBin, but it passed")
	}
}

func TestNATVSDaemonListenFailure(t *testing.T) {
	engine := NewNATVSEngine(t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	// Bind to the port first
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:51999")
	if err != nil {
		t.Fatalf("Failed to resolve: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer conn.Close()
	
	// Try to run RunDaemon on the same port, which must fail!
	err = engine.RunDaemon(ctx, 51999)
	if err == nil {
		t.Errorf("Expected RunDaemon to fail when port is already in use, but it passed")
	}
}

func TestMainFuncFailure(t *testing.T) {
	// Mock logFatal
	oldFatal := logFatal
	defer func() { logFatal = oldFatal }()
	
	// Save and restore os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"natvs-engine"}
	
	fatalCalled := 0
	logFatal = func(format string, v ...interface{}) {
		fatalCalled++
	}
	
	// Force failure by running with invalid workspace root
	t.Setenv("TEST_WORKSPACE_ROOT", "/nonexistent_root_dir_abc")
	t.Setenv("SKIP_RECURSIVE_TESTS", "true")
	
	main()
	
	if fatalCalled == 0 {
		t.Errorf("Expected logFatal to be called, but it was not")
	}
}

func writeMockGatekeeper(t *testing.T, tempDir string) string {
	dir := filepath.Join(tempDir, "00flow/s-mcp/00200-logic-libraries/gatekeeper")
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		t.Fatalf("Failed to create mock gatekeeper dir: %v", err)
	}
	path := filepath.Join(dir, "gatekeeper.go")
	content := `package gatekeeper
import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)
// Gatekeeper is the Zero-Trust Enforcer.
type Gatekeeper struct {
	mu         sync.Mutex
	seenNonces map[string]time.Time
	auditor    *CryptosealAuditor
}

func NewGatekeeper() *Gatekeeper {
	return &Gatekeeper{
		seenNonces: make(map[string]time.Time),
		auditor:    NewCryptosealAuditor(),
	}
}

func (g *Gatekeeper) InterrogateEnvelope(env SignedEnvelope, secret []byte, maxAge time.Duration) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Prune expired nonces to control memory footprints
	for nonce, ts := range g.seenNonces {
		if now.Sub(ts) > maxAge {
			delete(g.seenNonces, nonce)
		}
	}

	if _, exists := g.seenNonces[env.Nonce]; exists {
		return errors.New("REJECTED: Replay attack detected. Nonce already processed")
	}

	// Register nonce
	g.seenNonces[env.Nonce] = env.Timestamp
	return nil
}

func GeneratePrecomputedMTLS() (*PrecomputedTLSConfig, error) {
	// 1. Generate ephemeral private keys
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
	}
	return nil, nil
}
`
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock gatekeeper file: %v", err)
	}
	return path
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

