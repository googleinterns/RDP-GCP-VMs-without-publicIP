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

// Package admin is used for the custom admin operation functionality including reading from config, setting up and running the operations.
package admin

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/gcloud"
	"github.com/spf13/viper"
)

const (
	configFileReadError                string = "Error reading configuration file, please make sure it has the necessary permissions and is in DIRECTORY"
	configFileDataError                string = "Error reading data from configuration file, please follow the format specified"
	configOperationMissingParams       string = "Config is missing variables for these operation(s): %s"
	configOperationMissingDependencies string = "Config has invalid dependencies for these common parameter(s) and operation(s): %s"
	operationNotFoundError             string = "%s operation was not found in the config"
	missingParamsError                 string = "Missing parameters defined in config file for this operation: %s"
	missingDependenciesError           string = "These parameters are required due to the dependencies for this operation: %s"
	missingInstanceParamsError         string = "Missing parameters in the instance needed for this operation: %s, use Admin Operations instead for this operation"
	captureParamRegex                  string = `\${{(?s)([A-Z]+_*[A-Z]+)}}(?s)`
)

// configParam points to a variable in the config file
type configParam struct {
	Default      string            `json:"default"`
	Optional     bool              `json:"optional"`
	Description  string            `json:"description"`
	Sample       string            `json:"sample"`
	Choices      []string          `json:"choices"`
	Dependencies map[string]string `json:"dependencies"`
}

// configAdminOperation points to a configured admin operation
type configAdminOperation struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Operation      string                 `json:"operation"`
	Params         map[string]configParam `json:"params"`
	RealtimeOutput bool                   `mapstructure:"realtime_output"`
}

type preRdpOperation struct {
	Name         string            `json:"name"`
	Operation    string            `json:"operation"`
	Dependencies map[string]string `json:"dependencies"`
}

type configWorkflow struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Operations  []string `json:"operations"`
}

// Config is the fully loaded config represented as structures
type Config struct {
	InstanceOperations       []configAdminOperation `json:"instance_operations"`
	Operations               []configAdminOperation `json:"operations"`
	CommonParams             map[string]configParam `json:"common_params"`
	ProjectOperation         string                 `json:"project_operation"`
	ValidateProjectOperation string                 `json:"validate_project_operation"`
	PreRDPOperations         []preRdpOperation      `json:"pre_rdp_operations"`
	Workflows                []configWorkflow       `json:"workflows"`
}

