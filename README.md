# bkmk

A terminal-based command bookmark manager with fuzzy search.

## Installation

```bash
go install github.com/sammcj/bkmk/cmd/bkmk@latest
```

Or build from source:

```bash
make build
./bin/bkmk
```

## Usage

```bash
# Launch interactive TUI
bkmk

# Add a group
bkmk add-group docker

# Add a command to a group
bkmk add docker ps "docker ps -a" "List all containers"

# List all bookmarks
bkmk list

# Remove a command
bkmk remove docker ps

# Remove a group
bkmk remove-group docker

# Browse shell history to add commands
bkmk history
```

## TUI Controls

| Key              | Action                          |
|------------------|---------------------------------|
| `↑/↓` or `j/k`   | Navigate                        |
| `Enter` or `Tab` | Select group / open action menu |
| `/`              | Search all commands (fuzzy)     |
| `h`              | Browse shell history to add     |
| `Ctrl+N/P`       | Navigate search results         |
| `a`              | Add group (in groups view) or command (in commands view) |
| `e`              | Edit selected command           |
| `d`              | Delete selected item (with confirmation) |
| `Esc`            | Go back / cancel                |
| `q` or `Ctrl+C`  | Quit                            |

### Action Menu

When selecting a command, an action menu appears:

| Key     | Action                          |
|---------|---------------------------------|
| `r`     | Run the command                 |
| `c`     | Copy to clipboard               |
| `Enter` | Execute selected action         |
| `Esc`   | Cancel                          |

## Config

Stored at `~/.config/bkmk/config.yaml`

```yaml
groups:
  - name: docker
    commands:
      - id: 1
        name: ps
        command: docker ps -a
        description: List all containers
        default_action: copy  # optional: copy, run, or none
```

Each command has a unique ID displayed in the TUI for reference. Commands can have a `default_action` that skips the action menu:
- `copy` - copies to clipboard immediately
- `run` - runs the command immediately
- `none` - shows the action menu (default)
