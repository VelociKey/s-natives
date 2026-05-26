# Workspace: s-natives

This workspace follows the 8xxxx Semantic Taxonomy defined in s-seed/00200-Workspace-Taxonomy/taxonomy.txt.

## Engine Usage

The NATVS Engine is a sovereign orchestrator that manages the fleet-aware execution lifecycle.

### Antigravity Agent Manager
- **Discovery:** The Agent Manager automatically discovers the engine via the global `AGENTS.md` registry.
- **Integration:** Utilizes SACP (Sovereign Agent Control Protocol) over UDP for telemetry and command dispatch.
- **Daemon Mode:** The engine can run as a background daemon listening for fleet instructions on UDP port `50989`.

### Gemini CLI
- **Delegation:** Tasks are delegated to the engine using simple positional shell arguments.
- **Pattern:** `natvs-engine.exe "<objective>" "<context_path>"`
- **Protocol:** Gemini CLI monitors the engine's output for progress updates and uses the `ask_user` tool for manual approval during critical NATVS phase transitions (e.g., Synthesis).

## Adding New Agents

To register a new agent within the fleet so it is recognized by both platforms:

### 1. Register in Global AGENTS.md
Add a new `### Agent Name` section to `C:\aCogSpaceSeed\AGENTS.md`. Include its Role, Responsibility, and Interface. This is the primary source of truth for both Antigravity and Gemini CLI.

### 2. Configure Gemini CLI Subagent (Optional)
If the agent requires specialized tool handling beyond simple shell commands:
- Create a markdown file in `.gemini/agents/<agent-name>.md`.
- Define its tools (e.g., `run_shell_command`, `read_file`) and system prompt in the YAML frontmatter.

### 3. Verify Discovery
- **Antigravity:** Open the **Agent Manager** panel; the agent should appear automatically if documented in `AGENTS.md`.
- **Gemini CLI:** Run `/agents reload` followed by `/agents list` to confirm the agent is available for delegation.

---

## 🛠️ Verification & Conformance Testing

To maintain the architectural integrity of the NATVS Engine and SACP transport layers, the following testing suites must be executed and remain green:

### 1. Engine Main & Daemon Tests
Validates the internal phases of the NATVS engine (Negotiation, Assimilation, Transformation, Verification, Synthesis) and the background telemetry daemon:
```powershell
# From the workspace root (C:\aCogSpaceSeed):
& "C:\aCogSpaceSeed\00flow\s-forge\92000-external-toolchains\go\bin\go.exe" test -v sov.fleet/s-natives/...
```

### 2. SACP Broker Watchdog & Stream-Swap Conformance
Validates the pooled warm-worker stream-swap protocol and the watchdog reset-on-task idle shutdown invariants:
* **`TestIdleShutdownWatchdog`**: Verifies that the broker watchdog timer shuts down after inactivity but correctly delays and resets upon metabolic goal arrival.
* **`TestStreamSwapLifecycle`**: Verifies the dynamic allocation and cleanup of parallel invocation streams while keeping the control stream hot.
```powershell
# From the workspace root (C:\aCogSpaceSeed):
& "C:\aCogSpaceSeed\00flow\s-forge\92000-external-toolchains\go\bin\go.exe" test -v sov.fleet/s-hydration/100-synthesis-engine/...
```

### 3. SACP Go-Native Command Twins A/B Benchmarks
Measures the performance improvements of Go-native command polyfills (`pkg/bash` in `s-fab-aides`) compared to spawning external host shell processes (PowerShell):
```powershell
# From the workspace root (C:\aCogSpaceSeed):
& "C:\aCogSpaceSeed\00flow\s-forge\92000-external-toolchains\go\bin\go.exe" test -v -run "TestABPerformanceComparison" sov.fleet/s-fab-aides/81000-active-source/pkg/bash/...
```
* **Performance Baseline:** Bypasses process startup allocation delays entirely, demonstrating a **>3,400x speedup** (microsecond vs. millisecond scale) for file scans and pattern matching.