// OperationToFill is sent by the extension detailing a operation and the variables to be filled
type OperationToFill struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"variables"`
}

// ProjectOperationParams is sent by the extension detailing a operation and the variables to be filled
type ProjectOperationParams struct {
	Type        string            `json:"type"`
	ProjectName string            `json:"project_name"`
	Params      map[string]string `json:"variables"`
}

type InstanceOperationToFill struct {
	Name     string            `json:"name"`
	Instance gcloud.Instance   `json:"instance"`
	Params   map[string]string `json:"variables"`
}

// OperationToRun contains a ready to run operation with its status and a unique
type OperationToRun struct {
	Operation      string `json:"operation"`
	Hash           string `json:"hash"`
	Status         string `json:"status"`
	RealtimeOutput bool
}

func checkConfigForMissingParams(config Config) map[string][]string {
	missingParams := make(map[string][]string)

	r := regexp.MustCompile(captureParamRegex)
	for _, operation := range config.Operations {

		// Get all variables in the operation
		matches := r.FindAllStringSubmatch(operation.Operation, -1)
		for _, match := range matches {
			// Check if variable is defined in either common variables or the operation's variables
			if _, inCommonParams := config.CommonParams[match[1]]; !inCommonParams {
				if _, inCommandParams := operation.Params[match[1]]; !inCommandParams {
					missingParams[operation.Name] = append(missingParams[operation.Name], match[1])
				}
			}
		}
	}

	if config.ProjectOperation != "" {
		// Get all variables in the operation
		matches := r.FindAllStringSubmatch(config.ProjectOperation, -1)
		for _, match := range matches {
			// Check if variable is defined in commonparams
			if _, inCommonParams := config.CommonParams[match[1]]; !inCommonParams {
				missingParams["config-project-operation"] = append(missingParams["config-project-operation"], match[1])
			}
		}
	}

	if config.ValidateProjectOperation != "" {
		// Get all variables in the operation
		matches := r.FindAllStringSubmatch(config.ValidateProjectOperation, -1)
		for _, match := range matches {
			// Check if variable is defined in either commonparams
			if _, inCommonParams := config.CommonParams[match[1]]; !inCommonParams {
				missingParams["config-validate-project-operation"] = append(missingParams["config-validate-project-operation"], match[1])
			}
		}
	}

	for _, operation := range config.PreRDPOperations {

		// Get all variables in the operation
		matches := r.FindAllStringSubmatch(operation.Operation, -1)
		for _, match := range matches {
			// Check if variable is defined in common variables
			if _, inCommonParams := config.CommonParams[match[1]]; !inCommonParams {
				missingParams[operation.Name] = append(missingParams[operation.Name], match[1])
			}
		}
	}

	for _, operation := range config.InstanceOperations {
		// Get all variables in the operation
		matches := r.FindAllStringSubmatch(operation.Operation, -1)
		instanceParams := []string{"NAME", "ZONE", "PROJECT", "NETWORKIP"}
		for _, match := range matches {

			isInstanceParam := false

			for _, param := range instanceParams {
				if param == match[1] {
					isInstanceParam = true
					break
				}
			}

			// Check if variable is defined in common variables
			if _, inCommonParams := config.CommonParams[match[1]]; !inCommonParams && !isInstanceParam {
				missingParams[operation.Name] = append(missingParams[operation.Name], match[1])
			}
		}
	}

	return missingParams
}

func validateConfigDependencies(config Config) map[string][]string {
	missingDependencies := make(map[string][]string)

	for name, param := range config.CommonParams {
		for dependency, _ := range param.Dependencies {
			dependency = strings.ToUpper(dependency)
			if _, inCommonParams := config.CommonParams[dependency]; !inCommonParams {
				missingDependencies[name] = append(missingDependencies[name], dependency)
			}
		}
	}

	for _, operation := range config.Operations {
		for _, param := range operation.Params {
			for dependency, _ := range param.Dependencies {
				dependency = strings.ToUpper(dependency)
				if _, inCommonParams := config.CommonParams[dependency]; !inCommonParams {
					if _, inOperationParam := operation.Params[dependency]; !inOperationParam {
						missingDependencies[operation.Name] = append(missingDependencies[operation.Name], dependency)
					}
				}
			}
		}
	}

	for _, operation := range config.PreRDPOperations {
		for dependency, _ := range operation.Dependencies {
			dependency = strings.ToUpper(dependency)
			if _, inCommonParams := config.CommonParams[dependency]; !inCommonParams {
				missingDependencies[operation.Name] = append(missingDependencies[operation.Name], dependency)
			}
		}
	}

	return missingDependencies
}

// LoadConfig reads the config file and unmarshals the data to structs
func LoadConfig(configPath *string) (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(*configPath)
	viper.SetConfigType("yml")

	var config Config
	if err := viper.ReadInConfig(); err != nil {
		log.Println(err)
		return &Config{}, err
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Println(err)
		return &Config{}, errors.New(configFileDataError)
	}

	if config.CommonParams != nil {
		for key, value := range config.CommonParams {
			if key == strings.ToLower(key) {
				config.CommonParams[strings.ToUpper(key)] = value
				delete(config.CommonParams, key)
			}
		}
	}

	missingParams := checkConfigForMissingParams(config)

	if len(missingParams) > 0 {
		var errorStrings []string
		// Join all the missing variables in a list
		for key, val := range missingParams {
			errString := fmt.Sprintf("%s: %s", key, strings.Join(val, ", "))
			errorStrings = append(errorStrings, errString)
		}
		return &Config{}, fmt.Errorf(configOperationMissingParams, strings.Join(errorStrings, ". "))
	}

	missingDependencies := validateConfigDependencies(config)

	if len(missingDependencies) > 0 {
		var errorStrings []string
		// Join all the missing variables in a list
		for key, val := range missingDependencies {
			errString := fmt.Sprintf("%s: %s", key, strings.Join(val, ", "))
			errorStrings = append(errorStrings, errString)
		}
		return &Config{}, fmt.Errorf(configOperationMissingDependencies, strings.Join(errorStrings, ". "))
	}

	return &config, nil
}

func checkIfParamInChoices(value string, variableName string, variablesToCheck map[string]configParam) bool {
	for _, choice := range variablesToCheck[variableName].Choices {
		if choice == value {
			return true
		}
	}

	return false
}

// getMissingParams checks variables to the current ones in the operation either adding them to missingParams or variablesFound
func getMissingParams(variablesFound map[string]string, variablesInCommand map[string]string, variablesToCheck map[string]configParam, missingParams *[]string) {
	for variableName := range variablesToCheck {
		if value, ok := variablesInCommand[variableName]; value != "" && ok {
			if variablesToCheck[variableName].Choices != nil {
				paramValid := checkIfParamInChoices(value, variableName, variablesToCheck)
				if paramValid {
					variablesFound[variableName] = value
				} else {
					*missingParams = append(*missingParams, variableName)
				}
			} else {
				variablesFound[variableName] = value
			}
		} else if variablesToCheck[variableName].Optional {
			variablesFound[variableName] = ""
		} else {
			*missingParams = append(*missingParams, variableName)
		}
	}
}

func checkMissingDependencies(variables map[string]string, commonParams map[string]configParam, operationParams map[string]configParam, missingDependencies *[]string) {
	for name, value := range variables {
		if value == "" {
			if _, isCommonParam := commonParams[name]; isCommonParam {
				for dependency, requiredValue := range commonParams[name].Dependencies {
					dependency = strings.ToUpper(dependency)
					if variables[dependency] == requiredValue {
						*missingDependencies = append(*missingDependencies, name)
						break
					}
				}
			} else if _, isOperationParam := operationParams[name]; isOperationParam {
				for dependency, requiredValue := range operationParams[name].Dependencies {
					if variables[dependency] == requiredValue {
						*missingDependencies = append(*missingDependencies, name)
						break
					}
				}
			}
		}
	}
}

func ReadOperationFromCommonParams(operation ProjectOperationParams, operationToFill string, config *Config) (string, map[string]string, error) {
	variables := make(map[string]string)

	var missingParams []string
	getMissingParams(variables, operation.Params, config.CommonParams, &missingParams)
	if len(missingParams) > 0 {
		return "", nil, fmt.Errorf(missingParamsError, strings.Join(missingParams, ", "))
	}

	var missingDependencies []string
	checkMissingDependencies(variables, config.CommonParams, nil, &missingDependencies)
	if len(missingDependencies) > 0 {
		return "", nil, fmt.Errorf(missingDependenciesError, strings.Join(missingDependencies, ", "))
	}

	filledOperation := operationToFill

	for name, value := range variables {
		if value == "" {
			r := regexp.MustCompile(fmt.Sprintf(`((--[^=]+=)*\${{%s}})`, name))

			filledOperation = r.ReplaceAllString(filledOperation, "")
		} else {
			filledOperation = strings.Replace(filledOperation, "${{"+name+"}}", value, -1)
		}
	}

	log.Println(filledOperation)

	return strings.TrimSpace(strings.TrimSuffix(filledOperation, "\n")), variables, nil
}

// ReadAdminOperation takes a operationToFill and a config and returns a ready to go operation
func ReadAdminOperation(operation OperationToFill, config *Config) (OperationToRun, error) {
	var configuredAdminOperation configAdminOperation

	for _, configCommand := range config.Operations {
		if configCommand.Name == operation.Name {
			configuredAdminOperation = configCommand
			break
		}
	}

	if configuredAdminOperation.Name == "" {
		return OperationToRun{}, fmt.Errorf(operationNotFoundError, operation.Name)
	}

	variables := make(map[string]string)
	var missingParams []string

	getMissingParams(variables, operation.Params, config.CommonParams, &missingParams)
	getMissingParams(variables, operation.Params, configuredAdminOperation.Params, &missingParams)
	if len(missingParams) > 0 {
		return OperationToRun{}, fmt.Errorf(missingParamsError, strings.Join(missingParams, ", "))
	}

	var missingDependencies []string
	checkMissingDependencies(variables, config.CommonParams, configuredAdminOperation.Params, &missingDependencies)
	if len(missingDependencies) > 0 {
		return OperationToRun{}, fmt.Errorf(missingDependenciesError, strings.Join(missingDependencies, ", "))
	}

	for name, value := range variables {
		if value == "" {
			log.Println(name)
			r := regexp.MustCompile(fmt.Sprintf(`((--[^=]+=)*\${{%s}})`, name))

			configuredAdminOperation.Operation = r.ReplaceAllString(configuredAdminOperation.Operation, "")
		} else {
			configuredAdminOperation.Operation = strings.Replace(configuredAdminOperation.Operation, "${{"+name+"}}", value, -1)
		}
	}

	var operationToRun OperationToRun
	operationToRun.Operation = strings.TrimSpace(strings.TrimSuffix(configuredAdminOperation.Operation, "\n"))
	operationToRun.Status = "ready"
	operationToRun.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(operationToRun.Operation)))
	operationToRun.RealtimeOutput = configuredAdminOperation.RealtimeOutput

	return operationToRun, nil
}

func captureParamsFromInstanceOperation(operationToFill InstanceOperationToFill, operation *string) []string {
	var missingParams []string
	r := regexp.MustCompile(captureParamRegex)
	// Get all variables in the operation
	matches := r.FindAllStringSubmatch(*operation, -1)
	for _, match := range matches {
		switch match[1] {
		case "NAME":
			*operation = strings.Replace(*operation, "${{"+match[1]+"}}", operationToFill.Instance.Name, -1)
		case "ZONE":
			*operation = strings.Replace(*operation, "${{"+match[1]+"}}", operationToFill.Instance.Zone, -1)
		case "NETWORKIP":
			*operation = strings.Replace(*operation, "${{"+match[1]+"}}", operationToFill.Instance.NetworkInterfaces[0].IP, -1)
		case "PROJECT":
			*operation = strings.Replace(*operation, "${{"+match[1]+"}}", operationToFill.Instance.ProjectName, -1)
		default:
			if value, inParams := operationToFill.Params[match[1]]; inParams {
				*operation = strings.Replace(*operation, "${{"+match[1]+"}}", value, -1)
			} else {
				missingParams = append(missingParams, match[1])
			}
		}
	}

	return missingParams
}

func ReadInstanceOperation(operation InstanceOperationToFill, config *Config) (OperationToRun, error) {
	var configuredAdminOperation configAdminOperation

	for _, configCommand := range config.InstanceOperations {
		if configCommand.Name == operation.Name {
			configuredAdminOperation = configCommand
			break
		}
	}

	if configuredAdminOperation.Name == "" {
		return OperationToRun{}, fmt.Errorf(operationNotFoundError, operation.Name)
	}

	missingParams := captureParamsFromInstanceOperation(operation, &configuredAdminOperation.Operation)

	if len(missingParams) > 0 {
		return OperationToRun{}, fmt.Errorf(missingInstanceParamsError, strings.Join(missingParams, ", "))
	}

	var operationToRun OperationToRun
	operationToRun.Operation = strings.TrimSpace(strings.TrimSuffix(configuredAdminOperation.Operation, "\n"))
	operationToRun.Status = "ready"
	operationToRun.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(operationToRun.Operation)))
	operationToRun.RealtimeOutput = configuredAdminOperation.RealtimeOutput
	return operationToRun, nil
}
