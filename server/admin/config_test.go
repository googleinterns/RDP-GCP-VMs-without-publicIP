package admin

import (
	"fmt"
	"reflect"
	"testing"
)

func buildTestConfig() Config {
	configVariable := ConfigVariable{Type: "string"}

	commonVariables := make(map[string]ConfigVariable)
	commonVariables["TEST_COMMON"] = configVariable

	commandVariables := make(map[string]ConfigVariable)
	commandVariables["TEST_COMMAND"] = configVariable

	configCommand := ConfigAdminCommand{Name: "test-cmd", Command: "$TEST_COMMON $TEST_COMMAND", Variables: commandVariables}
	return Config{Commands: []ConfigAdminCommand{configCommand}, CommonVariables: commonVariables}
}

func TestCheckConfigForMissingVariables(t *testing.T) {
	config := buildTestConfig()
	config.Commands[0].Command = "$TEST_COMMON $TEST_COMMAND $TEST_MISSING"

	expected := make(map[string][]string)
	expected["test-cmd"] = []string{"TEST_MISSING"}

	if missing := checkConfigForMissingVariables(config); !reflect.DeepEqual(expected, missing) {
		t.Errorf("checkConfigForMissingVariables didn't return the right value, got %v, expected %v", missing, expected)
	}

	config.Commands[0].Command = "$TEST_COMMON $TEST_COMMAND"

	delete(expected, "test-cmd")

	if missing := checkConfigForMissingVariables(config); !reflect.DeepEqual(expected, missing) {
		t.Errorf("checkConfigForMissingVariables didn't return the right value, got %v, expected %v", missing, expected)
	}
}

func TestGetMissingVariables(t *testing.T) {
	variablesFound := make(map[string]string)
	var missingVariables []string
	variablesToCheck := make(map[string]ConfigVariable)
	variablesToCheck["REQUIRED"] = ConfigVariable{Type: "string"}
	variablesToCheck["OPTIONAL"] = ConfigVariable{Type: "string", Optional: true}

	commandVariables := make(map[string]string)
	commandVariables["REQUIRED"] = "required"
	commandVariables["OPTIONAL"] = "optional"

	expected := make(map[string]string)
	expected["REQUIRED"] = "required"
	expected["OPTIONAL"] = "optional"
	getMissingVariables(variablesFound, commandVariables, variablesToCheck, &missingVariables)
	if len(missingVariables) > 0 {
		t.Errorf("getMissingVariables put non missing variable in missing, got %v, expected empty slice", missingVariables)
	}
	if !reflect.DeepEqual(variablesFound, expected) {
		t.Errorf("getMissingVariables didn't set correct found variables, got %v, expected %v", variablesFound, expected)
	}

	delete(commandVariables, "OPTIONAL")
	expected["OPTIONAL"] = ""
	getMissingVariables(variablesFound, commandVariables, variablesToCheck, &missingVariables)
	if len(missingVariables) > 0 {
		t.Errorf("getMissingVariables put non missing variable in missing, got %v, expected empty slice", missingVariables)
	}
	if !reflect.DeepEqual(variablesFound, expected) {
		t.Errorf("getMissingVariables didn't set correct found variables when optional variable was not given, got %v, expected %v", variablesFound, expected)
	}

	variablesFound = make(map[string]string)
	delete(commandVariables, "REQUIRED")
	delete(expected, "REQUIRED")
	getMissingVariables(variablesFound, commandVariables, variablesToCheck, &missingVariables)
	if len(missingVariables) == 0 {
		t.Errorf("getMissingVariables didn't put missing variable in missingVariables, got empty slice, expected %v", []string{"REQUIRED"})
	}
	if !reflect.DeepEqual(variablesFound, expected) {
		t.Errorf("getMissingVariables didn't set correct found variables when required variable was not given, got %v, expected %v", variablesFound, expected)
	}
}

func TestReadAdminCommand(t *testing.T) {
	config := buildTestConfig()
	variables := make(map[string]string)
	command := CommandToFill{Name: "COMMAND_NOT_FOUND", Variables: variables}

	if _, err := ReadAdminCommand(command, &config); err.Error() != fmt.Sprintf(commandNotFoundError, command.Name) {
		t.Errorf("ReadAdminCommand didn't error out on invalid command, got %v, expected %v", err, fmt.Errorf(commandNotFoundError, command.Name))
	}

	command.Name = "test-cmd"
	if _, err := ReadAdminCommand(command, &config); err.Error() != fmt.Sprintf(missingVariablesError, "TEST_COMMON, TEST_COMMAND") {
		t.Errorf("ReadAdminCommand didn't error out on missing variables, got %v, expected %v", err, fmt.Errorf(missingVariablesError, "TEST_COMMON, TEST_COMMAND"))
	}

	config.Commands[0].Command = "$TEST_COMMON $TEST_COMMAND --optional=$OPTIONAL"
	config.Commands[0].Variables["OPTIONAL"] = ConfigVariable{Type: "string", Optional: true}

	command.Variables["TEST_COMMON"] = "test1"
	command.Variables["TEST_COMMAND"] = "test2"
	if commandToRun, err := ReadAdminCommand(command, &config); commandToRun.Command != "test1 test2" || err != nil {
		t.Errorf("ReadAdminCommand didn't set command properly, got %v, expected %v", commandToRun.Command, "test1 test2")
	}

	command.Variables["OPTIONAL"] = "optional"
	if commandToRun, err := ReadAdminCommand(command, &config); commandToRun.Command != "test1 test2 --optional=optional" || err != nil {
		t.Errorf("ReadAdminCommand didn't set command properly, got %v, expected %v", commandToRun.Command, "test1 test2 --optional=optional")
	}
}
