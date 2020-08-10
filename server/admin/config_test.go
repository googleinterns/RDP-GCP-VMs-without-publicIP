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

package admin

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

var validComputeInstance = []byte(
	`{
		  "id": "test",
		  "name": "test",
		  "status": "RUNNING",
		  "description": "",
		  "zone": "https://www.googleapis.com/compute/v1/projects/test/zones/us-central1-a",
		  "disks": [
			{
			  "guestOsFeatures": [
				{
				  "type": "VIRTIO_SCSI_MULTIQUEUE"
				},
				{
				  "type": "WINDOWS"
				},
				{
				  "type": "MULTI_IP_SUBNET"
				},
				{
				  "type": "UEFI_COMPATIBLE"
				}
			  ]
			}
		  ],
		  "networkInterfaces": [
			{
			  "name": "nic0",
			  "network": "https://www.googleapis.com/compute/v1/projects/test/global/networks/default",
			  "networkIP": "10.128.0.2"
			}
		  ],
		  "project": "project",
		  "firewallNetwork": ""
		}`)

func buildTestConfig() Config {
	param := configParam{}

	commonParams := make(map[string]configParam)
	commonParams["TEST_COMMON"] = param

	operationParams := make(map[string]configParam)
	operationParams["TEST_COMMAND"] = param

	configOperation := ConfigAdminOperation{Name: "test-cmd", Operation: "${{TEST_COMMON}} ${{TEST_COMMAND}}", Params: operationParams}

	configInstanceOperation := ConfigAdminOperation{Name: "test-instance", Operation: "${{NAME}} ${{ZONE}} ${{NETWORKIP}} ${{PROJECT}}"}

	configPreRDPOperation := preRdpOperation{Name: "test-rdp", Operation: "hello"}

	return Config{Operations: []ConfigAdminOperation{configOperation}, CommonParams: commonParams, InstanceOperations: []ConfigAdminOperation{configInstanceOperation}, PreRDPOperations: []preRdpOperation{configPreRDPOperation}}
}

func TestCheckConfigForMissingParams(t *testing.T) {
	config := buildTestConfig()
	config.Operations[0].Operation = "${{TEST_COMMON}} ${{TEST_COMMAND}} ${{TEST_MISSING}}"
	config.ProjectOperation = "${{MISSING_PROJECT}}"
	config.ValidateProjectOperation = "${{MISSING_VALIDATE}}"
	expected := make(map[string][]string)
	expected["test-cmd"] = []string{"TEST_MISSING"}
	expected["config-project-operation"] = []string{"MISSING_PROJECT"}
	expected["config-validate-project-operation"] = []string{"MISSING_VALIDATE"}

	if missing := checkConfigForMissingParams(config); !reflect.DeepEqual(expected, missing) {
		t.Errorf("checkConfigForMissingParams didn't return the right value, got %v, expected %v", missing, expected)
	}

	config.Operations[0].Operation = "${{TEST_COMMON}} ${{TEST_COMMAND}}"

	delete(expected, "test-cmd")

	if missing := checkConfigForMissingParams(config); !reflect.DeepEqual(expected, missing) {
		t.Errorf("checkConfigForMissingParams didn't return the right value, got %v, expected %v", missing, expected)
	}
}

func TestValidateConfigDependencies(t *testing.T) {
	config := buildTestConfig()

	commonDeps := make(map[string]string)
	commonDeps["MISSING"] = "yes"

	config.CommonParams["TEST_COMMON"] = configParam{Optional: true, Dependencies: commonDeps}

	operationDeps := make(map[string]string)
	operationDeps["PARAM1"] = "yes"
	config.Operations[0].Operation = "${{PARAM1}} ${{PARAM2}}"

	paramWithDeps := configParam{Optional: true, Dependencies: operationDeps}
	config.Operations[0].Params["PARAM2"] = paramWithDeps

	config.PreRDPOperations[0].Dependencies = operationDeps

	expected := make(map[string][]string)
	expected["TEST_COMMON"] = []string{"MISSING"}
	expected["test-cmd"] = []string{"PARAM1"}
	expected["test-rdp"] = []string{"PARAM1"}

	if missing := validateConfigDependencies(config); !reflect.DeepEqual(expected, missing) {
		t.Errorf("validateConfigDependencies didn't return the right value, got %v, expected %v", missing, expected)
	}

	config.CommonParams["PARAM1"] = configParam{}
	config.CommonParams["MISSING"] = configParam{}

	delete(expected, "TEST_COMMON")
	delete(expected, "test-cmd")
	delete(expected, "test-rdp")

	if missing := validateConfigDependencies(config); !reflect.DeepEqual(expected, missing) {
		t.Errorf("validateConfigDependencies didn't return the right value, got %v, expected %v", missing, expected)
	}

}

