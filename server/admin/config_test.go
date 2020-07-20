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
	"fmt"
	"reflect"
	"testing"
)

func buildTestConfig() Config {
	param := configParam{}

	commonParams := make(map[string]configParam)
	commonParams["TEST_COMMON"] = param

	operationParams := make(map[string]configParam)
	operationParams["TEST_COMMAND"] = param

	configOperation := configAdminOperation{Name: "test-cmd", Operation: "$TEST_COMMON $TEST_COMMAND", Params: operationParams}
	return Config{Operations: []configAdminOperation{configOperation}, CommonParams: commonParams}
}

func TestCheckConfigForMissingParams(t *testing.T) {
	config := buildTestConfig()
	config.Operations[0].Operation = "$TEST_COMMON $TEST_COMMAND $TEST_MISSING"

	expected := make(map[string][]string)
	expected["test-cmd"] = []string{"TEST_MISSING"}

	if missing := checkConfigForMissingParams(config); !reflect.DeepEqual(expected, missing) {
		t.Errorf("checkConfigForMissingParams didn't return the right value, got %v, expected %v", missing, expected)
	}

	config.Operations[0].Operation = "$TEST_COMMON $TEST_COMMAND"

	delete(expected, "test-cmd")

	if missing := checkConfigForMissingParams(config); !reflect.DeepEqual(expected, missing) {
		t.Errorf("checkConfigForMissingParams didn't return the right value, got %v, expected %v", missing, expected)
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

	config.Operations[0].Operation = "$TEST_COMMON $TEST_COMMAND --optional=$OPTIONAL"
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
