# s-natives (NATVS Engine) — Future Work Review

> Filed: 2026-05-29
> Source: Antigravity workspace review (conversation 20688ce4)

## What It Is

The **NATVS Engine** (`sov.fleet/s-natives`) is the sovereign fleet-aware orchestrator at the heart of the 00flow platform. It manages the 5-phase NATVS lifecycle — **Negotiation → Assimilation → Transformation → Verification → Synthesis** — for delegating and executing tasks across the workspace fleet via the Jules agent, SACP broker, and QUIC control channels.

| Metric | Value |
|--------|-------|
| Module | `sov.fleet/s-natives` (Go 1.26.3) |
| Production code | ~2,600 lines across 16 `.go` files |
| Test code | ~1,600 lines across 8 test files |
| Binary output | `natvs-engine.exe` (13.2 MB) |
| Taxonomy directories | 141 (of which ~100+ are empty placeholders) |
| Cross-workspace deps | `s-fab-aides`, `s-logiclibrary`, `s-sacp`, `go-lib-quic-go` |

---

## Current Efficacy Summary

### Source Layout

All production code lives in two locations:

```
89000-internal-actors-source/          ← 6 files (main package)
  ├── natvs_engine.go                  ← CLI entry point, arg routing (348 lines)
  ├── natvs_engine_daemon.go           ← QUIC daemon, warm-VM tracking (392 lines)
  ├── daemon_runner.go                 ← Daemon/self-conformance bootstrap (62 lines)
  ├── execution_runner.go              ← Client-side task delegation (204 lines)
  ├── jules_commands.go                ← Jules auth/status/login (194 lines)
  └── queue_runner.go                  ← Per-task NATVS lifecycle runner (114 lines)
  └── engine/lifecycle/                ← 10 files (lifecycle package)
      ├── engine.go                    ← Core types (NATVSEngine, OrchestrationConfig) (87 lines)
      ├── negotiate.go                 ← Phase 1: Capability handshake (19 lines) ⚠️ STUB
      ├── assimilate.go                ← Phase 2: Taxonomy audit (50 lines)
      ├── transform.go                 ← Phase 3: Jules/compendium dispatch (97 lines)
      ├── verify.go                    ← Phase 4: Test discovery & execution (481 lines) ✅ STRONGEST
      ├── synthesis.go                 ← Phase 5: State promotion (19 lines) ⚠️ STUB
      ├── daemon.go                    ← SACP UDP telemetry (101 lines)
      ├── queue.go                     ← Task queue + weBNF serialization (494 lines) ✅ STRONGEST
      ├── cache.go                     ← Git-based workspace caching (172 lines)
      └── sandbox.go                   ← Hermetic Jules env builder (106 lines)
```

### Phase Maturity

| Phase | File | Lines | Status |
|-------|------|-------|--------|
| Negotiate | negotiate.go | 19 | ⚠️ **Stub** — simulated 10ms sleep |
| Assimilate | assimilate.go | 50 | ✅ Functional — taxonomy audit via bash polyfill |
| Transform | transform.go | 97 | ✅ Functional — routes jules-run, compendium, generic |
| Verify | verify.go | 481 | ✅ **Strongest** — workspace discovery, rehydration, conformance, parallel `go test` |
| Synthesis | synthesis.go | 19 | ⚠️ **Stub** — simulated 5ms sleep, logs only |

### Test Coverage

The test suite in `900-attestations/` is **well-structured** and covers:

- ✅ Full 5-phase lifecycle (end-to-end)
- ✅ SACP UDP bicodec round-trip
- ✅ Daemon watchdog timeout
- ✅ SACP broker multi-client coordination
- ✅ Stale socket cleanup
- ✅ Warm vs. cold start performance benchmarks
- ✅ Git cache operations + dirty state handling
- ✅ Queue serialization (weBNF read/write round-trip)
- ✅ Exclusive file locking
- ✅ Parallel atomic access (10 goroutines × 5 iterations)
- ✅ Persistent session (daemon survives task 1 → processes task 2)
- ✅ Cross-workspace compendium orchestration
- ✅ Direct execution (no daemon) path

---

## Key Novel Capabilities

### 1. weBNF Task Serialization
Tasks are serialized as grammar-conforming weBNF programs (`tasks.queue.webnf`, `tasks.completed.webnf`) rather than JSON/YAML. Queue files are **compiler-validated contracts** that any LLM can parse with zero-shot compliance.

### 2. Warm-VM Bypass
If a persistent QUIC connection exists and a workspace has already been processed, the engine **skips Negotiate + Assimilate entirely** and leaps directly to Transform → Verify. Eliminates cold boot latency for previously-warmed contexts.

### 3. QUIC Stream-per-Task Isolation
Each task gets its own QUIC stream via `OpenStreamSync` over a shared persistent connection. Failures on Task A **cannot destabilize** Task B. An initial control frame carries the task ID and workspace coordinates.

### 4. Git-Based Workspace Caching
A SHA-256 state key is computed from `git log` hash + `git status --porcelain` + file metadata (size/mtime). Unchanged workspaces are skipped entirely, making fleet-wide sweeps incremental.

### 5. Zero-Polling TCP Trigger Notifications
Instead of polling the queue file for task completion, the client connects to a TCP trigger server and blocks. The daemon sends a completion notification the instant a task finishes.

### 6. Sandboxed Credential Isolation
Jules credentials are hermetically isolated per `trackingScope` under `c1000-credentials/<scope>/`. USERPROFILE, APPDATA, and LOCALAPPDATA are dynamically remapped so multiple concurrent scopes cannot leak credentials.

### 7. Go-Native Bash Polyfills (>3,400x speedup)
Filesystem scans and pattern matching use in-process Go-native replacements (from `s-fab-aides/pkg/bash`) instead of spawning PowerShell subprocesses.

