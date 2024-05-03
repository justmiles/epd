package dashboard

import (
	"github.com/naggie/dstask"
)

// DstaskOptions defines options for Dstask
type DstaskOptions struct {
	Enable bool
	Config dstask.Config
}

// WithDstask creates a custom dashboard with Task Warrior configured
func WithDstask(DstaskOptions *DstaskOptions) Options {
	return func(d *Dashboard) {
		d.dstaskOptions = DstaskOptions
		if d.dstaskOptions == nil {
			d.dstaskOptions.Config = dstask.NewConfig()
		}
	}
}

func (d *Dashboard) getDstaskTasks() (ss []string) {
	cfg := dstask.NewConfig()

	ts, err := dstask.LoadTaskSet(cfg.Repo, cfg.IDsFile, false)
	if err != nil {
		panic(err) // TODO: don't panic
	}

	tasks := ts.AllTasks()
	for _, task := range tasks {
		ss = append(ss, task.String())
	}

	return ss
}
