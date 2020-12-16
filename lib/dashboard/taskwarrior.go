package dashboard

import "fmt"

// TaskWarriorOptions defines options for TaskWarrior
type TaskWarriorOptions struct {
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

	for i, t := range d.taskWarriorService.Tasks {
		if t.Status == "pending" {
			ss = append(ss, fmt.Sprintf("%v) %s", i+1, t.Description))
		}
	}
	return ss
}