### 8. Transaction-Based Queue with Windows-Native Locking
Queue operations use `BeginQueueTransaction` → `Read` → `Write` → `Commit` with Windows `syscall.CreateFile` exclusive access locks. Prevents corruption under concurrent access.

### 9. SACP Bicodec Protocol
The Sovereign Agent Control Protocol uses `bicodec` binary encoding/decoding for compact UDP telemetry frames. Watchdog timer auto-shuts down idle daemons, broker coordinates multi-client connections.

### 10. Detached Server Spawning with Sanitized Environment
When no daemon is running, the engine spawns one as a fully detached process (`CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS`) with a **whitelisted** environment — only essential variables pass through.

---

## Future Work — Gaps to Fill

### 🔴 Priority 1 — Critical (Architectural Integrity)

- [ ] **Implement real Negotiate phase** — Replace the 10ms stub with actual capability/price/time handshake logic per the NATVS protocol definition in AGENTS.md. Agents must be able to declare capabilities and negotiate task boundaries.

- [ ] **Implement real Synthesis phase** — Replace the 5ms stub with fleet-aware state merging and metabolic pruning of ephemeral state. This is the final NATVS phase that ensures coordinated output across silos.

- [ ] **Reconcile dual `workspace.harness` files** — The root-level harness uses an older format (`go_build { type: NATIVE ... }`) while `71000-build-harness/workspace.harness` uses the canonical `hydrator.wag`-governed DSL syntax. Consolidate to the `71000` version and remove or deprecate the root version.

- [ ] **Update `s-natives.swdt.webnf`** — The workspace DNA declares `Domain "81000-active-source"` with entries `cmd` and `pkg`, but actual code lives in `89000-internal-actors-source/` with `engine/lifecycle/` structure. The DNA file must be updated to reflect reality.

---

### 🟠 Priority 2 — High (Functional Gaps)

- [ ] **Declare `go.mod` dependencies** — Add `require` directives for `s-fab-aides`, `s-logiclibrary`, `s-sacp`, and `go-lib-quic-go`. Currently works only under `go.work`; the module cannot be built standalone.

- [ ] **Add structured logging** — Replace all `fmt.Printf`/`fmt.Fprintf` with structured logging (e.g., `slog`). Add log levels, context fields, and machine-parseable output for a fleet orchestrator.

- [ ] **Add health check endpoint** — Expose an HTTP health endpoint for external monitoring, container orchestrators, or load balancers. The daemon currently only listens on UDP (SACP) and TCP (trigger server).

- [ ] **Add cross-platform file locking** — `queue.go` uses Windows-specific `syscall.CreateFile` with `FILE_SHARE_READ` for exclusive locking. Add `//go:build` tags and a Unix `flock` fallback to make the engine portable.

---

### 🟡 Priority 3 — Medium (Operational Gaps)

- [ ] **Write QUIC integration tests** — The QUIC control channel code in `natvs_engine_daemon.go` is untested. The test suite only exercises SACP (UDP bicodec) and TCP triggers. Stream-per-task isolation, warm-VM bypass, and TLS handshake paths need coverage.

- [ ] **Make constants configurable** — Extract hardcoded values into `OrchestrationConfig` or a config file:
  - UDP port `50989` (SACP daemon)
  - `15s` QUIC KeepAlivePeriod
  - `4` max concurrent tasks (semaphore size)
  - `3` max verify retry attempts

- [ ] **Regenerate compendium** — The current `c0411-compendium/s-natives_compendium.txt` (57KB) contains a pre-refactoring single-file version. Run compendium generation to capture the current multi-file architecture.

- [ ] **Populate governance directories** — `80000-system-governance/`, `80300-system-attestations/`, and `85000-standards/` are empty. Define formal governance rules, attestation policies, and standards for the sovereign engine.

- [ ] **Prune or document empty taxonomy** — ~100 of 141 directories are `.gitkeep` placeholders. Either document which are reserved for future use vs. which are vestigial, or prune the unused ones to reduce navigational noise.

---

### 🟢 Priority 4 — Low (Polish)

- [ ] **Expand test coverage data** — The existing `coverage.out` only covers `natvs_engine.go`. Generate coverage for the `engine/lifecycle/` package (especially `verify.go` at 481 lines and `queue.go` at 494 lines).

- [ ] **Add circuit breaker / backpressure** — The 3-retry fixer loop (Transform → Verify × 3) has no circuit breaker for cascading fleet-wide failures. Add backpressure signaling when multiple workspaces fail simultaneously.

- [ ] **Add graceful shutdown** — The daemon has a watchdog idle timer but no signal handler for SIGTERM/SIGINT. Add graceful draining of in-flight QUIC streams, flushing queue state, and closing the SACP listener cleanly.

---

## Summary Scorecard

| Dimension | Score | Notes |
|-----------|-------|-------|
| **Architecture** | ⭐⭐⭐⭐ | NATVS lifecycle, QUIC streams, task queue — well-designed |
| **Implementation** | ⭐⭐⭐ | Verify & Queue are strong; Negotiate & Synthesis are stubs |
| **Testing** | ⭐⭐⭐⭐ | Comprehensive suite covering concurrency, lifecycle, perf |
| **Novelty** | ⭐⭐⭐⭐⭐ | weBNF serialization, warm-VM bypass, >3400x polyfill — unique |
| **Operational Readiness** | ⭐⭐ | No observability, no health checks, Windows-only, hardcoded |
| **Taxonomy Completeness** | ⭐ | ~100 of 141 directories are empty placeholders |
| **Documentation** | ⭐⭐⭐⭐ | AGENTS.md, architecture/ docs are thorough |
