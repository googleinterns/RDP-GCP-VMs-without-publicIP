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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
)


var invalidAuthOutput = []byte("ERROR: (gcloud.compute.instances.list)         There was a problem refreshing your current auth tokens: Failed to retrieve token from the Google Compute Enginemetadata service. Response:\n{'metadata-flavor': 'Google', 'content-type': 'application/json', 'date': 'Sat, 20 Jun 2020 00:08:01 GMT', 'server': 'Metadata Server for VM', 'content-length': '38', 'x-xss-protection': '0', 'x-frame-options': 'SAMEORIGIN', 'status': '404'}\n        Please run:\n\n          $ gcloud auth login\n\n        to obtain new credentials.\nIf you have already logged in with a different account:\n\n    $ gcloud config set account ACCOUNT\n\nto select an already authenticated account to use.\n")
var invalidProjectOutput = []byte("ERROR: (gcloud.compute.instances.list) Some requests did not succeed:\n - Failed to find project badinvalidproject\n")
var validComputeInstanceOutput = []byte(`[
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
    "name": "project-name",
    "networkInterfaces": [
      {
        "fingerprint": "",
        "kind": "compute#networkInterface",
        "name": "nic0",
        "network": "",
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
    "zone": "https://www.googleapis.com/compute/v1/projects/project-name/zones/us-west1-b"
  }
]`)

func mockCmdAuthFailure(_ string) ([]byte, error) {
	return invalidAuthOutput, errors.New("error")
}

func mockCmdProjectFailure(_ string) ([]byte, error) {
	return invalidProjectOutput, errors.New("error")
}

func mockCmdComputeInstances(_ string) ([]byte, error) {
	return validComputeInstanceOutput, nil
}

// TestGetComputeInstances tests the CmdReader which outputs stdout/stderr as a ReadCloser
func TestGetComputeInstances(t *testing.T) {
	g := NewCmdRunner(mockCmdAuthFailure)
	if _, err := g.GetComputeInstances("invalidAuth"); err == nil || err.Error() != SdkAuthError {
		t.Errorf("GetComputeInstances didn't error on invalid auth")
	}

	g = NewCmdRunner(mockCmdProjectFailure)
	if _, err := g.GetComputeInstances("invalidProject"); err == nil || err.Error() != SdkProjectError {
		t.Errorf("GetComputeInstances didn't error on invalid project")
	}

	var validInstances []Instance
	err := json.Unmarshal(validComputeInstanceOutput, &validInstances)
	if err != nil {
		t.Errorf("GetComputeInstances JSON unmarshal error with test data")
	}

	g = NewCmdRunner(mockCmdComputeInstances)
	if instances, err := g.GetComputeInstances("validProject"); err != nil || !reflect.DeepEqual(instances, validInstances) {
		t.Errorf("GetComputeInstances failed getting instances, got %v, expected %v", instances, validInstances)
	}
}
