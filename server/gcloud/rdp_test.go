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
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type mockWebSocket struct {
	readMessageFunc func() (messageType int, p []byte, err error)
	writeJSONFunc   func(v interface{}) error
	closeFunc       func() error
}

func newMockWebSocket(readMessageFunc func() (messageType int, p []byte, err error), writeJSONFunc func(v interface{}) error, closeFunc func() error) mockWebSocket {
	ws := mockWebSocket{}
	ws.readMessageFunc = readMessageFunc
	ws.writeJSONFunc = writeJSONFunc
	ws.closeFunc = closeFunc
	return ws
}

func (m mockWebSocket) Close() error {
	return m.closeFunc()
}

func (m mockWebSocket) ReadMessage() (messageType int, p []byte, err error) {
	return m.readMessageFunc()
}

func (m mockWebSocket) WriteJSON(v interface{}) error {
	return m.writeJSONFunc(v)
}

var instance []byte = []byte(strings.Trim(strings.Trim(string(validComputeInstanceOutput), "["), "]"))

func TestGetComputeInstanceFromConn(t *testing.T) {
	message := []byte(nil)
	messageErr := errors.New("test error")

	var socketOutput socketMessage

	readMessage := func() (messageType int, p []byte, err error) {
		return websocket.TextMessage, message, messageErr
	}

	writeJSON := func(v interface{}) error {
		socketOutput = *(v.(*socketMessage))
		return nil
	}

	closeFunc := func() error {
		return nil
	}

	ws := newMockWebSocket(readMessage, writeJSON, closeFunc)

	_, err := getComputeInstanceFromConn(ws)
	if socketOutput.Err != "test error" && err.Error() != "test error" {
		t.Errorf("getComputeInstancesFromConn didn't error from socket ReadMessage error")
	}

	messageErr = nil

	message = []byte(`{"test": "test"`)
	_, err = getComputeInstanceFromConn(ws)
	if socketOutput.Err != "unexpected end of JSON input" && err.Error() != "unexpected end of JSON input" {
		t.Errorf("getComputeInstancesFromConn didn't error from bad JSON sent")
	}

	message = []byte(`{"test": "test"}`)
	_, err = getComputeInstanceFromConn(ws)
	if expected := missingInstanceValues; socketOutput.Err != expected && err.Error() != expected {
		t.Errorf("getComputeInstancesFromConn didn't error from missing values, got %v, expected %v", socketOutput.Message, expected)
	}

	message = instance

	var validInstances Instance
	json.Unmarshal(message, &validInstances)

	instance, err := getComputeInstanceFromConn(ws)
	if socketOutput.Err != "" && err != nil {
		t.Errorf("getComputeInstancesFromConn errored out on valid instances")
	}

	if !reflect.DeepEqual(validInstances, *instance) {
		t.Errorf("GetComputeInstances failed getting instances, got %v, expected %v", *instance, validInstances)
	}
}

func TestListenForCmd(t *testing.T) {
	message := []byte(nil)
	messageErr := errors.New("test error")

	readOnce := false
	readMessage := func() (messageType int, p []byte, err error) {
		if !readOnce {
			readOnce = true
			return websocket.TextMessage, message, messageErr
		}
		return 0, nil, errors.New("Hello")

	}

	writeJSON := func(v interface{}) error {
		return nil
	}

	closeFunc := func() error {
		return nil
	}

	ws := newMockWebSocket(readMessage, writeJSON, closeFunc)

	var instanceToUse Instance
	json.Unmarshal(instance, &instanceToUse)

	freePort := 9999
	endChan := make(chan bool)
	g := NewGcloudExecutor(&mockShell{})

	go g.listenForCmd(ws, &instanceToUse, freePort, endChan)
	if end := <-endChan; end != true {
		t.Errorf("listenForCmd didn't set quit channel out on readmessage error")
	}

	messageErr = nil

	message = []byte(`{"cmd": "end", "name":"test-project"}`)
	endChan = make(chan bool)
	readOnce = false
	go g.listenForCmd(ws, &instanceToUse, freePort, endChan)
	if end := <-endChan; end != true {
		t.Errorf("listenForCmd didn't set quit channel out on end cmd")
	}

	message = []byte(`{"cmd": "start-rdp", "username":"quit", "password": "password"}`)
	endChan = make(chan bool)
	readOnce = false
	go g.listenForCmd(ws, &instanceToUse, freePort, endChan)
	if end := <-endChan; end != true {
		t.Errorf("listenForCmd didn't set quit channel out on rdp quit")
	}
}

