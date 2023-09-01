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
	"github.com/aws/amazon-ecs-agent/agent/statechange"
)

var _ TaskEngine = &taskEngineWrapper{}

type taskEngineWrapper struct {
	dockerTaskEngine  TaskEngine
	processTaskEngine TaskEngine

	done              chan interface{}
	stateChangeEvents chan statechange.Event
}

func newTaskEngineWrapper(dte TaskEngine, pte TaskEngine) TaskEngine {
	te := &taskEngineWrapper{
		dockerTaskEngine:  dte,
		processTaskEngine: pte,
		done:              make(chan interface{}),
		stateChangeEvents: make(chan statechange.Event),
	}
	go te.processEvents()
	return te
}

func (tew *taskEngineWrapper) processEvents() {
	// Loop and forward events from managers forever until disable.
	for {
		select {
		case e := <-tew.dockerTaskEngine.StateChangeEvents():
			tew.stateChangeEvents <- e
		case e := <-tew.processTaskEngine.StateChangeEvents():
			tew.stateChangeEvents <- e
		case <-tew.done:
			return // exit routine
		}
	}
}

func (tew *taskEngineWrapper) Init(ctx context.Context) error {
	err := tew.dockerTaskEngine.Init(ctx)
	if err != nil {
		return err
	}
	return tew.processTaskEngine.Init(ctx)
}
func (tew *taskEngineWrapper) MustInit(ctx context.Context) {
	tew.dockerTaskEngine.MustInit(ctx)
	tew.processTaskEngine.MustInit(ctx)
}

func (tew *taskEngineWrapper) Disable() {
	tew.dockerTaskEngine.Disable()
	tew.processTaskEngine.Disable()

	// release all event listeners
	close(tew.done)
}

func (tew *taskEngineWrapper) StateChangeEvents() chan statechange.Event {
	return tew.stateChangeEvents
}

func (tew *taskEngineWrapper) SetDataClient(client data.Client) {
	// TODO: Is there any managed daemon task state to do here?
	tew.dockerTaskEngine.SetDataClient(client)
}

func (tew *taskEngineWrapper) AddTask(task *apitask.Task) {
	if len(task.Containers) == 1 && task.Containers[0].Type == container.ContainerManagedDaemon {
		// TODO: Should probably set some sort of func for "IsManagedDaemon"
		tew.processTaskEngine.AddTask(task)
	} else {
		tew.dockerTaskEngine.AddTask(task)
	}
}

func (tew *taskEngineWrapper) ListTasks() ([]*apitask.Task, error) {
	tasks, err := tew.dockerTaskEngine.ListTasks()
	if err != nil {
		return nil, err
	}
	ptasks, err := tew.processTaskEngine.ListTasks()
	if err != nil {
		return nil, err
	}
	allTasks := append(tasks, ptasks...)
	return allTasks, nil
}

func (tew *taskEngineWrapper) GetTaskByArn(arn string) (*apitask.Task, bool) {
	t, found := tew.dockerTaskEngine.GetTaskByArn(arn)
	if !found {
		t, found = tew.processTaskEngine.GetTaskByArn(arn)
	}
	return t, found
}

func (tew *taskEngineWrapper) Version() (string, error) {
	// Sorta a hack but prefer version of dte since pte is hidden concept
	return tew.dockerTaskEngine.Version()
}

func (tew *taskEngineWrapper) LoadState() error {
	// TODO: Is there any managed daemon task state to do here?
	return tew.dockerTaskEngine.LoadState()
}

func (tew *taskEngineWrapper) SaveState() error {
	// TODO: is there any managed daemon task state to do here?
	return tew.dockerTaskEngine.SaveState()
}

func (tew *taskEngineWrapper) MarshalJSON() ([]byte, error) {
	// TODO: is there any managed daemon task state to do here?
	return tew.dockerTaskEngine.MarshalJSON()
}

func (tew *taskEngineWrapper) UnmarshalJSON(data []byte) error {
	// TODO: is there any managed daemon task state to do here?
	return tew.dockerTaskEngine.UnmarshalJSON(data)
}