func TestGetMissingParams(t *testing.T) {
	paramsFound := make(map[string]string)
	var missingParams []string
	paramsToCheck := make(map[string]configParam)
	paramsToCheck["REQUIRED"] = configParam{}
	paramsToCheck["OPTIONAL"] = configParam{Optional: true}

	operationParams := make(map[string]string)
	operationParams["REQUIRED"] = "required"
	operationParams["OPTIONAL"] = "optional"

	expected := make(map[string]string)
	expected["REQUIRED"] = "required"
	expected["OPTIONAL"] = "optional"
	getMissingParams(paramsFound, operationParams, paramsToCheck, &missingParams)
	if len(missingParams) > 0 {
		t.Errorf("getMissingParams put non missing param in missing, got %v, expected empty slice", missingParams)
	}
	if !reflect.DeepEqual(paramsFound, expected) {
		t.Errorf("getMissingParams didn't set correct found params, got %v, expected %v", paramsFound, expected)
	}

	delete(operationParams, "OPTIONAL")
	expected["OPTIONAL"] = ""
	getMissingParams(paramsFound, operationParams, paramsToCheck, &missingParams)
	if len(missingParams) > 0 {
		t.Errorf("getMissingParams put non missing param in missing, got %v, expected empty slice", missingParams)
	}
	if !reflect.DeepEqual(paramsFound, expected) {
		t.Errorf("getMissingParams didn't set correct found params when optional param was not given, got %v, expected %v", paramsFound, expected)
	}

	paramsFound = make(map[string]string)
	delete(operationParams, "REQUIRED")
	delete(expected, "REQUIRED")
	getMissingParams(paramsFound, operationParams, paramsToCheck, &missingParams)
	if len(missingParams) == 0 {
		t.Errorf("getMissingParams didn't put missing param in missingParams, got empty slice, expected %v", []string{"REQUIRED"})
	}
	if !reflect.DeepEqual(paramsFound, expected) {
		t.Errorf("getMissingParams didn't set correct found params when required param was not given, got %v, expected %v", paramsFound, expected)
	}
}

func TestCheckMissingDependencies(t *testing.T) {
	config := buildTestConfig()
	params := make(map[string]string)
	params["REQUIRED_DEP"] = "true"
	params["OPTIONAL"] = ""

	deps := make(map[string]string)
	deps["REQUIRED_DEP"] = "true"

	config.Operations[0].Params["OPTIONAL"] = configParam{Optional: true, Dependencies: deps}

	var missingDependencies []string
	expected := []string{"OPTIONAL"}

	checkMissingDependencies(params, config.CommonParams, config.Operations[0].Params, &missingDependencies)
	if !reflect.DeepEqual(missingDependencies, expected) {
		t.Errorf("checkMissingDependencies didn't set proper missing deps, got %v, expected %v", missingDependencies, expected)
	}

	missingDependencies = []string{}
	expected = []string{}
	params["OPTIONAL"] = "value"

	checkMissingDependencies(params, config.CommonParams, config.Operations[0].Params, &missingDependencies)
	if !reflect.DeepEqual(missingDependencies, expected) {
		t.Errorf("checkMissingDependencies didn't set proper missing deps, got %v, expected %v", missingDependencies, expected)
	}
}