func TestReadIapTunnelOutput(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(gcloudErrorOutput))
	var cmdOutput []string
	tunnelCreated := false
	iapOutputChan := make(chan iapResult)

	go readIapTunnelOutput(scanner, &tunnelCreated, &cmdOutput, iapOutputChan)
	if iapResult := <-iapOutputChan; iapResult.tunnelCreated {
		fmt.Println(iapResult)
		t.Errorf("readIapTunnelOutput set tunnel created on gcloud error")
	}

	scanner = bufio.NewScanner(strings.NewReader(tunnelCreatedOutput))
	go readIapTunnelOutput(scanner, &tunnelCreated, &cmdOutput, iapOutputChan)
	if iapResult := <-iapOutputChan; !iapResult.tunnelCreated {
		fmt.Println(iapResult)
		t.Errorf("readIapTunnelOutput didn't set tunnel created on tunnel created output")
	}
}

func TestStartRdpProgram(t *testing.T) {
	var socketOutput socketMessage

	readMessage := func() (messageType int, p []byte, err error) {
		return websocket.TextMessage, nil, nil
	}

	writeJSON := func(v interface{}) error {
		socketOutput = *(v.(*socketMessage))
		return nil
	}

	closeFunc := func() error {
		return nil
	}

	ws := newMockWebSocket(readMessage, writeJSON, closeFunc)

	port := 9999
	creds := &credentials{Username: "error", Password: "password"}
	quitChan := make(chan bool)
	g := NewGcloudExecutor(&mockShell{})

	g.startRdpProgram(ws, creds, port, quitChan)
	if expected := fmt.Sprintf(rdpProgramError, "error"); socketOutput.Err != expected {
		t.Errorf("startRdpProgram didn't write error to socket, got %v, expected %v", socketOutput.Message, expected)
	}

	creds = &credentials{Username: "quit", Password: "password"}
	go g.startRdpProgram(ws, creds, port, quitChan)
	quit := <-quitChan
	if !quit {
		t.Errorf("startRdpProgram didn't set quit channel to true on quit")
	}
	if expected := fmt.Sprintf(rdpProgramQuit, "quit"); socketOutput.Message != expected {
		t.Errorf("startRdpProgram didn't write quit to socket, got %v, expected %v", socketOutput.Message, expected)
	}
}

