package tududi

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxTaskNameLen = 50
)

// priorityArrow converts a Tududi priority integer to a B&W-safe ASCII indicator.
// Priority: 1=low, 2=medium, 3=high, 4=urgent
func priorityArrow(p int) string {
	switch p {
	case PriorityUrgent:
		return "^^"
	case PriorityHigh:
		return " ^"
	case PriorityMedium:
		return " -"
	case PriorityLow:
		return " v"
	default:
		return " -"
	}
}

// formatDueDate formats a Tududi due-date string ("2006-01-02") to a short
// human-readable form ("Jan 02"). Returns "—" if the date is nil or unparseable.
func formatDueDate(d *string) string {
	if d == nil || *d == "" {
		return "  —  "
	}
	t, err := time.Parse("2006-01-02", *d)
	if err != nil {
		return "  —  "
	}
	return t.Format("Jan 02")
}

// truncate shortens s to at most maxLen runes, appending "…" if truncated.
func truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen-1]) + "…"
}

// FormatTasksAsMarkdown converts a slice of Tasks into a compact GFM table
// suitable for the 800×480 e-paper display. At most maxTasks rows are rendered.
// If maxTasks <= 0, it defaults to 15.
func FormatTasksAsMarkdown(tasks []Task, maxTasks int) string {
	if maxTasks <= 0 {
		maxTasks = 15
	}

	if len(tasks) == 0 {
		return "_No active tasks._"
	}

	if len(tasks) > maxTasks {
		tasks = tasks[:maxTasks]
	}

	var sb strings.Builder

	// Table header
	fmt.Fprintf(&sb, "| %-50s | %-6s | Pri |\n", "Task", "Due")
	fmt.Fprintf(&sb, "|%s|%s|%s|\n",
		strings.Repeat("-", 52),
		strings.Repeat("-", 8),
		strings.Repeat("-", 5),
	)

	for _, t := range tasks {
		name := truncate(t.Name, maxTaskNameLen)
		due := formatDueDate(t.DueDate)
		pri := priorityArrow(t.Priority)
		fmt.Fprintf(&sb, "| %-50s | %-6s | %s |\n", name, due, pri)
	}

	remaining := len(tasks)
	if remaining == maxTasks && remaining > 0 {
		// We might have cut off tasks — add a note
		fmt.Fprintf(&sb, "\n_Showing %d tasks._", maxTasks)
	}

	return sb.String()
}
