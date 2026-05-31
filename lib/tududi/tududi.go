package tududi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// Task statuses
	StatusNotStarted = 0
	StatusInProgress = 1
	StatusCompleted  = 2
	StatusArchived   = 6

	// Task priorities (low → urgent)
	PriorityLow    = 1
	PriorityMedium = 2
	PriorityHigh   = 3
	PriorityUrgent = 4
)

// Options holds configuration for the Tududi body-text provider.
type Options struct {
	URL      string // base URL, e.g. "http://localhost:3004"
	APIToken string // Bearer token from Tududi Settings → API Tokens
	Project  string // optional: project ID (integer) or project name
	MaxTasks int    // max tasks to render; defaults to 15 when ≤ 0
}

// Task represents a Tududi task as returned by the API.
type Task struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Status    int     `json:"status"`    // 0=not_started, 1=in_progress, 2=completed, 6=archived
	Priority  int     `json:"priority"`  // 1=low, 2=medium, 3=high, 4=urgent
	DueDate   *string `json:"due_date"`  // "2026-02-10" or null
	ProjectID *int    `json:"project_id"`
}

// Project represents a Tududi project.
type Project struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Client is an HTTP client for the Tududi REST API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Tududi API client.
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ListTasks returns all active (non-completed, non-archived) tasks.
// When projectID is non-nil, only tasks belonging to that project are returned.
// Tasks are sorted by priority descending (urgent first), then by due date ascending (nulls last).
func (c *Client) ListTasks(projectID *int) ([]Task, error) {
	url := c.baseURL + "/api/v1/tasks"
	if projectID != nil {
		url += "?project_id=" + strconv.Itoa(*projectID)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}

	var tasks []Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("decoding tasks response: %w", err)
	}

	// Filter out completed and archived tasks
	active := tasks[:0]
	for _, t := range tasks {
		if t.Status != StatusCompleted && t.Status != StatusArchived {
			active = append(active, t)
		}
	}

	// Sort: priority desc (4→1), then due date asc (nulls last)
	sort.Slice(active, func(i, j int) bool {
		ti, tj := active[i], active[j]
		if ti.Priority != tj.Priority {
			return ti.Priority > tj.Priority
		}
		// Both have no due date — equal
		if ti.DueDate == nil && tj.DueDate == nil {
			return false
		}
		// Nulls sort last
		if ti.DueDate == nil {
			return false
		}
		if tj.DueDate == nil {
			return true
		}
		return *ti.DueDate < *tj.DueDate
	})

	return active, nil
}

// ListProjects returns all projects from the Tududi API.
func (c *Client) ListProjects() ([]Project, error) {
	url := c.baseURL + "/api/v1/projects"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}

	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("decoding projects response: %w", err)
	}

	return projects, nil
}

// FindProjectByName finds a project using the given query.
// It first tries to parse query as an integer project ID; if that succeeds it
// returns the matching project directly (fetching project list to confirm).
// Otherwise it falls back to a case-insensitive substring name match.
func (c *Client) FindProjectByName(query string) (*Project, error) {
	projects, err := c.ListProjects()
	if err != nil {
		return nil, err
	}

	// Try ID match first
	if id, err := strconv.Atoi(query); err == nil {
		for i := range projects {
			if projects[i].ID == id {
				return &projects[i], nil
			}
		}
		return nil, fmt.Errorf("no project found with ID %d", id)
	}

	// Fall back to case-insensitive name match
	queryLower := strings.ToLower(query)
	for i := range projects {
		if strings.Contains(strings.ToLower(projects[i].Name), queryLower) {
			return &projects[i], nil
		}
	}

	return nil, fmt.Errorf("no project found matching %q", query)
}
