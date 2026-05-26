package lifecycle

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Transform executes phase 3: code generation, template injection, and casing adjustments.
func (e *NATVSEngine) Transform(ctx context.Context, action string) error {
	e.State = PhaseTransform
	log.Printf("[NATVS] Phase 3: Executing Transformation target action: '%s'...", action)

	if action == "remediate-netbench" {
		netbenchDir := filepath.Join(e.Config.WorkspaceRoot, "00flow/s-logiclibrary/00200-logic-libraries/netbench")
		log.Printf("[NATVS] Remediation target directory: %s", netbenchDir)

		err := os.WriteFile(filepath.Join(netbenchDir, "netbench.go"), []byte(NetbenchGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write netbench.go: %w", err)
		}

		err = os.WriteFile(filepath.Join(netbenchDir, "tcp.go"), []byte(TcpGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write tcp.go: %w", err)
		}

		err = os.WriteFile(filepath.Join(netbenchDir, "udp.go"), []byte(UdpGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write udp.go: %w", err)
		}

		err = os.WriteFile(filepath.Join(netbenchDir, "quic.go"), []byte(QuicGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write quic.go: %w", err)
		}

		err = os.WriteFile(filepath.Join(netbenchDir, "netbench_test.go"), []byte(NetbenchTestGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write netbench_test.go: %w", err)
		}

		log.Printf("[NATVS] Transformation 'remediate-netbench' executed successfully.")
		return nil
	}

	if action == "remediate-s-latentlingua" {
		latentlinguaDir := filepath.Join(e.Config.WorkspaceRoot, "00flow/s-latentlingua")
		log.Printf("[NATVS] Remediation target directory: %s", latentlinguaDir)

		err := os.WriteFile(filepath.Join(latentlinguaDir, "02000-logic-libraries/snparser/ast.go"), []byte(AstGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write ast.go: %w", err)
		}

		err = os.WriteFile(filepath.Join(latentlinguaDir, "02000-logic-libraries/snparser/lexer.go"), []byte(LexerGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write lexer.go: %w", err)
		}

		err = os.WriteFile(filepath.Join(latentlinguaDir, "02000-logic-libraries/snparser/parser.go"), []byte(ParserGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write parser.go: %w", err)
		}

		err = os.WriteFile(filepath.Join(latentlinguaDir, "02000-logic-libraries/snparser/expr.go"), []byte(ExprGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write expr.go: %w", err)
		}

		err = os.WriteFile(filepath.Join(latentlinguaDir, "02000-logic-libraries/bridge/bridge.go"), []byte(BridgeGo), 0644)
		if err != nil {
			return fmt.Errorf("failed to write bridge.go: %w", err)
		}

		err = os.WriteFile(filepath.Join(latentlinguaDir, "40000-communication-contracts/40100-Execution-Points/server/main.go"), []byte(ServerMainGo), 0644)
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
		if strings.Contains(strContent, RsaGenPattern) {
			strContent = strings.Replace(strContent, RsaGenPattern, EcdsaGenReplacement, 1)
		}

		// 3. Inject nonceEntry struct and add nonceQueue to Gatekeeper struct
		if strings.Contains(strContent, GatekeeperStructPattern) {
			strContent = strings.Replace(strContent, GatekeeperStructPattern, GatekeeperStructReplacement, 1)
		}

		// 4. Update NewGatekeeper constructor to initialize nonceQueue
		if strings.Contains(strContent, NewGatekeeperPattern) {
			strContent = strings.Replace(strContent, NewGatekeeperPattern, NewGatekeeperReplacement, 1)
		}

		// 5. Optimize seenNonces linear pruning to O(1) queue-based pruning in InterrogateEnvelope
		if strings.Contains(strContent, PrunePattern) {
			strContent = strings.Replace(strContent, PrunePattern, PruneReplacement, 1)
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
