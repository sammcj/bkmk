# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
make build      # Build binary to bin/bkmk
make test       # Run all tests
make lint       # Run golangci-lint
make run        # Build and run
```

Run a single test:
```bash
go test -v -run TestName ./internal/config/
```

## Architecture

bkmk is a lightweight, clean terminal-based command bookmark manager built with the Bubble Tea TUI framework.

### Package Structure

- `cmd/bkmk/main.go` - CLI entrypoint, handles both CLI subcommands and TUI launch
- `internal/config/` - YAML config management at `~/.config/bkmk/config.yaml`, handles groups/commands CRUD
- `internal/tui/` - Bubble Tea model with multiple view modes (groups, commands, search, history, action selection)
- `internal/runner/` - Command execution and clipboard operations (pbcopy on macOS, xclip/xsel on Linux)
- `internal/history/` - Shell history parsing (zsh extended format and bash)

### TUI State Machine

The TUI uses a `viewMode` enum to manage UI states. Key flows:
- Groups view → Commands view (enter/tab)
- Any view → Search view (/)
- Any view → History view (h) → Group selection → Add details
- Commands/Search → Action selection → Run or Copy

Config changes are saved immediately after each CRUD operation.

### Data Model

Commands have unique auto-incrementing IDs (tracked via `NextID` in config). The `FlatCommand` type denormalises group membership for search results.

## General Information

- Always build, lint and test from the `make` command entrypoint
- Always follow the latest golang best practices for 2025
- When changing or adding new functionality:
  - Follow the existing code and interface styles
  - Add a simple, lightweight unit test created
  - Ensure the README is kept up to date and free off fluff
  - Always run `make lint && make test && make build` after adding, changing or removing features or functionality and ensure there are no warnings or errors
- Always use Australian English spelling in all code, comments and documentation