func TestCreateIapFirewall(t *testing.T) {
	message := []byte(nil)
	messageErr := errors.New("test error")

	var socketOutput socketMessage

	readMessage := func() (messageType int, p []byte, err error) {
		return websocket.TextMessage, message, messageErr
	}

	writeJSON := func(v interface{}) error {
		socketOutput = *(v.(*socketMessage))
		return nil
	}

	closeFunc := func() error {
		return nil
	}

	ws := newMockWebSocket(readMessage, writeJSON, closeFunc)

	var instanceToUse Instance
	json.Unmarshal(instance, &instanceToUse)
	g := NewGcloudExecutor(&mockShell{})

	instanceToUse.ProjectName = "auth-error"
	err := g.createIapFirewall(ws, &instanceToUse)
	if expected := fmt.Sprintf(didntCreateFirewallOutput, instanceToUse.Name); socketOutput.Message != expected {
		t.Errorf("createIapFirewall didn't send message to socket about not creating firewall due to auth error, got %v, expected %v", socketOutput.Message, expected)
	}
	if expected := SdkAuthError; socketOutput.Err != expected && err.Error() != expected {
		t.Errorf("createIapFirewall didn't error from auth error output, got %v, expected %v", socketOutput.Message, expected)
	}

	instanceToUse.ProjectName = "project-error"
	err = g.createIapFirewall(ws, &instanceToUse)
	if expected := fmt.Sprintf(didntCreateFirewallOutput, instanceToUse.Name); socketOutput.Message != expected {
		t.Errorf("createIapFirewall didn't send message to socket about not creating firewall due to invalid project error, got %v, expected %v", socketOutput.Message, expected)
	}
	if expected := SdkProjectError; socketOutput.Err != expected && err.Error() != expected {
		t.Errorf("createIapFirewall didn't error from invalid project error output, got %v, expected %v", socketOutput.Message, expected)
	}

	instanceToUse.ProjectName = "exists"
	err = g.createIapFirewall(ws, &instanceToUse)
	if expected := fmt.Sprintf(firewallRuleAlreadyExistsOutput, instanceToUse.Name); socketOutput.Message != expected {
		t.Errorf("createIapFirewall didn't send message to socket about firewall existing, got %v, expected %v", socketOutput.Message, expected)
	}
	if err != nil {
		t.Errorf("createIapFirewall errored from firewall existing")
	}

	instanceToUse.ProjectName = "valid"
	err = g.createIapFirewall(ws, &instanceToUse)
	if expected := fmt.Sprintf(createdFirewallOutput, instanceToUse.Name); socketOutput.Message != expected {
		t.Errorf("createIapFirewall didn't send message to socket about creating firewall, got %v, expected %v", socketOutput.Message, expected)
	}
	if err != nil {
		t.Errorf("createIapFirewall errored from creating firewall")
	}
}

func TestStartIapTunnel(t *testing.T) {
	var socketOutput socketMessage

	readMessage := func() (messageType int, p []byte, err error) {
		return websocket.TextMessage, nil, nil
	}

	writeJSON := func(v interface{}) error {
		socketOutput = *(v.(*socketMessage))
		return nil
	}

	closeFunc := func() error {
		return nil
	}

	ws := newMockWebSocket(readMessage, writeJSON, closeFunc)

	var instanceToUse Instance
	json.Unmarshal(instance, &instanceToUse)
	g := NewGcloudExecutor(&mockShell{})

	addr, _ := net.ResolveTCPAddr("tcp", "localhost:9999")
	port, _ := net.ListenTCP("tcp", addr)
	outputChan := make(chan iapResult)
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Hour)

	instanceToUse.ProjectName = "invalid"
	go g.startIapTunnel(ctx, ws, &instanceToUse, port, outputChan)
	output := <-outputChan

	if output.tunnelCreated {
		t.Errorf("startIapTunnel created tunnel on error")
	}
	if expected := []string{gcloudErrorOutput}; !reflect.DeepEqual(output.cmdOutput, expected) {
		t.Errorf("startIapTunnel output chan cmdoutput not equal to output from cmd, got %v, expected %v", socketOutput.Message, expected)
	}
	if expected := fmt.Sprintf(iapTunnelError, instanceToUse.Name); socketOutput.Err != expected {
		t.Errorf("startIapTunnel didn't write iapTunnelError to socket, got %v, expected %v", socketOutput.Err, expected)
	}

	instanceToUse.ProjectName = "valid"
	go g.startIapTunnel(ctx, ws, &instanceToUse, port, outputChan)
	output = <-outputChan

	if !output.tunnelCreated {
		t.Errorf("startIapTunnel didn't create tunnel on valid output")
	}
	if expected := []string{tunnelCreatedOutput}; !reflect.DeepEqual(output.cmdOutput, expected) {
		t.Errorf("startIapTunnel output chan cmdoutput not equal to output from cmd, got %v, expected %v", socketOutput.Message, expected)
	}
	if expected := fmt.Sprintf(iapTunnelStarted, instanceToUse.Name, 9999); socketOutput.Message != expected {
		t.Errorf("startIapTunnel didn't write iapTunnelStarted to socket, got %v, expected %v", socketOutput.Message, expected)
	}

}
