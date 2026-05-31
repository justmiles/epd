package tududi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxTaskNameLen        = 50
	maxTaskNameGroupedLen = 38 // narrower when project name is a heading above
)

// priorityArrow converts a Tududi priority integer to a B&W-safe ASCII indicator.
// Priority: 1=low, 2=medium, 3=high, 4=urgent; nil means unset (treated as medium).
func priorityArrow(p *int) string {
	if p == nil {
		return " -"
	}
	switch *p {
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

// sortTasks sorts in-place: priority descending (urgent first), then due date
// ascending (nulls last).
func sortTasks(tasks []Task) {
	sort.Slice(tasks, func(i, j int) bool {
		ti, tj := tasks[i], tasks[j]
		pi, pj := 0, 0
		if ti.Priority != nil {
			pi = *ti.Priority
		}
		if tj.Priority != nil {
			pj = *tj.Priority
		}
		if pi != pj {
			return pi > pj
		}
		if ti.DueDate == nil && tj.DueDate == nil {
			return false
		}
		if ti.DueDate == nil {
			return false
		}
		if tj.DueDate == nil {
			return true
		}
		return *ti.DueDate < *tj.DueDate
	})
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
	fmt.Fprintf(&sb, "| %-58s | Pri |\n", "Task")
	fmt.Fprintf(&sb, "|%s|%s|\n", strings.Repeat("-", 60), strings.Repeat("-", 5))
	for _, t := range tasks {
		fmt.Fprintf(&sb, "| %-58s | %s |\n", truncate(t.Name, 58), priorityArrow(t.Priority))
	}
	return sb.String()
}

// FormatTasksByProject renders tasks grouped by project. Each project is shown
// as a bold heading followed by a mini-table of its top maxPerProject tasks
// (sorted by priority desc, then due date asc). Groups themselves are ordered
// by their highest-priority task. Tasks with no project are shown last under
// "Other".
func FormatTasksByProject(tasks []Task, projectNames map[int]string, maxPerProject int) string {
	if maxPerProject <= 0 {
		maxPerProject = 3
	}
	if len(tasks) == 0 {
		return "_No active tasks._"
	}

	// Group tasks by project key ("<id>" or "~none")
	type projectGroup struct {
		name        string
		tasks       []Task
		maxPriority int
	}

	groupMap := make(map[string]*projectGroup)
	var groupOrder []string

	for i := range tasks {
		t := tasks[i]
		key := "~none"
		name := "Other"
		if t.ProjectID != nil {
			key = strconv.Itoa(*t.ProjectID)
			if n, ok := projectNames[*t.ProjectID]; ok {
				name = n
			} else {
				name = fmt.Sprintf("Project %d", *t.ProjectID)
			}
		}
		if _, exists := groupMap[key]; !exists {
			groupMap[key] = &projectGroup{name: name}
			groupOrder = append(groupOrder, key)
		}
		g := groupMap[key]
		g.tasks = append(g.tasks, t)
		p := 0
		if t.Priority != nil {
			p = *t.Priority
		}
		if p > g.maxPriority {
			g.maxPriority = p
		}
	}

	// Sort groups: highest priority task first; "Other" always last
	sort.SliceStable(groupOrder, func(i, j int) bool {
		ki, kj := groupOrder[i], groupOrder[j]
		if ki == "~none" {
			return false
		}
		if kj == "~none" {
			return true
		}
		return groupMap[ki].maxPriority > groupMap[kj].maxPriority
	})

	var sb strings.Builder

	for idx, key := range groupOrder {
		g := groupMap[key]
		sortTasks(g.tasks)

		shown := g.tasks
		if len(shown) > maxPerProject {
			shown = shown[:maxPerProject]
		}

		if idx > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "**%s**\n\n", g.name)
		fmt.Fprintf(&sb, "| %-46s | Pri |\n", "Task")
		fmt.Fprintf(&sb, "|%s|%s|\n", strings.Repeat("-", 48), strings.Repeat("-", 5))
		for _, t := range shown {
			fmt.Fprintf(&sb, "| %-46s | %s |\n",
				truncate(t.Name, 46),
				priorityArrow(t.Priority),
			)
		}
	}

	return sb.String()
}
