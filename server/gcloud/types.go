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
	"context"
	"io"
	"time"
)

// consts containing possible errors from running gcloud commands
const (
	gcloudAuthError string = "there was a problem refreshing your current auth tokens"
	projectCmdError string = "failed to find project"
	// SdkAuthError is returned if there is an gcloud SDK auth error
	SdkAuthError string = `gCloud SDK authorization is invalid, missing or expired, please try relogging using "gcloud auth login" and try again.`
	// SdkProjectError is returned if the gcloud project given is invalid
	SdkProjectError   string = "gCloud SDK project is invalid for the current credentials"
	gcloudErrorOutput string = "ERROR:"
)

// iap firewall consts
const (
	iapFirewallCreateCmd            string = "gcloud compute firewall-rules create admin-extension-private-rdp-%v --direction=INGRESS   --action=allow   --rules=tcp:3389   --source-ranges=35.235.240.0/20 --source-tags=%s --project=%s --network=%s"
	firewallDeleteCmd               string = "gcloud compute firewall-rules delete admin-extension-private-rdp-%v -q --project=%s"
	firewallRuleExistsCmdOutput     string = "resource 'projects/%s/global/firewalls/admin-extension-private-rdp-%v' already exists"
	firewallRuleAlreadyExistsOutput string = "Firewall rule already exists for %v"
	didntCreateFirewallOutput       string = "Could not create firewall for %v"
	createdFirewallOutput           string = "Created firewall for %v, will delete in %v"
	multipleNetworksError           string = "%v has 0 or more than 1 network interface"
	deleteFirewallAuthError         string = "Couldn't delete IAP firewall rule: admin-extension-private-rdp-%v due to auth error, please delete it manually"
	deleteFirewallProjectError      string = "Couldn't delete IAP firewall rule: admin-extension-private-rdp-%v due to project error, please delete it manually"
)

// iap tunnel and websocket consts
const (
	getComputeInstancesForProjectPrefix string = "gcloud compute instances list --format=json --project="
	missingInstanceValues               string = "Missing value from instance data sent"
	iapTunnelCmd                        string = "gcloud compute start-iap-tunnel %v 3389 --project=%v --local-host-port=localhost:%v --zone=%s --verbosity=debug"
	tunnelCreatedOutput                 string = "DEBUG: CLOSE"
	iapTunnelError                      string = "Could not start IAP tunnel for %v"
	iapTunnelStarted                    string = "Started IAP tunnel for %v on port: %v. Will close in %v"
	receivedEndCmd                      string = "Received end RDP command from connection"
	receivedStartRdpCmd                 string = "Received command to start RDP program with credentials"
	endingIapTunnel                     string = "Ending IAP tunnel for %v"
	createIapFailed                     string = "Creating IAP tunnel failed"
	// IMPORTANT: IF CHANGED, NEEDS TO BE CHANGED IN EXTENSION AS WELL
	readyForCommandOutput string = "Ready for command"
	shutDownRdp           string = "Shutdown private RDP for %v"
	deletingFirewall      string = "Deleting firewall for %v"
)

// automated rdp program consts
const (
	rdpProgramCmd   string = "xfreerdp /v:localhost /port:%v /u:%v /p:%v +sec-rdp /cert-ignore"
	rdpProgramError string = "Unable to start RDP program for %v"
	rdpProgramQuit  string = "Quit RDP program for %v"
)

const (
	endRdpSocketCmd   string = "end"
	startRdpSocketCmd string = "start-rdp"
)

const (
	rdpContextTimeout      time.Duration = 2 * time.Hour
	firewallContextTimeout time.Duration = 2 * time.Minute
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
	FirewallNetwork   string              `json:"firewallNetwork"`
	PreRDPParams      map[string]string   `json:"params"`
}

type shell interface {
	ExecuteCmd(string) ([]byte, error)
	ExecuteCmdWithContext(context.Context, string) ([]byte, error)
	ExecuteCmdReader(string) ([]io.ReadCloser, context.CancelFunc, error)
}

// GcloudExecutor is used to call gcloud functions with the shell passed in.
type GcloudExecutor struct {
	shell shell
}

// socketMessage is the struct that is sent to the websockets
type socketMessage struct {
	Message string `json:"message"`
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
