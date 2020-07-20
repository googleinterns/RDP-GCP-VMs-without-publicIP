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

	"github.com/spf13/viper"
)

const (
	configFileReadError          string = "Error reading configuration file, please make sure it has the necessary permissions and is in DIRECTORY"
	configFileDataError          string = "Error reading data from configuration file, please follow the format specified"
	configOperationMissingParams string = "Config is missing variables for these operation(s): %s"
	operationNotFoundError       string = "%s operation was not found in the config"
	missingParamsError           string = "Missing parameters defined in config file for this operation: %s"
)

// configParam points to a variable in the config file
type configParam struct {
	Default     string `json:"default"`
	Type        string `json:"type"`
	Optional    bool   `json:"optional"`
	Description string `json:"description"`
	Sample      string `json:"sample"`
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
	Operations   []configAdminOperation `json:"operations"`
	CommonParams map[string]configParam `json:"common_params"`
	EnableRdp    bool                   `json:"enable_rdp"`
}

// OperationToFill is sent by the extension detailing a operation and the variables to be filled
type OperationToFill struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"variables"`
}

// OperationToRun contains a ready to run operation with its status and a unique
type OperationToRun struct {
	Operation string `json:"operation"`
	Hash      string `json:"hash"`
	Status    string `json:"status"`
}

func checkConfigForMissingParams(config Config) map[string][]string {
	missingParams := make(map[string][]string)

	r := regexp.MustCompile(`\$(?s)([A-Z]+_*[A-Z]+)`)
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
		return &Config{}, errors.New(configFileReadError)
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

// getMissingParams checks variables to the current ones in the operation either adding them to missingParams or variablesFound
func getMissingParams(variablesFound map[string]string, variablesInCommand map[string]string, variablesToCheck map[string]configParam, missingParams *[]string) {
	for variableName := range variablesToCheck {
		log.Println(variableName)
		log.Println(variablesInCommand)
		if value, ok := variablesInCommand[variableName]; value != "" && ok {
			variablesFound[variableName] = value
		} else if variablesToCheck[variableName].Optional {
			log.Println(variableName)
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
			r := regexp.MustCompile(fmt.Sprintf(`(--[^=]+=\$%s)`, name))

			configuredAdminOperation.Operation = r.ReplaceAllString(configuredAdminOperation.Operation, "")
		} else {
			configuredAdminOperation.Operation = strings.Replace(configuredAdminOperation.Operation, "$"+name, value, -1)
		}
	}

	var operationToRun OperationToRun
	operationToRun.Operation = strings.TrimSpace(strings.TrimSuffix(configuredAdminOperation.Operation, "\n"))
	operationToRun.Status = "ready"
	operationToRun.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(operationToRun.Operation)))

	return operationToRun, nil

}
