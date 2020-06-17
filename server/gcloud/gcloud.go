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
	"fmt"
	"github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/shell"
	"strings"
)

const (
	loginCmdError string = "gcloud auth login"
	projectCmdError string = "failed to find project"
	SdkAuthError string = "gCloud SDK auth invalid"
	SdkProjectError string = "gCloud SDK project invalid"
)

type Instance struct {
	Id string `json:"id"`
	Name string `json:"name"`
	Status string `json:"status"`
	Description string `json:"description"`
	Zone string `json:"zone"`
	Disk []struct {
		OSFeatures []struct {
			Type string `json:"type"`
		} `json:"guestOsFeatures"`
	} `json:"disks"`
	NetworkInterfaces []struct {
		Name string `json:"name"`
		Network string `json:"network"`
		Ip string `json:"networkIP"`
	}
}

func GetComputeInstances(projectName string) ([]Instance, error) {
	instanceOutput, err := shell.Cmd("gcloud compute instances list --format=json --project=" + projectName)
	if err != nil {
		stringOutput := strings.ToLower(string(instanceOutput))
		if strings.Contains(stringOutput, loginCmdError) {
			return nil, errors.New(SdkAuthError)
		}
		if strings.Contains(stringOutput, projectCmdError) {
			return nil, errors.New(SdkProjectError)
		}
		return nil, err
	}

	var instances []Instance
	if err := json.Unmarshal(instanceOutput, &instances); err != nil {
		return nil, err
	}

	fmt.Println(instances)
	return instances, nil
}
