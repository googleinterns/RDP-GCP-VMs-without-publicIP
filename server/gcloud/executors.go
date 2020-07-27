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

// Package gcloud runs gCloud SDK commands like getting instances, starting IAP tunnels and firewalls
package gcloud

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
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

// createIapFirewall creates the firewall rule for the instance to allow starting IAP tunnels
func (gcloudExecutor *GcloudExecutor) createIapFirewall(ws conn, instance *Instance) error {
	log.Println("Creating firewall for ", instance.Name)

	cmd := fmt.Sprintf(iapFirewallCreateCmd, instance.Name, instance.Name, instance.ProjectName, instance.FirewallNetwork)

	instanceOutput, err := gcloudExecutor.shell.ExecuteCmd(cmd)

	var (
		output    string
		returnErr error
	)

	if err != nil {
		if stringOutput := strings.ToLower(string(instanceOutput)); strings.Contains(stringOutput, gcloudAuthError) {
			output = fmt.Sprintf(didntCreateFirewallOutput, instance.Name)
			returnErr = errors.New(SdkAuthError)
		} else if strings.Contains(stringOutput, projectCmdError) {
			output = fmt.Sprintf(didntCreateFirewallOutput, instance.Name)
			returnErr = errors.New(SdkProjectError)
		} else if strings.Contains(stringOutput, fmt.Sprintf(firewallRuleExistsCmdOutput, instance.ProjectName, instance.Name)) {
			output = fmt.Sprintf(firewallRuleAlreadyExistsOutput, instance.Name)
			returnErr = nil
		} else {
			output = fmt.Sprintf(didntCreateFirewallOutput, instance.Name)
			returnErr = err
		}
	} else {
		output = fmt.Sprintf(createdFirewallOutput, instance.Name)
		returnErr = nil
	}
	log.Println(output)

	if err := writeToSocket(ws, string(instanceOutput), returnErr); err != nil {
		return err
	}

	if err := writeToSocket(ws, output, returnErr); err != nil {
		returnErr = err
	}

	return returnErr
}

// deleteIapFirewall creates the firewall rule created for that instance
func (gcloudExecutor *GcloudExecutor) deleteIapFirewall(instance *Instance) {
	log.Println("Deleting firewall for ", instance.Name)
	cmd := fmt.Sprintf(iapFirewallDeleteCmd, instance.Name, instance.ProjectName)
	gcloudExecutor.shell.ExecuteCmd(cmd)
}

// readIapTunnelOutput is used as an helper to parse the output from starting the IAP tunnel
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

// startIapTunnel is used run the start iap tunnel command and return the appropriate output
func (gcloudExecutor *GcloudExecutor) startIapTunnel(ctx context.Context, ws conn, instance *Instance, portListener *net.TCPListener, outputChan chan<- iapResult) {
	log.Println("Starting IAP tunnel for ", instance.Name)
	port := portListener.Addr().(*net.TCPAddr).Port
	cmd := fmt.Sprintf(iapTunnelCmd, instance.Name, instance.ProjectName, port, instance.Zone)
	portListener.Close()
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
		if err := writeToSocket(ws, strings.Join(iapResult.cmdOutput, "\n"), fmt.Errorf(iapTunnelError, instance.Name)); err != nil {
			iapResult.err = err
		}
	} else {
		if err := writeToSocket(ws, fmt.Sprintf(iapTunnelStarted, instance.Name, port), nil); err != nil {
			iapResult.err = err
		}
	}

	outputChan <- iapResult

	<-ctx.Done()
	log.Println("calling cancel func")
	cmdCancel()
}

// startRdpProgram starts the RDP program if the command is sent to the server
func (gcloudExecutor *GcloudExecutor) startRdpProgram(ws conn, creds *credentials, port int, quit chan<- bool) {
	log.Println("Starting xfreerdp for ", creds.Username)
	cmd := fmt.Sprintf(rdpProgramCmd, port, creds.Username, creds.Password)
	instanceOutput, err := gcloudExecutor.shell.ExecuteCmd(cmd)

	if err != nil {
		writeToSocket(ws, "", fmt.Errorf(rdpProgramError, creds.Username))
		return
	}
	if instanceOutput != nil {
		writeToSocket(ws, fmt.Sprintf(rdpProgramQuit, creds.Username), nil)
		quit <- true
		return
	}

}
