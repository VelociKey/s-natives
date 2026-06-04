package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sov.fleet/s-natives/89000-internal-actors-source/engine/lifecycle"
)

func runJulesStatusCheck(engine *NATVSEngine) error {
	workspaceRoot := engine.Config.WorkspaceRoot
	julesPath := filepath.Clean(filepath.Join(workspaceRoot, "00flow", "s-forge", "94000-external-actors", "jules", "jules"+lifecycle.GetExeSuffix()))

	cmdSession := exec.Command(julesPath, "remote", "list", "--session")
	cmdSession.Env = lifecycle.BuildSandboxEnv(workspaceRoot, "")
	sessionBytes, err := cmdSession.CombinedOutput()
	sessionStr := "none"
	var whoamiStr string
	if err == nil {
		sessionStr = strings.TrimSpace(string(sessionBytes))
		whoamiStr = "Active (Cached in s-forge sandbox)"
	} else {
		sessionStr = strings.TrimSpace(string(sessionBytes))
		if sessionStr == "" {
			sessionStr = "none"
		}
		whoamiStr = "None (Run jules auth login)"
	}

	envLabel := "The Outside World (Decoupled, Flat-Rate Subscription Track)"

	activeSessionsCount := 0
	if err == nil {
		lines := strings.Split(sessionStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.Contains(line, "ID") && !strings.Contains(line, "---") && !strings.Contains(line, "Description") {
				activeSessionsCount++
			}
		}
	}

	pendingQueueCount := 0
	queuePath := filepath.Join(workspaceRoot, "c0990-ephemeral-scratch/natvs coordination/tasks.queue.webnf")
	if queueTasks, queueErr := ParseQueueFileAtomic(queuePath); queueErr == nil {
		for _, t := range queueTasks {
			if t.State == StateCreated || t.State == StateInProgress {
				pendingQueueCount++
			}
		}
	}

	remainingTasks := 10 - activeSessionsCount - pendingQueueCount
	if remainingTasks < 0 {
		remainingTasks = 0
	}

	email := strings.ToLower(getSandboxedEmail(workspaceRoot))
	tierPath := filepath.Join(workspaceRoot, "00flow/s-forge/94000-external-actors/jules/c1000-credentials/tier.txt")
	var savedTier string
	if tierData, tierErr := os.ReadFile(tierPath); tierErr == nil {
		savedTier = strings.TrimSpace(strings.ToLower(string(tierData)))
	}

	isPro := strings.Contains(email, "velocikey") || savedTier == "pro" || savedTier == "ai pro"

	var remainingTasksStr string
	if isPro {
		remainingTasksStr = "Unlimited (AI Pro Tier)"
	} else {
		remainingTasksStr = fmt.Sprintf("%d (Included Tier)", remainingTasks)
	}

	output := "=========================================================\n" +
		"             JULES AUTONOMOUS ENGINE STATUS              \n" +
		"=========================================================\n" +
		"Active Credentials  : " + whoamiStr + "\n" +
		"Whoami              : " + getSandboxedEmail(workspaceRoot) + "\n" +
		"Remaining Tasks     : " + remainingTasksStr + "\n" +
		"Session Tracking    : " + sessionStr + "\n" +
		"Execution Track     : " + envLabel + "\n" +
		"=========================================================\n"
	if _, err := os.Stdout.Write([]byte(output)); err != nil {
		slog.Error("Failed to write status to stdout", "error", err)
	}

	return nil
}

func runJulesLogin(engine *NATVSEngine) error {
	workspaceRoot := engine.Config.WorkspaceRoot
	julesPath := filepath.Clean(filepath.Join(workspaceRoot, "00flow", "s-forge", "94000-external-actors", "jules", "jules"+lifecycle.GetExeSuffix()))

	sandboxHome := filepath.Join(workspaceRoot, "00flow", "s-forge", "94000-external-actors", "jules", "c1000-credentials")
	if errDel := os.RemoveAll(sandboxHome); errDel != nil {
		slog.Warn("Failed to clean up sandbox home credentials directory", "path", sandboxHome, "error", errDel)
	}
	if errMk := os.MkdirAll(sandboxHome, 0755); errMk != nil {
		return fmt.Errorf("failed to create sandbox credentials directory: %w", errMk)
	}

	var emailArg string
	var tierArg string
	for i, arg := range os.Args {
		if (arg == "--email" || arg == "-e") && i+1 < len(os.Args) {
			emailArg = os.Args[i+1]
		}
		if (arg == "--tier" || arg == "-t") && i+1 < len(os.Args) {
			tierArg = os.Args[i+1]
		}
	}

	cmd := exec.Command(julesPath, "login")
	cmd.Env = lifecycle.BuildSandboxEnv(workspaceRoot, "")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	whoamiDir := filepath.Join(workspaceRoot, "00flow/s-forge/94000-external-actors/jules/c1000-credentials")
	if errMkDir := os.MkdirAll(whoamiDir, 0755); errMkDir != nil {
		slog.Warn("Failed to create whoami directory", "path", whoamiDir, "error", errMkDir)
	}
	whoamiPath := filepath.Join(whoamiDir, "whoami.txt")
	tierPath := filepath.Join(whoamiDir, "tier.txt")

	var email string
	if emailArg != "" {
		email = emailArg
		if errWrite := os.WriteFile(whoamiPath, []byte(email), 0644); errWrite != nil {
			slog.Error("Failed to write whoami.txt", "path", whoamiPath, "error", errWrite)
		}
	} else {
		if _, err := os.Stdout.Write([]byte("Enter the Google email address used for this login: ")); err != nil {
			slog.Error("Failed to write to stdout", "error", err)
		}
		reader := bufio.NewReader(os.Stdin)
		emailInput, err := reader.ReadString('\n')
		if err == nil {
			email = strings.TrimSpace(emailInput)
			if email != "" {
				if errWrite := os.WriteFile(whoamiPath, []byte(email), 0644); errWrite != nil {
					slog.Error("Failed to write whoami.txt", "path", whoamiPath, "error", errWrite)
				}
			}
		}
	}

	if tierArg != "" {
		if errWrite := os.WriteFile(tierPath, []byte(strings.ToLower(strings.TrimSpace(tierArg))), 0644); errWrite != nil {
			slog.Error("Failed to write tier.txt", "path", tierPath, "error", errWrite)
		}
	} else if strings.Contains(strings.ToLower(email), "velocikey") {
		if errWrite := os.WriteFile(tierPath, []byte("pro"), 0644); errWrite != nil {
			slog.Error("Failed to write tier.txt", "path", tierPath, "error", errWrite)
		}
	} else {
		if _, err := os.Stdout.Write([]byte("Enter your tier (pro/free) [default: free]: ")); err != nil {
			slog.Error("Failed to write to stdout", "error", err)
		}
		reader := bufio.NewReader(os.Stdin)
		tierInput, err := reader.ReadString('\n')
		if err == nil {
			t := strings.TrimSpace(strings.ToLower(tierInput))
			if t == "" {
				t = "free"
			}
			if errWrite := os.WriteFile(tierPath, []byte(t), 0644); errWrite != nil {
				slog.Error("Failed to write tier.txt", "path", tierPath, "error", errWrite)
			}
		}
	}

	return nil
}

func getSandboxedEmail(workspaceRoot string) string {
	whoamiPath := filepath.Join(workspaceRoot, "00flow/s-forge/94000-external-actors/jules/c1000-credentials/whoami.txt")
	data, err := os.ReadFile(whoamiPath)
	if err == nil {
		return strings.TrimSpace(string(data))
	}
	return "Unknown (Run jules auth login to set)"
}

