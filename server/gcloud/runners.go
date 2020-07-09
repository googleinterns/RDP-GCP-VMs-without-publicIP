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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"

	pshell "github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/shell"
	"github.com/gorilla/websocket"
)

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

// writeToSocket is a wrapper that is used to write JSON to the websocket
func writeToSocket(ws conn, message string, err error) error {
	if err := ws.WriteJSON(newSocketMessage(message, err)); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// StartPrivateRdp is a task runner that runs all the individual functions for automated RDP.
func (gcloudExecutor *GcloudExecutor) StartPrivateRdp(ws *websocket.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), rdpContextTimeout)
	iapOutputChan := make(chan iapResult)
	endRdpChan := make(chan bool)

	instanceToConn, err := getComputeInstanceFromConn(ws)
	if err != nil {
		gcloudExecutor.cleanUpRdp(ws, instanceToConn, false, false, cancel)
		return
	}
	if err := writeToSocket(ws, fmt.Sprintf("Server received instance %s", instanceToConn.Name), err); err != nil {
		gcloudExecutor.cleanUpRdp(ws, instanceToConn, false, false, cancel)
		return
	}

	log.Println("Got instance", instanceToConn.Name)

	if err := gcloudExecutor.createIapFirewall(ws, instanceToConn); err != nil {
		gcloudExecutor.cleanUpRdp(ws, instanceToConn, false, false, cancel)
		return
	}

	portListener, err := pshell.FindOpenPort()
	if err != nil {
		writeToSocket(ws, "", errors.New("Could not get a unused port on system"))
		gcloudExecutor.cleanUpRdp(ws, instanceToConn, true, false, cancel)
		return
	}

	freePort := portListener.Addr().(*net.TCPAddr).Port

	log.Println("Got free port ", freePort)

	go gcloudExecutor.startIapTunnel(ctx, ws, instanceToConn, portListener, iapOutputChan)
	if output := <-iapOutputChan; !output.tunnelCreated || output.err != nil {
		writeToSocket(ws, "", errors.New(createIapFailed))
		gcloudExecutor.cleanUpRdp(ws, instanceToConn, true, false, cancel)
		return
	} else {
		writeToSocket(ws, readyForCommandOutput, nil)
	}

	go gcloudExecutor.listenForCmd(ws, instanceToConn, freePort, endRdpChan)

	if endRdp := <-endRdpChan; endRdp {
		gcloudExecutor.cleanUpRdp(ws, instanceToConn, true, true, cancel)
		return
	}

	cancel()
	return
}

// getComputeInstancesFromConn reads the instance that is sent at the start of the websocket connection
func getComputeInstanceFromConn(ws conn) (*Instance, error) {
	for {
		_, message, err := ws.ReadMessage()

		log.Println("got message: ", string(message))

		if err != nil {
			writeToSocket(ws, "", err)
			return nil, err
		}
		var instance Instance
		if err := json.Unmarshal(message, &instance); err != nil {
			log.Println("error unmarshalling instances")
			writeToSocket(ws, "", err)
			return nil, err
		}
		if instance.Name == "" || instance.ProjectName == "" {
			log.Println("missing instance data values")
			err = errors.New(missingInstanceValues)
			writeToSocket(ws, "", err)
			return nil, err
		}
		return &instance, nil
	}
}

// listenForCmd continuosly reads from the websocket expecting the start-rdp or end rdp command
func (gcloudExecutor *GcloudExecutor) listenForCmd(ws conn, instance *Instance, freePort int, endChan chan<- bool) {
	for {
		log.Println("listening for cmd for", instance.Name)

		_, message, err := ws.ReadMessage()

		log.Printf("listenForCmd for %v got message %v", instance.Name, string(message))

		if err != nil {
			endChan <- true
			return
		}

		var cmd socketCmd
		if err := json.Unmarshal(message, &cmd); err != nil {
			log.Printf("listenForCmd for %v failed due to %v", instance.Name, err)
		}

		if cmd.Cmd == endRdpSocketCmd && cmd.InstanceName == instance.Name {
			writeToSocket(ws, receivedEndCmd, nil)
			endChan <- true
			return
		}

		if cmd.Cmd == startRdpSocketCmd && cmd.Username != "" {
			log.Println("starting rdp")
			writeToSocket(ws, receivedStartRdpCmd, nil)
			creds := credentials{Username: cmd.Username, Password: cmd.Password}
			go gcloudExecutor.startRdpProgram(ws, &creds, freePort, endChan)
		}

	}
}

// cleanUpRdp deletes the created firewall rules and ends the IAP tunnel and closes websocket
func (gcloudExecutor *GcloudExecutor) cleanUpRdp(ws conn, instance *Instance, cleanFirewall bool, cleanIap bool, cancelFunc context.CancelFunc) {
	log.Println("clean up rdp for ", instance.Name)
	if cleanFirewall {
		writeToSocket(ws, fmt.Sprintf(deletingIapFirewall, instance.Name), nil)
		log.Println("deleting iap firewall for ", instance.Name)
		gcloudExecutor.deleteIapFirewall(instance)
	}
	if cleanIap {
		writeToSocket(ws, fmt.Sprintf(endingIapTunnel, instance.Name), nil)
		log.Println("ending rdp for ", instance.Name)
		cancelFunc()
	}
	writeToSocket(ws, fmt.Sprintf(shutDownRdp, instance.Name), nil)
	ws.Close()
}
