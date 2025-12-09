# CLAUDE.md

<BUILD>
```bash
make lint && make test && make build   # Always run all three before completing work
go test -v -run TestName ./internal/config/   # Single test
```

Version info injected via ldflags: `-X main.Version=... -X main.Commit=... -X main.BuildDate=...`
</BUILD>

<ARCHITECTURE>
Terminal bookmark manager using Bubble Tea TUI framework. Config stored at `~/.config/bkmk/config.yaml`.

**Packages:**
- `cmd/bkmk/` - CLI entrypoint, handles subcommands and TUI launch
- `internal/config/` - YAML config with groups/commands CRUD, auto-backup on save
- `internal/tui/` - Bubble Tea model with `viewMode` state machine
- `internal/runner/` - Command execution, clipboard (pbcopy/xclip/xsel)
- `internal/history/` - Shell history parsing (zsh extended + bash formats)

**Data flow:** TUI always works with `FlatCommand` (denormalised). Nested `Command`/`Group` structure only used for YAML serialisation.

**State machine flows:**
- Groups → Commands (enter/tab)
- Any view → Search (/) or History (h)
- Commands/Search → Action selection → Run or Copy
</ARCHITECTURE>

<CONVENTIONS>
- Config saves immediately on every CRUD operation - no batching
- Commands have auto-incrementing IDs via `NextID` in config; ID migration runs on load for legacy configs
- `ActionType` enum: `ActionNone`, `ActionCopy`, `ActionRun` - commands with default_action skip the action menu
- Backups use microsecond timestamps (`20060102-150405.000000`) for uniqueness and are rotated when a maximum number is reached
- History skips noise commands: ls, cd, pwd, clear, exit, history
- Tests use `t.TempDir()` for file fixtures; preserve/restore env vars when testing
</CONVENTIONS>

<GOTCHAS>
**Bubble Tea model type:** `main.go` explicitly handles both `Model` and `*Model` return types from Bubble Tea's `Run()`. Don't assume pointer behaviour.

**Cursor bounds:** Always clamp cursor after filtering or state changes. Use `maxCursor()` helper and check bounds in filter updates to prevent panics.

**Form inputs are destructive:** Creating new form inputs via `createFormInputs()` clears `formError` and resets focus. No state preserved between form invocations.

**No command timeout:** `RunCommand()` passes through stdio directly with no timeout. Long-running commands block the TUI.

**View height:** History page size reserves 8 lines for UI chrome. Terminals under ~13 lines cause pagination issues.

**Platform clipboard:** macOS uses `pbcopy`; Linux falls back to `xclip` then `xsel`. Shell detected from `$SHELL` or defaults to `/bin/sh`.
</GOTCHAS>

<TESTING>
Tests alongside implementation files. Table-driven for parsing, temp directories for file I/O.

When modifying TUI handlers, verify cursor boundary handling. When changing config operations, ensure immediate save behaviour preserved.
</TESTING>

<INSTRUCTIONS>
- Always use Australian English spelling in code, comments and documentation
- Run `make lint && make test && make build` after any changes - all must pass with no warnings
- Add lightweight unit tests for new functionality
- Follow existing code and interface styles
- Keep the readme clean, consise and free of fluff
</INSTRUCTIONS>
