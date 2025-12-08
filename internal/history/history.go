package history

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Entry struct {
	Command   string
	Index     int
	Timestamp time.Time
}

func GetHistoryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return GetHistoryPathWithHome(home)
}

func GetHistoryPathWithHome(home string) (string, error) {
	// Check HISTFILE environment variable first (user's explicit preference)
	if histFile := os.Getenv("HISTFILE"); histFile != "" {
		if _, err := os.Stat(histFile); err == nil {
			return histFile, nil
		}
	}

	// Check for zsh history
	zshHistory := filepath.Join(home, ".zsh_history")
	if _, err := os.Stat(zshHistory); err == nil {
		return zshHistory, nil
	}

	// Check for bash history
	bashHistory := filepath.Join(home, ".bash_history")
	if _, err := os.Stat(bashHistory); err == nil {
		return bashHistory, nil
	}

	// Check for generic .history (common with custom HISTFILE configurations)
	dotHistory := filepath.Join(home, ".history")
	if _, err := os.Stat(dotHistory); err == nil {
		return dotHistory, nil
	}

	return "", os.ErrNotExist
}

func ReadHistory(limit int) ([]Entry, error) {
	path, err := GetHistoryPath()
	if err != nil {
		return nil, err
	}

	return ReadHistoryFrom(path, limit)
}

func ReadHistoryFrom(path string, limit int) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var allLines []string
	scanner := bufio.NewScanner(file)

	// Increase buffer size for long commands
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		allLines = append(allLines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Process lines (handle zsh extended history format)
	var entries []Entry
	seen := make(map[string]bool)

	// Read from end (most recent first)
	for i := len(allLines) - 1; i >= 0; i-- {
		cmd, ts := parseHistoryLine(allLines[i])
		if cmd == "" {
			continue
		}

		// Skip duplicates
		if seen[cmd] {
			continue
		}
		seen[cmd] = true

		entries = append(entries, Entry{
			Command:   cmd,
			Index:     len(entries),
			Timestamp: ts,
		})

		if limit > 0 && len(entries) >= limit {
			break
		}
	}

	return entries, nil
}

func parseHistoryLine(line string) (string, time.Time) {
	var ts time.Time

	// Handle zsh extended history format: : timestamp:0;command
	if strings.HasPrefix(line, ": ") {
		if idx := strings.Index(line, ";"); idx != -1 {
			// Extract timestamp between ": " and ":"
			tsPart := line[2:idx]
			if colonIdx := strings.Index(tsPart, ":"); colonIdx != -1 {
				tsPart = tsPart[:colonIdx]
			}
			if epoch, err := strconv.ParseInt(tsPart, 10, 64); err == nil {
				ts = time.Unix(epoch, 0)
			}
			line = line[idx+1:]
		}
	}

	// Trim whitespace
	line = strings.TrimSpace(line)

	// Skip empty lines and very short commands
	if len(line) < 2 {
		return "", ts
	}

	// Skip common non-useful commands
	skip := []string{"ls", "cd", "pwd", "clear", "exit", "history"}
	if slices.Contains(skip, line) {
		return "", ts
	}

	return line, ts
}
