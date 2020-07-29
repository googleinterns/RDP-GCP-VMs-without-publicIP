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
	configFileReadError          string = "Error reading configuration file, please make sure it has the necessary permissions and is in DIRECTORY"
	configFileDataError          string = "Error reading data from configuration file, please follow the format specified"
	configOperationMissingParams string = "Config is missing variables for these operation(s): %s"
	operationNotFoundError       string = "%s operation was not found in the config"
	missingParamsError           string = "Missing parameters defined in config file for this operation: %s"
	missingInstanceParamsError   string = "Missing parameters in the instance needed for this operation: %s, use Admin Operations instead for this operation"
	captureParamRegex            string = `\${{(?s)([A-Z]+_*[A-Z]+)}}(?s)`
)

// configParam points to a variable in the config file
type configParam struct {
	Default     string   `json:"default"`
	Optional    bool     `json:"optional"`
	Description string   `json:"description"`
	Sample      string   `json:"sample"`
	Choices     []string `json:"choices"`
}

// configAdminOperation points to a configured admin operation
type configAdminOperation struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Operation   string                 `json:"operation"`
	Params      map[string]configParam `json:"params"`
}

// Config is the fully loaded config represented as structures
type Config struct {
	InstanceOperations []configAdminOperation `json:"instance_operations"`
	Operations         []configAdminOperation `json:"operations"`
	CommonParams       map[string]configParam `json:"common_params"`
	EnableRdp          bool                   `json:"enable_rdp"`
}

// OperationToFill is sent by the extension detailing a operation and the variables to be filled
type OperationToFill struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"variables"`
}

type InstanceOperationToFill struct {
	Name     string          `json:"name"`
	Instance gcloud.Instance `json:"instance"`
}

// OperationToRun contains a ready to run operation with its status and a unique
type OperationToRun struct {
	Operation string `json:"operation"`
	Hash      string `json:"hash"`
	Status    string `json:"status"`
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

	return missingParams
}

// LoadConfig reads the config file and unmarshals the data to structs
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
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

	for name, value := range variables {
		if value == "" {
			r := regexp.MustCompile(fmt.Sprintf(`(--[^=]+=\${{%s}})`, name))

			configuredAdminOperation.Operation = r.ReplaceAllString(configuredAdminOperation.Operation, "")
		} else {
			configuredAdminOperation.Operation = strings.Replace(configuredAdminOperation.Operation, "${{"+name+"}}", value, -1)
		}
	}

	var operationToRun OperationToRun
	operationToRun.Operation = strings.TrimSpace(strings.TrimSuffix(configuredAdminOperation.Operation, "\n"))
	operationToRun.Status = "ready"
	operationToRun.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(operationToRun.Operation)))

	return operationToRun, nil
}

func captureParamsFromInstanceOperation(instance gcloud.Instance, operation *string) []string {
	var missingParams []string
	r := regexp.MustCompile(captureParamRegex)
	// Get all variables in the operation
	matches := r.FindAllStringSubmatch(*operation, -1)
	for _, match := range matches {
		switch match[1] {
		case "NAME":
			*operation = strings.Replace(*operation, "${{"+match[1]+"}}", instance.Name, -1)
		case "ZONE":
			*operation = strings.Replace(*operation, "${{"+match[1]+"}}", instance.Zone, -1)
		case "NETWORKIP":
			*operation = strings.Replace(*operation, "${{"+match[1]+"}}", instance.NetworkInterfaces[0].IP, -1)
		case "PROJECT":
			*operation = strings.Replace(*operation, "${{"+match[1]+"}}", instance.ProjectName, -1)
		default:
			missingParams = append(missingParams, match[1])
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

	missingParams := captureParamsFromInstanceOperation(operation.Instance, &configuredAdminOperation.Operation)

	if len(missingParams) > 0 {
		return OperationToRun{}, fmt.Errorf(missingInstanceParamsError, strings.Join(missingParams, ", "))
	}

	var operationToRun OperationToRun
	operationToRun.Operation = strings.TrimSpace(strings.TrimSuffix(configuredAdminOperation.Operation, "\n"))
	operationToRun.Status = "ready"
	operationToRun.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(operationToRun.Operation)))

	return operationToRun, nil
}
