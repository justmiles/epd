package dashboard

import (
	"fmt"
	"sort"

	"github.com/jubnzv/go-taskwarrior"
)

// TaskWarriorOptions defines options for TaskWarrior
type TaskWarriorOptions struct {
	Enable     bool
	ConfigPath string
}

// WithTaskWarrior creates a custom dashboard with Task Warrior configured
func WithTaskWarrior(taskWarriorOptions *TaskWarriorOptions) Options {
	return func(d *Dashboard) {
		d.taskWarriorOptions = taskWarriorOptions
	}
}

func (d *Dashboard) getTaskWarriorTasks() (ss []string) {
	d.taskWarriorService.FetchAllTasks()

	// Sort Tasks
	sort.Sort(byUrgency(d.taskWarriorService.Tasks))

	for _, t := range d.taskWarriorService.Tasks {
		if t.Status == "pending" && t.Priority != "L" {
			var project string
			if t.Project != "" {
				project = fmt.Sprintf("[%s] ", t.Project)
			}

			ss = append(ss, fmt.Sprintf("%v) %s%s", t.Id, project, t.Description))
		}
	}
	return ss
}

type byUrgency []taskwarrior.Task

func (s byUrgency) Len() int {
	return len(s)
}
func (s byUrgency) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byUrgency) Less(i, j int) bool {
	return s[i].Urgency > s[j].Urgency
}
