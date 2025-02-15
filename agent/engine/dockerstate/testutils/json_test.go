//go:build unit

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

package testutils

import (
	"encoding/json"
	"runtime/debug"
	"strconv"
	"testing"

	apicontainer "github.com/aws/amazon-ecs-agent/agent/api/container"
	apicontainerstatus "github.com/aws/amazon-ecs-agent/agent/api/container/status"
	apitask "github.com/aws/amazon-ecs-agent/agent/api/task"
	apitaskstatus "github.com/aws/amazon-ecs-agent/agent/api/task/status"
	"github.com/aws/amazon-ecs-agent/agent/engine/dockerstate"
	"github.com/stretchr/testify/assert"
)

func createTestContainer(num int) *apicontainer.Container {
	return &apicontainer.Container{
		Name:                "busybox-" + strconv.Itoa(num),
		Image:               "public.ecr.aws/docker/library/busybox:1.34.1",
		Essential:           true,
		DesiredStatusUnsafe: apicontainerstatus.ContainerRunning,
	}
}

func createTestTask(arn string, numContainers int) *apitask.Task {
	task := &apitask.Task{
		Arn:                 arn,
		Family:              arn,
		Version:             "1",
		DesiredStatusUnsafe: apitaskstatus.TaskRunning,
		Containers:          []*apicontainer.Container{},
	}

	for i := 0; i < numContainers; i++ {
		task.Containers = append(task.Containers, createTestContainer(i+1))
	}
	return task
}

func decodeEqual(t *testing.T, state dockerstate.TaskEngineState) dockerstate.TaskEngineState {
	data, err := json.Marshal(&state)
	assert.NoError(t, err, "marshal state")

	otherState := dockerstate.NewTaskEngineState()
	err = json.Unmarshal(data, &otherState)
	assert.NoError(t, err, "unmarshal state")

	if !DockerStatesEqual(state, otherState) {
		debug.PrintStack()
		t.Error("States were not equal")
	}
	return otherState
}

func TestJsonEncoding(t *testing.T) {
	state := dockerstate.NewTaskEngineState()
	decodeEqual(t, state)

	testState := dockerstate.NewTaskEngineState()
	testTask := createTestTask("test1", 1)
	testState.AddTask(testTask)
	for i, cont := range testTask.Containers {
		testState.AddContainer(&apicontainer.DockerContainer{DockerID: "docker" + strconv.Itoa(i), DockerName: "someName", Container: cont}, testTask)
	}
	other := decodeEqual(t, testState)
	_, ok := other.ContainerMapByArn("test1")
	assert.True(t, ok, "could not retrieve expected task")
}
