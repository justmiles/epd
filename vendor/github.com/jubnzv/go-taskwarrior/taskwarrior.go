// The MIT License (MIT)
// Copyright (C) 2018 Georgy Komarov <jubnzv@gmail.com>
//
// Most general definitions to manage list of tasks and taskwarrior instance configuration.
//
// To interact with taskwarrior I decided to use their command-line interface, instead manually parse .data files
// from `data.location` option. This solution looks better because there are few unique .data formats depending of
// taskwarrior version. For more detailed explanations see: https://taskwarrior.org/docs/3rd-party.html.

package taskwarrior

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Represents a single taskwarrior instance.
type TaskWarrior struct {
	Config *TaskRC // Configuration options
	Tasks  []Task  // Task JSON entries
}

// Create new empty TaskWarrior instance.
func NewTaskWarrior(configPath string) (*TaskWarrior, error) {
	// Read the configuration file.
	taskRC, err := ParseTaskRC(configPath)
	if err != nil {
		return nil, err
	}

	// Create new TaskWarrior instance.
	tw := &TaskWarrior{Config: taskRC}
	return tw, nil
}

// Fetch all tasks for given TaskWarrior with system `taskwarrior` command call.
func (tw *TaskWarrior) FetchAllTasks() error {
	if tw == nil {
		return fmt.Errorf("Uninitialized taskwarrior database!")
	}

	rcOpt := "rc:" + tw.Config.ConfigPath
	out, err := exec.Command("task", rcOpt, "export").Output()
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(out), &tw.Tasks)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// Pretty print for all tasks represented in given TaskWarrior.
func (tw *TaskWarrior) PrintTasks() {
	out, _ := json.MarshalIndent(tw.Tasks, "", "\t")
	os.Stdout.Write(out)
}

// Add new Task entry to given TaskWarrior.
func (tw *TaskWarrior) AddTask(task *Task) {
	tw.Tasks = append(tw.Tasks, *task)
}

// Save current changes of given TaskWarrior instance.
func (tw *TaskWarrior) Commit() error {
	tasks, err := json.Marshal(tw.Tasks)
	if err != nil {
		return err
	}

	cmd := exec.Command("task", "import", "-")
	cmd.Stdin = bytes.NewBuffer(tasks)
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
