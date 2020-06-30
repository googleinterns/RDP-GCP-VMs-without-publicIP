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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/gorilla/websocket"
)

// NewGcloudExecutor creates a new gcloudExecutor struct with a struct that implements shell.
func NewGcloudExecutor(shell shell) *GcloudExecutor {
	return &GcloudExecutor{
		shell: shell,
	}
}

// GetComputeInstances runs the gCloud instances command, parses the output to the Instances struct and returns
func (gcloudExecutor *GcloudExecutor) GetComputeInstances(projectName string) ([]Instance, error) {
	instanceOutput, err := gcloudExecutor.shell.ExecuteCmd(getComputeInstancesForProjectPrefix + projectName)
	if err != nil {
		if stringOutput := strings.ToLower(string(instanceOutput)); strings.Contains(stringOutput, gcloudAuthError) {
			return nil, errors.New(SdkAuthError)
		} else if strings.Contains(stringOutput, projectCmdError) {
			return nil, errors.New(SdkProjectError)
		}
		return nil, err
	}

	var instances []Instance
	if err := json.Unmarshal(instanceOutput, &instances); err != nil {
		return nil, err
	}

	return instances, nil
}

func (gcloudExecutor *GcloudExecutor) createIapFirewall(ws *websocket.Conn, instance *Instance) error {
	log.Println("Creating firewall for ", instance.Name)

	cmd := fmt.Sprintf(iapFirewallCreateCmd, instance.Name, instance.Name, instance.ProjectName)

	instanceOutput, err := gcloudExecutor.shell.ExecuteCmd(cmd)

	var firewallCreated bool
	var output string
	var returnErr error

	if err != nil {
		if stringOutput := strings.ToLower(string(instanceOutput)); strings.Contains(stringOutput, gcloudAuthError) {
			firewallCreated = false
			output = stringOutput
			returnErr = errors.New(SdkAuthError)
		} else if strings.Contains(stringOutput, projectCmdError) {
			firewallCreated = false
			output = stringOutput
			returnErr = errors.New(SdkProjectError)
		} else if strings.Contains(stringOutput, fmt.Sprintf(firewallRuleExistsOutput, instance.ProjectName, instance.Name)) {
			firewallCreated = true
			output = stringOutput
		} else {
			firewallCreated = false
			output = stringOutput
			returnErr = errors.New(SdkProjectError)
		}
	} else {
		firewallCreated = true
		output = string(instanceOutput)
	}

	if !firewallCreated {
		writeToSocket(ws, output, returnErr)
	} else {
		if err := writeToSocket(ws, output, nil); err != nil {
			returnErr = err
		}
	}

	return nil
}

func (gcloudExecutor *GcloudExecutor) deleteIapFirewall(instance *Instance) {
	log.Println("Deleting firewall for ", instance.Name)
	cmd := fmt.Sprintf(iapFirewallDeleteCmd, instance.Name, instance.ProjectName)
	gcloudExecutor.shell.ExecuteCmd(cmd)
}

func readIapTunnelOutput(scanner *bufio.Scanner, tunnelCreated *bool, cmdOutput *[]string, iapOutputChan chan<- iapResult) {
	for scanner.Scan() {
		line := scanner.Text()
		*cmdOutput = append(*cmdOutput, line)
		if strings.Contains(line, tunnelCreatedOutput) {
			*tunnelCreated = true
			iapOutputChan <- iapResult{*tunnelCreated, *cmdOutput, nil}
		} else if strings.Contains(line, gcloudErrorOutput) {
			iapOutputChan <- iapResult{*tunnelCreated, *cmdOutput, nil}
		}
	}
}

func (gcloudExecutor *GcloudExecutor) startIapTunnel(ctx context.Context, ws *websocket.Conn, instance *Instance, port int, outputChan chan<- iapResult) {
	log.Println("Starting IAP tunnel for ", instance.Name)

	cmd := fmt.Sprintf(iapTunnelCmd, instance.Name, instance.ProjectName, port)
	output, cmdCancel, err := gcloudExecutor.shell.ExecuteCmdReader(cmd)
	if err != nil {
		log.Println(err)
	}

	stdoutScanner, stderrScanner := bufio.NewScanner(output[0]), bufio.NewScanner(output[1])

	var cmdOutput []string
	tunnelCreated := false
	iapOutputChan := make(chan iapResult)

	go readIapTunnelOutput(stdoutScanner, &tunnelCreated, &cmdOutput, iapOutputChan)
	go readIapTunnelOutput(stderrScanner, &tunnelCreated, &cmdOutput, iapOutputChan)

	iapResult := <-iapOutputChan
	if !iapResult.tunnelCreated {
		if err := writeToSocket(ws, strings.Join(iapResult.cmdOutput, "\n"), errors.New("Could not start IAP tunnel")); err != nil {
			iapResult.err = err
		}
	} else {
		if err := writeToSocket(ws, fmt.Sprintf("Started IAP tunnel for %v", instance.Name), nil); err != nil {
			iapResult.err = err
		}
	}

	outputChan <- iapResult

	<-ctx.Done()
	cmdCancel()
}

func (gcloudExecutor *GcloudExecutor) startRdpProgram(ws *websocket.Conn, creds *credentials, port int, quit chan<- bool) {
	log.Println("Starting xfreerdp for ", creds.Username)
	cmd := fmt.Sprintf(rdpProgramCmd, port, creds.Username, creds.Password)
	instanceOutput, err := gcloudExecutor.shell.ExecuteCmd(cmd)

	if err != nil {
		writeToSocket(ws, fmt.Sprintf("Unable to start RDP program for %v", creds.Username), nil)
		return
	}
	if instanceOutput != nil {
		writeToSocket(ws, fmt.Sprintf("Unable to start RDP program for %v", creds.Username), nil)
		quit <- true
		return
	}

}
