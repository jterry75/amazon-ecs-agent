// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Package engine contains the core logic for managing tasks
package engine

import (
	"context"

	"github.com/aws/amazon-ecs-agent/agent/api/container"
	apitask "github.com/aws/amazon-ecs-agent/agent/api/task"
	"github.com/aws/amazon-ecs-agent/agent/data"
	dm "github.com/aws/amazon-ecs-agent/agent/engine/daemonmanager"
	"github.com/aws/amazon-ecs-agent/agent/statechange"
)

var _ TaskEngine = &processTaskEngine{}

type processTaskEngine struct {
	daemonManagers map[string]dm.DaemonManager

	stateChanges chan statechange.Event
	runningTasks map[string]*apitask.Task
}

func newProcessTaskEngine(daemonManagers map[string]dm.DaemonManager) TaskEngine {
	return &processTaskEngine{
		daemonManagers: daemonManagers,
		stateChanges:   make(chan statechange.Event),
		runningTasks:   make(map[string]*apitask.Task),
	}
}

func (pte *processTaskEngine) Init(ctx context.Context) error {
	return nil
}
func (pte *processTaskEngine) MustInit(ctx context.Context) {
	return
}

func (pte *processTaskEngine) Disable() {
	// TODO: Is there any managed daemon task state to do here?
}

func (pte *processTaskEngine) StateChangeEvents() chan statechange.Event {
	return pte.stateChanges
}

func (pte *processTaskEngine) SetDataClient(_ data.Client) {
	// TODO: Is there any managed daemon task state to do here?
}

func (pte *processTaskEngine) AddTask(task *apitask.Task) {
	isManagedDaemon := len(task.Containers) == 1 && task.Containers[0].Type == container.ContainerManagedDaemon
	if !isManagedDaemon {
		panic("unsupported process engine type")
	}

	if task.IsEBSTaskAttachEnabled() {
		// TODO: Load the managed daemon to this scope if not loaded
		// TODO: Start the background task command from the install dir
		// TODO: Monitor process
		// TODO: Event on process changes
	}
}

func (pte *processTaskEngine) ListTasks() ([]*apitask.Task, error) {
	var allTasks []*apitask.Task
	for _, v := range pte.runningTasks {
		// TODO: This is a ptr, probably not safe to return by ref
		allTasks = append(allTasks, v)
	}
	return allTasks, nil
}

func (pte *processTaskEngine) GetTaskByArn(arn string) (*apitask.Task, bool) {
	task, exists := pte.runningTasks[arn]
	if !exists {
		return nil, false
	}
	// TODO: This is a ptr, probably not safe to return by ref
	return task, true
}

func (pte *processTaskEngine) Version() (string, error) {
	return "1.0.0", nil
}

func (pte *processTaskEngine) LoadState() error {
	// TODO: Is there any managed daemon task state to do here?
	return nil
}

func (pte *processTaskEngine) SaveState() error {
	// TODO: is there any managed daemon task state to do here?
	return nil
}

func (pte *processTaskEngine) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (pte *processTaskEngine) UnmarshalJSON(data []byte) error {
	return nil
}
