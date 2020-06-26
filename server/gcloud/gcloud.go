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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	pshell "github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/shell"

	"github.com/gorilla/websocket"
)

const (
	gcloudAuthError string = "there was a problem refreshing your current auth tokens"
	projectCmdError string = "failed to find project"
	// SdkAuthError is returned if there is an gcloud SDK auth error
	SdkAuthError string = "gCloud SDK auth invalid"
	// SdkProjectError is returned if the gcloud project given is invalid
	SdkProjectError                     string = "gCloud SDK project invalid"
	getComputeInstancesForProjectPrefix string = "gcloud compute instances list --format=json --project="
	iapRdpFirewallRuleCmd               string = "gcloud compute firewall-rules create admin-chrome-extension-private-rdp --direction=INGRESS   --action=allow   --rules=tcp:3389   --source-ranges=35.235.240.0/20 --source-tags=%s --project=%s"
	firewallRuleExistsOutput            string = "resource 'projects/%s/global/firewalls/admin-chrome-extension-private-rdp' already exists"
	firewallRuleAlreadyExists           string = "Firewall rule already exists"
	iapTunnelCmd                        string = "gcloud compute start-iap-tunnel %v 3389 --project=%v --local-host-port=localhost:%v"
)

type osFeatures struct {
	Type string `json:"type"`
}

type disk struct {
	OSFeatures []osFeatures `json:"guestOsFeatures"`
}

type networkInterfaces struct {
	Name    string `json:"name"`
	Network string `json:"network"`
	IP      string `json:"networkIP"`
}

// Instance is used as a structure for gcloud compute instances.
type Instance struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	Status            string              `json:"status"`
	Description       string              `json:"description"`
	Zone              string              `json:"zone"`
	Disk              []disk              `json:"disks"`
	NetworkInterfaces []networkInterfaces `json:"networkInterfaces"`
	ProjectName       string              `json:"project"`
}

type shell interface {
	ExecuteCmd(string) ([]byte, error)
	ExecuteCmdReader(string) ([]io.ReadCloser, error)
}

// GcloudExecutor is used to call gcloud functions with the shell passed in.
type GcloudExecutor struct {
	shell shell
}

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

type socketMessage struct {
	Message string `json:"messsage"`
	Err     string `json:"error"`
}

func newSocketMessage(message string, err error) *socketMessage {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	return &socketMessage{
		Message: message,
		Err:     errorMessage,
	}
}

func getComputeInstanceFromConn(ws *websocket.Conn) (*Instance, error) {
	for {
		_, message, err := ws.ReadMessage()
		fmt.Println("reading message")
		fmt.Println(string(message))
		if err != nil {
			log.Println("reading message error")
			return nil, err
		}
		var instance Instance
		if err := json.Unmarshal(message, &instance); err != nil {
			fmt.Println("error unmarshalling json")
			ws.Close()
			return nil, err
		}
		if instance.Name == "" || instance.ProjectName == "" {
			return nil, errors.New("missing values from data sent")
		}
		return &instance, nil
	}
}

func (gcloudExecutor *GcloudExecutor) createIapRdpFirewall(instance *Instance) (bool, string, error) {
	fmt.Println("Creating firewall")
	cmd := fmt.Sprintf(iapRdpFirewallRuleCmd, instance.Name, instance.ProjectName)
	instanceOutput, err := gcloudExecutor.shell.ExecuteCmd(cmd)

	if err != nil {
		if stringOutput := strings.ToLower(string(instanceOutput)); strings.Contains(stringOutput, gcloudAuthError) {
			return false, stringOutput, errors.New(SdkAuthError)
		} else if strings.Contains(stringOutput, projectCmdError) {
			return false, stringOutput, errors.New(SdkProjectError)
		} else if strings.Contains(stringOutput, fmt.Sprintf(firewallRuleExistsOutput, instance.ProjectName)) {
			fmt.Println("already exists")
			return true, stringOutput, errors.New(firewallRuleAlreadyExists)
		} else {
			return false, stringOutput, err
		}
	}
	return true, string(instanceOutput), nil
}

func (gcloudExecutor *GcloudExecutor) startIapTunnel(instance *Instance, port int) {
	fmt.Println("Starting IAP tunnel")
	cmd := fmt.Sprintf(iapTunnelCmd, instance.Name, instance.ProjectName, port)
	fmt.Println(cmd)
	output, err := gcloudExecutor.shell.ExecuteCmdReader(cmd)
	if err != nil {
		log.Println(err)
	}
	//stderr: ERROR: (gcloud) The project property must be set to a valid project ID, not the project name [Dsadas]
	//stderr: ERROR: (gcloud.compute.start-iap-tunnel) Could not fetch resource:
	//stderr: ERROR: (gcloud.compute.start-iap-tunnel) Could not fetch resource:
	// stderr:  - The resource 'projects/rishabl-test/zones/us-west1-b/instances/invalidname' was not found
	//stderr: ERROR: (gcloud.compute.start-iap-tunnel) Local port [23966] is not available.
	stdout, stderr := output[0], output[1]

	stdoutScanner := bufio.NewScanner(stdout)
	go func() {
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			fmt.Println(fmt.Sprintf("stdout: %v", line))
		}
	}()

	stderrScanner := bufio.NewScanner(stderr)
	go func() {
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			fmt.Println(fmt.Sprintf("stderr: %v", line))
		}
	}()

}

func StartPrivateRdp(ws *websocket.Conn) {
	instanceToConn, err := getComputeInstanceFromConn(ws)
	if err != nil {
		log.Println(err)
		if err := ws.WriteJSON(newSocketMessage("", err)); err != nil {
			log.Println(err)
		}
		ws.Close()
		return
	}
	if err := ws.WriteJSON(newSocketMessage(fmt.Sprintf("Server received instance %s", instanceToConn.Name), err)); err != nil {
		ws.Close()
		log.Println(err)
		return
	}
	fmt.Println("got instances")
	shell := &pshell.CmdShell{}
	g := NewGcloudExecutor(shell)

	// firewallCreated, output, err := g.createIapRdpFirewall(instanceToConn)
	// if err != nil {
	// 	log.Println(err)
	// 	if err := ws.WriteJSON(newSocketMessage(output, err)); err != nil {
	// 		ws.Close()
	// 		log.Println(err)
	// 		return
	// 	}
	// }
	// fmt.Println(firewallCreated)
	// if firewallCreated {
	// 	if err := ws.WriteJSON(newSocketMessage("IAP tunnel firewall was created", nil)); err != nil {
	// 		log.Println(err)
	// 		ws.Close()
	// 		return
	// 	}
	// } else {
	// 	if err := ws.WriteJSON(newSocketMessage("IAP tunnel firewall was not created", nil)); err != nil {
	// 		log.Println(err)
	// 		ws.Close()
	// 		return
	// 	}
	// }

	freePort, err := pshell.GetPort()
	if err != nil {
		if err := ws.WriteJSON(newSocketMessage("", errors.New("Could not get a unused port on system"))); err != nil {
			log.Println(err)
			ws.Close()
			return
		}
	}
	fmt.Println(freePort)
	g.startIapTunnel(instanceToConn, freePort)

}
