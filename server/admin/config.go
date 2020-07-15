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
	configFileReadError           string = "Error reading configuration file, please make sure it has the necessary permissions and is in DIRECTORY"
	configFileDataError           string = "Error reading data from configuration file, please follow the format specified"
	configCommandMissingVariables string = "Config is missing variables for these command(s): %s"
	commandNotFoundError          string = "%s command was not found in the config"
	missingVariablesError         string = "Missing variables defined in config file: %s"
)

// configVariable points to a variable in the config file
type configVariable struct {
	Default  string `json:"default"`
	Type     string `json:"type"`
	Optional bool   `json:"optional"`
}

// configAdminCOmmand points to a configured admin command
type configAdminCommand struct {
	Name      string                    `json:"name"`
	Command   string                    `json:"command"`
	Variables map[string]configVariable `json:"variables"`
}

// Config is the fully loaded config represented as structures
type Config struct {
	Commands        []configAdminCommand      `json:"commands"`
	CommonVariables map[string]configVariable `json:"common_variables"`
}

// CommandToFill is sent by the extension detailing a command and the variables to be filled
type CommandToFill struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
}

// CommandToRun contains a ready to run command with its status and a unique
type CommandToRun struct {
	Command string `json:"command"`
	Hash    string `json:"hash"`
	Status  string `json:"status"`
}

func checkConfigForMissingVariables(config Config) map[string][]string {
	missingVariables := make(map[string][]string)

	r := regexp.MustCompile(`\$(?s)([A-Z]+_*[A-Z]+)`)
	for _, command := range config.Commands {

		// Get all variables in the command
		matches := r.FindAllStringSubmatch(command.Command, -1)
		for _, match := range matches {

			// Check if variable is defined in either common variables or the command's variables
			if _, inCommonVariables := config.CommonVariables[match[1]]; !inCommonVariables {
				if _, inCommandVariables := command.Variables[match[1]]; !inCommandVariables {
					missingVariables[command.Name] = append(missingVariables[command.Name], match[1])
				}
			}
		}
	}

	return missingVariables
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
			if key == strings.ToLower(key) {
				config.CommonVariables[strings.ToUpper(key)] = value
				delete(config.CommonVariables, key)
			}
		}
	}

	missingVariables := checkConfigForMissingVariables(config)

	if len(missingVariables) > 0 {
		var errorStrings []string
		// Join all the missing variables in a list
		for key, val := range missingVariables {
			errString := fmt.Sprintf("%s: %s", key, strings.Join(val, ", "))
			errorStrings = append(errorStrings, errString)
		}
		return &Config{}, fmt.Errorf(configCommandMissingVariables, strings.Join(errorStrings, ". "))
	}

	return &config, nil
}

// getMissingVariables checks variables to the current ones in the command either adding them to missingVariables or variablesFound
func getMissingVariables(variablesFound map[string]string, variablesInCommand map[string]string, variablesToCheck map[string]configVariable, missingVariables *[]string) {
	for variableName := range variablesToCheck {
		if value, ok := variablesInCommand[variableName]; ok {
			variablesFound[variableName] = value
		} else if variablesToCheck[variableName].Optional {
			variablesFound[variableName] = ""
		} else {
			*missingVariables = append(*missingVariables, variableName)
		}
	}
}

// ReadAdminCommand takes a commandToFill and a config and returns a ready to go command
func ReadAdminCommand(command CommandToFill, config *Config) (CommandToRun, error) {
	var configuredAdminCommand configAdminCommand

	for _, configCommand := range config.Commands {
		if configCommand.Name == command.Name {
			configuredAdminCommand = configCommand
			break
		}
	}

	if configuredAdminCommand.Name == "" {
		return CommandToRun{}, fmt.Errorf(commandNotFoundError, command.Name)
	}

	variables := make(map[string]string)
	var missingVariables []string

	getMissingVariables(variables, command.Variables, config.CommonVariables, &missingVariables)
	getMissingVariables(variables, command.Variables, configuredAdminCommand.Variables, &missingVariables)

	if len(missingVariables) > 0 {
		return CommandToRun{}, fmt.Errorf(missingVariablesError, strings.Join(missingVariables, ", "))
	}

	for name, value := range variables {
		if value == "" {
			r := regexp.MustCompile(fmt.Sprintf(`(--[^=]+=\$%s)`, name))

			configuredAdminCommand.Command = r.ReplaceAllString(configuredAdminCommand.Command, "")
		} else {
			configuredAdminCommand.Command = strings.Replace(configuredAdminCommand.Command, "$"+name, value, -1)
		}
	}

	var commandToRun CommandToRun
	commandToRun.Command = strings.TrimSpace(strings.TrimSuffix(configuredAdminCommand.Command, "\n"))
	commandToRun.Status = "ready"
	commandToRun.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(commandToRun.Command)))

	return commandToRun, nil

}
