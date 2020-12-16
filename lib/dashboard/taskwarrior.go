package dashboard

import "fmt"

// TaskWarriorOptions defines options for TaskWarrior
type TaskWarriorOptions struct {
	ConfigPath string
}

// WithTaskWarrior creates a custom dashboard with Task Warrior configured
func WithTaskWarrior(taskWarriorOptions *TaskWarriorOptions) Options {
	return func(cd *Dashboard) {
		cd.taskWarriorOptions = taskWarriorOptions
	}
}

func (cd *Dashboard) getTaskWarriorTasks() (ss []string) {
	cd.taskWarriorService.FetchAllTasks()

	for i, t := range cd.taskWarriorService.Tasks {
		if t.Status == "pending" {
			ss = append(ss, fmt.Sprintf("%v) %s", i+1, t.Description))
		}
	}
	return ss
}