func TestReadAdminOperation(t *testing.T) {
	config := buildTestConfig()
	params := make(map[string]string)
	operation := OperationToFill{Name: "COMMAND_NOT_FOUND", Params: params}

	if _, err := ReadAdminOperation(operation, &config); err.Error() != fmt.Sprintf(operationNotFoundError, operation.Name) {
		t.Errorf("ReadAdminOperation didn't error out on invalid operation, got %v, expected %v", err, fmt.Errorf(operationNotFoundError, operation.Name))
	}

	operation.Name = "test-cmd"
	if _, err := ReadAdminOperation(operation, &config); err.Error() != fmt.Sprintf(missingParamsError, "TEST_COMMON, TEST_COMMAND") {
		t.Errorf("ReadAdminOperation didn't error out on missing params, got %v, expected %v", err, fmt.Errorf(missingParamsError, "TEST_COMMON, TEST_COMMAND"))
	}

	config.Operations[0].Operation = "${{TEST_COMMON}} ${{TEST_COMMAND}} --optional=${{OPTIONAL}}"
	config.Operations[0].Params["OPTIONAL"] = configParam{Optional: true}

	operation.Params["TEST_COMMON"] = "test1"
	operation.Params["TEST_COMMAND"] = "test2"
	if operationToRun, err := ReadAdminOperation(operation, &config); operationToRun.Operation != "test1 test2" || err != nil {
		t.Errorf("ReadAdminOperation didn't set operation properly, got %v, expected %v", operationToRun.Operation, "test1 test2")
	}

	operation.Params["OPTIONAL"] = "optional"
	if operationToRun, err := ReadAdminOperation(operation, &config); operationToRun.Operation != "test1 test2 --optional=optional" || err != nil {
		t.Errorf("ReadAdminOperation didn't set operation properly, got %v, expected %v", operationToRun.Operation, "test1 test2 --optional=optional")
	}
}

func TestCaptureParamsFromInstanceOperation(t *testing.T) {
	var instance Instance

	json.Unmarshal(validComputeInstance, &instance)

	params := make(map[string]string)
	params["COMMON"] = "COMMON"

	instanceOperation := InstanceOperationToFill{Instance: instance, Params: params}

	successOperation := "${{NAME}} ${{ZONE}} ${{NETWORKIP}} ${{PROJECT}} ${{COMMON}}"

	variables := make(map[string]string)

	successVariables := make(map[string]string)
	successVariables["NAME"] = instance.Name
	successVariables["ZONE"] = instance.Zone
	successVariables["NETWORKIP"] = instance.NetworkInterfaces[0].IP
	successVariables["PROJECT"] = instance.ProjectName
	successVariables["COMMON"] = "COMMON"

	missingParams := captureParamsFromInstanceOperation(variables, instanceOperation, successOperation)
	if len(missingParams) != 0 {
		t.Errorf("captureParamsFromInstanceOperation had missingParams for a proper operation: %s", strings.Join(missingParams, ", "))
	}

	if !reflect.DeepEqual(variables, successVariables) {
		t.Errorf("captureParamsFromInstanceOperation didn't set variables properly, got %s, expected %s", variables, successVariables)
	}

	missingParamInOperation := "${{NAME}} ${{ZONE}} ${{NETWORKIP}} ${{PROJECT}} ${{MISSING}}"

	missingParams = captureParamsFromInstanceOperation(variables, instanceOperation, missingParamInOperation)
	if len(missingParams) != 1 {
		t.Errorf("captureParamsFromInstanceOperation didn't had correct number missingParams for a proper operation: %s", strings.Join(missingParams, ", "))
	}

	if (missingParams[0]) != "MISSING" {
		t.Errorf("captureParamsFromInstanceOperation didn't set missing parameter properly, got %v, expected %v", missingParams[0], "MISSING")
	}

}

func TestReadInstanceOperation(t *testing.T) {
	config := buildTestConfig()

	var instance Instance
	json.Unmarshal(validComputeInstance, &instance)

	expectedOperation := fmt.Sprintf("%s %s %s %s", instance.Name, instance.Zone, instance.NetworkInterfaces[0].IP, instance.ProjectName)

	operation := InstanceOperationToFill{Name: "COMMAND_NOT_FOUND", Instance: instance}

	instanceOperation := config.InstanceOperations[0]
	operation.Name = "test-instance"
	if operationToRun, err := ReadInstanceOperation(operation, instanceOperation); operationToRun.Operation != expectedOperation || err != nil {
		t.Errorf("ReadInstanceOperation didn't set operation properly, got %v, expected %v", operationToRun.Operation, expectedOperation)
	}

	config.InstanceOperations[0].Operation = "${{MISSING}}"
	if _, err := ReadInstanceOperation(operation, config.InstanceOperations[0]); err.Error() != fmt.Sprintf(missingInstanceParamsError, "MISSING") {
		t.Errorf("ReadInstanceOperation didn't set error properly, got %v, expected %v", err.Error(), fmt.Sprintf(missingInstanceParamsError, "MISSING"))
	}
}
