# bkmk

A terminal-based command bookmark manager.

Organise, search and run your bookmarked shell commands from a simple TUI.

## Installation

```bash
go install github.com/sammcj/bkmk/cmd/bkmk@HEAD
```

Or build from source:

```bash
make build
./bin/bkmk
```

## Usage

```bash
bkmk              # Launch interactive TUI
bkmk last         # Bookmark the last command you ran
bkmk history      # Browse shell history to add commands
bkmk list         # List all bookmarks
bkmk suggest      # Show frequently used commands worth bookmarking

bkmk add-group docker
bkmk remove-group docker
bkmk add docker ps "docker ps -a" "List all containers"
bkmk remove docker ps
```

Optional shell aliases:

```bash
alias b='bkmk'
alias bl='bkmk last'
```

## TUI Controls

| Key                | Action                              |
|--------------------|-------------------------------------|
| `↑/↓` or `j/k`     | Navigate                            |
| `→` or `Enter/Tab` | Enter group                         |
| `←` or `Esc`       | Go back                             |
| `/`                | Search all commands (fuzzy)         |
| `s`                | Show all bookmarks across groups    |
| `h`                | Browse shell history                |
| `a`                | Add group or command                |
| `e`                | Edit selected item                  |
| `d`                | Delete selected item                |
| `o`                | Open config in editor               |
| `q` or `Ctrl+C`    | Quit                                |

### Action Menu

When selecting a command:

| Key     | Action              |
|---------|---------------------|
| `r`     | Run command         |
| `c`     | Copy to clipboard   |
| `Esc`   | Cancel              |

## Config

Stored at `~/.config/bkmk/config.yaml`. Backups saved to `~/.config/bkmk/backup/`.

```yaml
editor: code  # Optional: editor for 'o' key (falls back to $EDITOR, then vi)

groups:
  - name: docker
    commands:
      - id: 1
        name: ps
        command: docker ps -a
        description: List all containers
        default_action: copy  # Optional: copy, run, or none (default)
```

### Default Actions

Set `default_action` on a command to skip the action menu:
- `copy` - copy to clipboard immediately
- `run` - run immediately
- `none` - show action menu (default)
