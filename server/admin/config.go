package admin

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

const (
	configFileReadError string = "Error reading configuration file, please make sure it has the necessary permissions and is in DIRECTORY"
	configFileDataError string = "Error reading data from configuration file, please follow the format specified"
)

type ConfigVariable struct {
	Default string `json:"default"`
	Type    string `json:"type"`
}

type ConfigAdminCommand struct {
	Name      string                    `json:"name"`
	Command   string                    `json:"command"`
	Variables map[string]ConfigVariable `json:"variables"`
}

type Config struct {
	Commands        []ConfigAdminCommand      `json:"commands"`
	CommonVariables map[string]ConfigVariable `json:"common_variables"`
}

type CommandToFill struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
}

type CommandToRun struct {
	Command string `json:"command"`
	Hash    string `json:"hash"`
	Status  string `json:"status"`
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

	if config.CommonVariables != nil {
		for key, value := range config.CommonVariables {
			config.CommonVariables[strings.ToUpper(key)] = value
			delete(config.CommonVariables, key)
		}
	}

	return &config, nil
}

func ReadAdminCommand(command CommandToFill, config *Config) (CommandToRun, error) {
	var configuredAdminCommand ConfigAdminCommand

	for _, configCommand := range config.Commands {
		if configCommand.Name == command.Name {
			configuredAdminCommand = configCommand
			break
		}
	}

	if configuredAdminCommand.Name == "" {
		return CommandToRun{}, errors.New("command not found")
	}

	variables := make(map[string]string)
	var missingVariables []string
	for variableName, _ := range config.CommonVariables {
		if value, ok := command.Variables[variableName]; ok {
			variables[variableName] = value
		} else {
			missingVariables = append(missingVariables, variableName)
		}
	}

	log.Println(configuredAdminCommand.Variables)

	for variableName, _ := range configuredAdminCommand.Variables {
		if value, ok := command.Variables[variableName]; ok {
			variables[variableName] = value
		} else {
			missingVariables = append(missingVariables, variableName)
		}
	}
	log.Println(len(missingVariables))

	if len(missingVariables) > 0 {
		log.Println(missingVariables)
		return CommandToRun{}, fmt.Errorf("Missing variables: %v", strings.Join(missingVariables, ", "))
	}

	for name, value := range variables {
		configuredAdminCommand.Command = strings.Replace(configuredAdminCommand.Command, "$"+name, value, -1)
	}

	var commandToRun CommandToRun
	commandToRun.Command = configuredAdminCommand.Command
	commandToRun.Status = "ready"
	commandToRun.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(commandToRun.Command)))

	return commandToRun, nil

}
