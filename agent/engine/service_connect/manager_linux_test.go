//go:build linux && unit
// +build linux,unit

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

package serviceconnect

import (
	"testing"

	"github.com/aws/amazon-ecs-agent/agent/api/serviceconnect"

	apicontainer "github.com/aws/amazon-ecs-agent/agent/api/container"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
)

func TestDNSConfigToDockerExtraHostsFormat(t *testing.T) {
	tt := []struct {
		dnsConfigs      []serviceconnect.DNSConfigEntry
		expectedRestult []string
	}{
		{
			dnsConfigs: []serviceconnect.DNSConfigEntry{
				{
					HostName: "my.test.host",
					Address:  "169.254.1.1",
				},
				{
					HostName: "my.test.host2",
					Address:  "ff06::c3",
				},
			},
			expectedRestult: []string{
				"my.test.host:169.254.1.1",
				"my.test.host2:ff06::c3",
			},
		},
		{
			dnsConfigs:      nil,
			expectedRestult: nil,
		},
	}

	for _, tc := range tt {
		res := DNSConfigToDockerExtraHostsFormat(tc.dnsConfigs)
		assert.Equal(t, tc.expectedRestult, res, "Wrong docker host config ")
	}
}

func TestPauseContainerModificationsForServiceConnect(t *testing.T) {
	scTask, pauseContainer, _ := getAWSVPCTask(t)

	expectedPauseExtraHosts := []string{
		"host1.my.corp:169.254.1.1",
		"host1.my.corp:ff06::c4",
	}

	type testCase struct {
		name               string
		container          *apicontainer.Container
		expectedExtraHosts []string
	}
	testcases := []testCase{
		{
			name:               "Pause container has extra hosts",
			container:          pauseContainer,
			expectedExtraHosts: expectedPauseExtraHosts,
		},
	}
	// Add test cases for other containers expecting no modifications
	for _, container := range scTask.Containers {
		if container != pauseContainer {
			testcases = append(testcases, testCase{name: container.Name, container: container})
		}
	}
	scManager := NewManager()

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			hostConfig := &dockercontainer.HostConfig{}
			err := scManager.AugmentTaskContainer(scTask, tc.container, hostConfig)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.expectedExtraHosts, hostConfig.ExtraHosts)
		})
	}
}

func TestAgentContainerModificationsForServiceConnect_NonPrivileged(t *testing.T) {
	testAgentContainerModificationsForServiceConnect(t, false)
}