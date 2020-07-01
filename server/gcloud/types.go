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
	"context"
	"io"
)

// consts containing possible errors from running gcloud commands
const (
	gcloudAuthError string = "there was a problem refreshing your current auth tokens"
	projectCmdError string = "failed to find project"
	// SdkAuthError is returned if there is an gcloud SDK auth error
	SdkAuthError string = "gCloud SDK auth invalid"
	// SdkProjectError is returned if the gcloud project given is invalid
	SdkProjectError   string = "gCloud SDK project invalid"
	gcloudErrorOutput string = "Error:"
)

// iap firewall consts
const (
	iapFirewallCreateCmd      string = "gcloud compute firewall-rules create admin-extension-private-rdp-%v --direction=INGRESS   --action=allow   --rules=tcp:3389   --source-ranges=35.235.240.0/20 --source-tags=%s --project=%s"
	iapFirewallDeleteCmd      string = "gcloud compute firewall-rules delete admin-extension-private-rdp-%v -q --project=%s"
	firewallRuleExistsOutput  string = "resource 'projects/%s/global/firewalls/admin-extension-private-rdp-%v' already exists"
	firewallRuleAlreadyExists string = "Firewall rule already exists for %v"
	didntCreateFirewall       string = "Didn't create firewall for %v"
	createdFirewall           string = "Created firewall for %v"
)

// iap tunnel and websocket consts
const (
	getComputeInstancesForProjectPrefix string = "gcloud compute instances list --format=json --project="
	missingInstanceValues               string = "Missing value from instance data sent"
	iapTunnelCmd                        string = "gcloud compute start-iap-tunnel %v 3389 --project=%v --local-host-port=localhost:%v --verbosity=debug"
	tunnelCreatedOutput                 string = "DEBUG: CLOSE"
	iapTunnelError                      string = "Could not start IAP tunnel for %v"
	iapTunnelStarted                    string = "Started IAP tunnel for %v on port: %v"
)

// automated rdp program consts
const (
	rdpProgramCmd   string = "xfreerdp /v:localhost /port:%v /u:%v /p:%v +sec-rdp /cert-ignore"
	rdpProgramError string = "Unable to start RDP program for %v"
	rdpProgramQuit  string = "Quit RDP program for %v"
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
	ExecuteCmdReader(string) ([]io.ReadCloser, context.CancelFunc, error)
}

// GcloudExecutor is used to call gcloud functions with the shell passed in.
type GcloudExecutor struct {
	shell shell
}

// socketMessage is the struct that is sent to the websockets
type socketMessage struct {
	Message string `json:"messsage"`
	Err     string `json:"error"`
}

// credentials struct is used for the automated rdp program
type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// iapResult struct is returned from starting the IAP tunnel
type iapResult struct {
	tunnelCreated bool
	cmdOutput     []string
	err           error
}

// conn interface is used to mock websocket connections
type conn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteJSON(v interface{}) error
	Close() error
}

// socketCmd struct is used to read commands such as start-rdp and login from the websocket
type socketCmd struct {
	Cmd          string `json:"cmd"`
	InstanceName string `json:"name"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}
