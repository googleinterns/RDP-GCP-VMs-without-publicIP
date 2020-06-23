/***
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
***/

// Package gcloud runs gCloud SDK commands like getting instances, starting IAP tunnels and firewalls
package gcloud

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const (
	gcloudAuthError string = "there was a problem refreshing your current auth tokens"
	projectCmdError string = "failed to find project"
	// SdkAuthError is returned if there is an gcloud SDK auth error
	SdkAuthError string = "gCloud SDK auth invalid"
	// SdkProjectError is returned if the gcloud project given is invalid
	SdkProjectError                     string = "gCloud SDK project invalid"
	getComputeInstancesForProjectPrefix string = "gcloud compute instances list --format=json --project="
)

type osFeatures struct {
	Type string `json:"type"`
}

type disk struct {
	OSFeatures []osFeatures `json:"guestOsFeatures"`
}

type networkInterfaces struct {
	Name    string `json:"name"`
	Network string `json:"network"`
	IP      string `json:"networkIP"`
}

// Instance is used as a structure for gcloud compute instances.
type Instance struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	Status            string              `json:"status"`
	Description       string              `json:"description"`
	Zone              string              `json:"zone"`
	Disk              []disk              `json:"disks"`
	NetworkInterfaces []networkInterfaces `json:"networkInterfaces"`
}

type shell interface {
	ExecuteCmd(string) ([]byte, error)
	ExecuteCmdReader(string) ([]io.ReadCloser, error)
}

// GcloudExecutor is used to call gcloud functions with the shell passed in.
type GcloudExecutor struct {
	shell shell
}

// NewGcloudExecutor creates a new gcloudExecutor struct with a struct that implements shell.
func NewGcloudExecutor(shell shell) *GcloudExecutor {
	return &GcloudExecutor{
		shell: shell,
	}
}

// GetComputeInstances runs the gCloud instances command, parses the output to the Instances struct and returns
func (gcloudExecutor *GcloudExecutor) GetComputeInstances(projectName string) ([]Instance, error) {
	instanceOutput, err := gcloudExecutor.shell.ExecuteCmd(getComputeInstancesForProjectPrefix + projectName)
	if err != nil {
		if stringOutput := strings.ToLower(string(instanceOutput)); strings.Contains(stringOutput, gcloudAuthError) {
			return nil, errors.New(SdkAuthError)
		} else if strings.Contains(stringOutput, projectCmdError) {
			return nil, errors.New(SdkProjectError)
		}
		return nil, err
	}

	var instances []Instance
	if err := json.Unmarshal(instanceOutput, &instances); err != nil {
		return nil, err
	}

	return instances, nil
}
