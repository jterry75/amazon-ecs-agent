//go:build windows
// +build windows

// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//      http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package daemonmanager

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	apicontainer "github.com/aws/amazon-ecs-agent/agent/api/container"
	apicontainerstatus "github.com/aws/amazon-ecs-agent/agent/api/container/status"
	apitask "github.com/aws/amazon-ecs-agent/agent/api/task"
	apitaskstatus "github.com/aws/amazon-ecs-agent/agent/api/task/status"
	"github.com/aws/amazon-ecs-agent/agent/dockerclient/dockerapi"
	"github.com/aws/amazon-ecs-agent/ecs-agent/logger"
	"github.com/aws/amazon-ecs-agent/ecs-agent/logger/field"

	"github.com/docker/docker/api/types"
	"github.com/pborman/uuid"
)

func (dm *daemonManager) CreateDaemonTask() (*apitask.Task, error) {
	// TODO: how much of this do I really need to fill out?
	resp := &apitask.Task{
		// TODO: Arn with UUID seems a problem because you could have more than one MT per MD type.
		Arn:                 fmt.Sprintf("arn:::::/%s-%s", dm.managedDaemon.GetImageName(), uuid.NewUUID()),
		DesiredStatusUnsafe: apitaskstatus.TaskRunning,
		Containers: []*apicontainer.Container{
			{
				Type:                      apicontainer.ContainerManagedDaemon,
				TransitionDependenciesMap: make(map[apicontainerstatus.ContainerStatus]apicontainer.TransitionDependencySet),
				Essential:                 true,
			},
		},
		LaunchType:  "EC2",
		NetworkMode: apitask.HostNetworkMode,
		IsInternal:  true,
	}
	return resp, nil
}

func (dm *daemonManager) LoadImage(ctx context.Context, _ dockerapi.DockerClient) (*types.ImageInspect, error) {
	isLoaded, err := dm.IsLoaded(nil)
	if err != nil {
		return nil, err
	}
	if !isLoaded {
		err := dm.writeTarToDestinationDir()
		if err != nil {
			dm.cleanupDestinationDir()
			return nil, err
		}
	}
	// TODO: How much of this do we need to fill out.
	var resp = &types.ImageInspect{}
	return resp, nil
}

// IsLoaded returns if the ManagedDaemon install marker file exists or not.
func (dm *daemonManager) IsLoaded(_ dockerapi.DockerClient) (bool, error) {
	// TODO err provides no value on this func that I can tell for Windows or Linux
	i, err := os.Stat(dm.destinationDirMarkerPath())
	if err != nil {
		return false, nil
	}
	return !i.IsDir(), nil
}

func (dm *daemonManager) tarPathToDestinationDir() string {
	return strings.TrimSuffix(dm.managedDaemon.GetImageTarPath(), ".tar")
}

func (dm *daemonManager) destinationDirMarkerPath() string {
	return filepath.Join(dm.tarPathToDestinationDir(), "installed.marker")
}

// writeTarToDestinationDir writes all files within the ManagedDaemon tar path
// to the install dir. After successful extraction writes a file
// "installed.marker" file to the dir to signify successful completion incase
// retries are attempted.
//
// No cleanup on failure is preformed by this function. On failure if cleanup is
// desired call `cleanupDestinationDir`.
func (dm *daemonManager) writeTarToDestinationDir() error {
	daemonImageToLoad := dm.managedDaemon.GetImageName()
	daemonImageTarPath := dm.managedDaemon.GetImageTarPath()
	daemonImageInstallPath := dm.tarPathToDestinationDir()
	mdTar, err := os.Open(fmt.Sprintf(daemonImageTarPath))
	if err != nil {
		err = fmt.Errorf("failed to open ManagedDaemon tar for read: %w", err)
		logger.Warn(fmt.Sprintf("%s container tarball unavailable at path: %s", daemonImageToLoad, daemonImageTarPath), logger.Fields{
			field.Error: err,
		})
		return err
	}
	defer mdTar.Close() // Close source tar on return
	rdr := tar.NewReader(mdTar)
	for {
		header, err := rdr.Next()
		if err != nil {
			if err == io.EOF {
				break // we are done reading
			}
			// Unexpected error, return failure
			return fmt.Errorf("unknown error when processing ManagedDaemon tar: %w", err)
		}
		// TODO: is there a security risk here on join? Can Name in tar format have ../.. type paths
		targetPath := filepath.Join(daemonImageInstallPath, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(targetPath); err != nil {
				if err := os.MkdirAll(targetPath, 0755); err != nil { // 0755 is unneded on Windows but we do this everwhere for consistency
					return fmt.Errorf("failed to create directory in ManagedDaemon install path: %w", err)
				}
			}
		case tar.TypeReg:
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode)) // TODO: I wonder if we should always write these restricted and not care about tar mode?
			if err != nil {
				return fmt.Errorf("failed to create file in ManagedDaemon install path: %w", err)
			}
			// Dont defer in nested scope to avoid delayed close
			if _, err := io.Copy(outFile, rdr); err != nil {
				outFile.Close() // Failed Write, close in scope
				return fmt.Errorf("failed to copy file contents from tar to ManagedDaemon install path: %w", err)
			}
			outFile.Close() // Successful Write, close in scope
		default:
			panic(fmt.Sprintf("Windows ManagedDaemon Tar contained unexpected type: %s", string(header.Typeflag)))
		}
	}
	// Write the installed marker file into the dir
	mFile, err := os.Create(dm.destinationDirMarkerPath())
	if err != nil {
		return fmt.Errorf("failed to write maker file to ManagedDaemon install path: %w", err)
	}
	mFile.Close()
	return nil
}

func (dm *daemonManager) cleanupDestinationDir() {
	installDir := dm.tarPathToDestinationDir()
	err := os.RemoveAll(installDir)
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to cleanup DaemonManager path: %s", installDir), logger.Fields{
			field.Error: err,
		})
	}
}
