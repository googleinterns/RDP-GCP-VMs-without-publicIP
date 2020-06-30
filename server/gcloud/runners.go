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
	"time"

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

func getComputeInstanceFromConn(ws websocketConn) (*Instance, error) {
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
			err = errors.New("missing values from data sent")
			writeToSocket(ws, "", err)
			return nil, err
		}
		return &instance, nil
	}
}

func (gcloudExecutor *GcloudExecutor) listenForCmd(ws websocketConn, instance *Instance, freePort int, endChan chan<- bool, rdpQuitChan chan<- bool) {
	type socketCmd struct {
		Cmd          string `json:"cmd"`
		InstanceName string `json:"name"`
		Username     string `json:"username"`
		Password     string `json:"password"`
	}
	for {
		log.Println("listening for cmd for", instance.Name)

		_, message, err := ws.ReadMessage()

		log.Println("got message:", string(message))

		if err != nil {
			endChan <- true
			return
		}

		var cmd socketCmd
		if err := json.Unmarshal(message, &cmd); err != nil {
			log.Println("error unmarshalling socket command for ", instance.Name)
		}

		if cmd.Cmd == "end" && cmd.InstanceName == instance.Name {
			endChan <- true
			return
		}

		if cmd.Cmd == "start-rdp" && cmd.Username != "" {
			creds := credentials{Username: cmd.Username, Password: cmd.Password}
			go gcloudExecutor.startRdpProgram(ws, &creds, freePort, rdpQuitChan)
		}

	}
}

func (gcloudExecutor *GcloudExecutor) cleanUpRdp(ws websocketConn, instance *Instance, cleanFirewall bool, cleanRdp bool, cancelFunc context.CancelFunc) {
	log.Println("clean up rdp for ", instance.Name)
	if cleanFirewall {
		log.Println("deleting iap firewall for ", instance.Name)
		gcloudExecutor.deleteIapFirewall(instance)
	}
	if cleanRdp {
		log.Println("ending rdp for ", instance.Name)
		cancelFunc()
	}
	writeToSocket(ws, fmt.Sprintf("shut down private rdp for %v", instance.Name), nil)
	ws.Close()
}

// StartPrivateRdp is a task runner that runs all the individual functions for automated RDP.
func StartPrivateRdp(ws *websocket.Conn) {
	shell := &pshell.CmdShell{}
	g := NewGcloudExecutor(shell)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	iapOutputChan := make(chan iapResult)
	rdpQuitChan := make(chan bool)
	endRdpChan := make(chan bool)

	instanceToConn, err := getComputeInstanceFromConn(ws)
	if err != nil {
		g.cleanUpRdp(ws, instanceToConn, false, false, cancel)
		return
	}
	if err := writeToSocket(ws, fmt.Sprintf("Server received instance %s", instanceToConn.Name), err); err != nil {
		g.cleanUpRdp(ws, instanceToConn, false, false, cancel)
		return
	}

	log.Println("Got instance", instanceToConn.Name)

	if err := g.createIapFirewall(ws, instanceToConn); err != nil {
		g.cleanUpRdp(ws, instanceToConn, false, false, cancel)
		return
	}

	freePort, err := pshell.GetPort()
	if err != nil {
		writeToSocket(ws, "", errors.New("Could not get a unused port on system"))
		g.cleanUpRdp(ws, instanceToConn, true, false, cancel)
		return
	}

	log.Println("Got free port ", freePort)

	go g.startIapTunnel(ctx, ws, instanceToConn, freePort, iapOutputChan)
	if output := <-iapOutputChan; !output.tunnelCreated || output.err != nil {
		g.cleanUpRdp(ws, instanceToConn, true, false, cancel)
		return
	}

	go g.listenForCmd(ws, instanceToConn, freePort, endRdpChan, rdpQuitChan)

	if endRdp := <-endRdpChan; endRdp {
		g.cleanUpRdp(ws, instanceToConn, true, true, cancel)
		return
	}

	if rdpQuit := <-rdpQuitChan; rdpQuit {
		g.cleanUpRdp(ws, instanceToConn, true, true, cancel)
		return
	}
}

func writeToSocket(ws websocketConn, message string, err error) error {
	if err := ws.WriteJSON(newSocketMessage(message, err)); err != nil {
		log.Println(err)
		return err
	}
	return nil
}
