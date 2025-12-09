package history

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// FrequentCommand represents a command with its frequency count.
type FrequentCommand struct {
	Command string
	Count   int
}

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

// GetFrequentCommands analyses shell history and returns the most frequently
// used commands that have at least minArgs arguments, from the last daysBack days.
// Returns up to limit results sorted by frequency (descending).
func GetFrequentCommands(daysBack, minArgs, limit int) ([]FrequentCommand, error) {
	path, err := GetHistoryPath()
	if err != nil {
		return nil, err
	}
	return GetFrequentCommandsFrom(path, daysBack, minArgs, limit)
}

// GetFrequentCommandsFrom analyses shell history from a specific file.
func GetFrequentCommandsFrom(path string, daysBack, minArgs, limit int) ([]FrequentCommand, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cutoff := time.Now().AddDate(0, 0, -daysBack)
	counts := make(map[string]int)
	hasTimestamps := false

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		cmd, ts := parseHistoryLine(line)
		if cmd == "" {
			continue
		}

		// Track if we have timestamp support
		if !ts.IsZero() {
			hasTimestamps = true
		}

		// Filter by date if timestamps available
		if hasTimestamps {
			// Skip entries without timestamps (likely multiline fragments)
			if ts.IsZero() {
				continue
			}
			// Skip entries older than cutoff
			if ts.Before(cutoff) {
				continue
			}
		}

		// Filter by minimum argument count and length
		if countArgs(cmd) < minArgs || len(cmd) < 13 {
			continue
		}

		// Skip bkmk commands
		if strings.HasPrefix(cmd, "bkmk") {
			continue
		}

		// Skip line continuations and fragments from multiline commands
		if isMultilineFragment(cmd) {
			continue
		}

		counts[cmd]++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Convert to slice and sort by frequency
	result := make([]FrequentCommand, 0, len(counts))
	for cmd, count := range counts {
		result = append(result, FrequentCommand{Command: cmd, Count: count})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Command < result[j].Command // stable sort by command name
	})

	// Limit results
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// countArgs returns the number of space-separated arguments in a command.
// Handles quoted strings as single arguments.
func countArgs(cmd string) int {
	var count int
	var inQuote rune
	var hasContent bool

	for _, r := range cmd {
		switch {
		case inQuote != 0:
			if r == inQuote {
				inQuote = 0
			}
		case r == '"' || r == '\'':
			inQuote = r
			hasContent = true
		case r == ' ' || r == '\t':
			if hasContent {
				count++
				hasContent = false
			}
		default:
			hasContent = true
		}
	}

	if hasContent {
		count++
	}

	return count
}

// isMultilineFragment returns true if cmd looks like a fragment from a
// multiline command or heredoc rather than a standalone command.
func isMultilineFragment(cmd string) bool {
	// Too long - likely a heredoc or embedded content
	if len(cmd) > 300 {
		return true
	}

	// Line continuations (ends with backslash)
	if strings.HasSuffix(cmd, "\\") {
		return true
	}

	// Must start with something that looks like a command
	// (letter, path, variable, or sudo/env prefix)
	if len(cmd) == 0 {
		return true
	}
	first := cmd[0]
	isValidStart := (first >= 'a' && first <= 'z') ||
		(first >= 'A' && first <= 'Z') ||
		first == '.' || first == '/' || first == '~' || first == '$'

	return !isValidStart
}
