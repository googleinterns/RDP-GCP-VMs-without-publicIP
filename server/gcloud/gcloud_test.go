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

package gcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

var (
	invalidAuthOutput          = []byte("ERROR: (gcloud.compute.instances.list)         There was a problem refreshing your current auth tokens: Failed to retrieve token from the Google Compute Enginemetadata service. Response:\n{'metadata-flavor': 'Google', 'content-type': 'application/json', 'date': 'Sat, 20 Jun 2020 00:08:01 GMT', 'server': 'Metadata Server for VM', 'content-length': '38', 'x-xss-protection': '0', 'x-frame-options': 'SAMEORIGIN', 'status': '404'}\n        Please run:\n\n          $ gcloud auth login\n\n        to obtain new credentials.\nIf you have already logged in with a different account:\n\n    $ gcloud config set account ACCOUNT\n\nto select an already authenticated account to use.\n")
	invalidProjectOutput       = []byte("ERROR: (gcloud.compute.instances.list) Some requests did not succeed:\n - Failed to find project badinvalidproject\n")
	validComputeInstanceOutput = []byte(`[
{
    "canIpForward": false,
    "cpuPlatform": "Intel Broadwell",
    "creationTimestamp": "2020-06-03T12:34:16.025-07:00",
    "deletionProtection": false,
    "description": "",
    "disks": [
      {
        "autoDelete": true,
        "boot": true,
        "deviceName": "",
        "diskSizeGb": "50",
        "guestOsFeatures": [
          {
            "type": "VIRTIO_SCSI_MULTIQUEUE"
          },
          {
            "type": "UEFI_COMPATIBLE"
          },
          {
            "type": "MULTI_IP_SUBNET"
          },
          {
            "type": "WINDOWS"
          }
        ],
        "index": 0,
        "interface": "SCSI",
        "kind": "compute#attachedDisk",
        "licenses": [
          "https://www.googleapis.com/compute/v1/projects/windows-cloud/global/licenses/windows-server-2019-dc"
        ]
      }
    ],
    "displayDevice": {
      "enableDisplay": false
    },
    "fingerprint": "",
    "id": "",
    "kind": "compute#instance",
    "labelFingerprint": "",
    "machineType": "https://www.googleapis.com/compute/v1/projects/zones/us-west1-b/machineTypes/custom-1-7680-ext",
    "name": "test-project",
    "networkInterfaces": [
      {
        "fingerprint": "",
        "kind": "compute#networkInterface",
        "name": "nic0",
        "network": "default",
        "networkIP": "",
        "subnetwork": ""
      }
    ],
    "reservationAffinity": {
      "consumeReservationType": "ANY_RESERVATION"
    },
    "scheduling": {
      "automaticRestart": true,
      "onHostMaintenance": "MIGRATE",
      "preemptible": false
    },
    "shieldedInstanceConfig": {
      "enableIntegrityMonitoring": true,
      "enableSecureBoot": false,
      "enableVtpm": true
    },
    "shieldedInstanceIntegrityPolicy": {
      "updateAutoLearnPolicy": true
    },
    "startRestricted": false,
    "status": "RUNNING",
    "tags": {
      "fingerprint": ""
    },
    "zone": "https://www.googleapis.com/compute/v1/projects/project-name/zones/us-west1-b",
    "project": "project-name",
    "firewallNetwork": "default"
  }
]`)
)

type mockShell struct{}

func (*mockShell) ExecuteCmd(cmd string) ([]byte, error) {
	if cmd == fmt.Sprintf("%s%s", getComputeInstancesForProjectPrefix, "validProject") {
		return validComputeInstanceOutput, nil
	}
	if cmd == fmt.Sprintf("%s%s", getComputeInstancesForProjectPrefix, "invalidAuth") {
		return invalidAuthOutput, errors.New("error")
	}
	if cmd == fmt.Sprintf("%s%s", getComputeInstancesForProjectPrefix, "invalidProject") {
		return invalidProjectOutput, errors.New("error")
	}
	if cmd == fmt.Sprintf(rdpProgramCmd, 9999, "quit", "password") {
		return []byte("output"), nil
	}
	if cmd == fmt.Sprintf(rdpProgramCmd, 9999, "error", "password") {
		return []byte("output"), errors.New("error")
	}
	if cmd == fmt.Sprintf(iapFirewallCreateCmd, "test-project", "test-project", "auth-error", "default") {
		return []byte(gcloudAuthError), errors.New("error")
	}
	if cmd == fmt.Sprintf(iapFirewallCreateCmd, "test-project", "test-project", "project-error", "default") {
		return []byte(projectCmdError), errors.New("error")
	}
	if cmd == fmt.Sprintf(iapFirewallCreateCmd, "test-project", "test-project", "exists", "default") {
		return []byte(fmt.Sprintf(firewallRuleExistsCmdOutput, "exists", "test-project")), errors.New("error")
	}
	if cmd == fmt.Sprintf(iapFirewallCreateCmd, "test-project", "test-project", "valid", "default") {
		return []byte(""), nil
	}

	return nil, nil
}

func (*mockShell) ExecuteCmdWithContext(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

func (*mockShell) ExecuteCmdReader(cmd string) ([]io.ReadCloser, context.CancelFunc, error) {
	var instanceToUse Instance
	json.Unmarshal(instance, &instanceToUse)
	if cmd == fmt.Sprintf(iapTunnelCmd, "test-project", "invalid", 9999, instanceToUse.Zone) {
		return []io.ReadCloser{ioutil.NopCloser(strings.NewReader("")), ioutil.NopCloser(strings.NewReader(gcloudErrorOutput))}, nil, nil
	}
	if cmd == fmt.Sprintf(iapTunnelCmd, "test-project", "valid", 9999, instanceToUse.Zone) {
		return []io.ReadCloser{ioutil.NopCloser(strings.NewReader("")), ioutil.NopCloser(strings.NewReader(tunnelCreatedOutput))}, nil, nil
	}
	return nil, nil, nil
}

// TestGetComputeInstances tests the CmdReader which outputs stdout/stderr as a ReadCloser
func TestGetComputeInstances(t *testing.T) {
	g := NewGcloudExecutor(&mockShell{})
	if _, err := g.GetComputeInstances("invalidAuth"); err == nil || err.Error() != SdkAuthError {
		t.Errorf("GetComputeInstances didn't error on invalid auth")
	}

	if _, err := g.GetComputeInstances("invalidProject"); err == nil || err.Error() != SdkProjectError {
		t.Errorf("GetComputeInstances didn't error on invalid project")
	}

	var validInstances []Instance
	err := json.Unmarshal(validComputeInstanceOutput, &validInstances)
	if err != nil {
		t.Errorf("GetComputeInstances JSON unmarshal error with test data")
	}

	if instances, err := g.GetComputeInstances("validProject"); err != nil || !reflect.DeepEqual(instances, validInstances) {
		t.Errorf("GetComputeInstances failed getting instances, got %v, expected %v", instances, validInstances)
	}
}
