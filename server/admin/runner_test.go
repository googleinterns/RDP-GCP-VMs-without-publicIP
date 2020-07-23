package admin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

type mockShell struct{}

var mockOperationToRun = OperationToRun{Operation: "echo hello", Hash: "hash", Status: "ready"}

const (
	testErr string = "test error"
)

func (*mockShell) ExecuteCmd(cmd string) ([]byte, error) {
	return nil, nil
}

func (*mockShell) ExecuteCmdReader(cmd string) ([]io.ReadCloser, context.CancelFunc, error) {
	if cmd == "output" {
		return []io.ReadCloser{ioutil.NopCloser(strings.NewReader("stdout")), ioutil.NopCloser(strings.NewReader("stderr"))}, nil, nil
	}
	if cmd == testErr {
		return nil, nil, errors.New(testErr)
	}
	return nil, nil, nil
}

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

func TestGetOperationFromConn(t *testing.T) {
	message := []byte(nil)
	messageErr := errors.New(testErr)

	readMessage := func() (messageType int, p []byte, err error) {
		return websocket.TextMessage, message, messageErr
	}

	writeJSON := func(v interface{}) error {
		return nil
	}

	closeFunc := func() error {
		return nil
	}

	ws := newMockWebSocket(readMessage, writeJSON, closeFunc)

	operationPool := []OperationToRun{}

	_, err := ReadOperationHashFromConn(ws, &operationPool)
	if err.Error() != testErr {
		t.Errorf("GetOperationFromConn didn't error from socket ReadMessage error")
	}

	messageErr = nil

	message = []byte(`{"hash": "test"`)
	_, err = ReadOperationHashFromConn(ws, &operationPool)
	if err.Error() != "unexpected end of JSON input" {
		t.Errorf("GetOperationFromConn didn't error from bad JSON sent")
	}

	message = []byte(`{"hash": "bad hash"}`)
	_, err = ReadOperationHashFromConn(ws, &operationPool)
	if err.Error() != fmt.Sprintf(operationNotFound, "bad hash") {
		t.Errorf("GetOperationFromConn didn't error from missing values, got %v, expected %v", err.Error(), fmt.Sprintf(operationNotFound, "bad hash"))
	}

	message = []byte(`{"hash": "hash"}`)

	operationPool = append(operationPool, mockOperationToRun)

	operation, err := ReadOperationHashFromConn(ws, &operationPool)
	runningOperation := mockOperationToRun
	runningOperation.Status = "running"
	if !reflect.DeepEqual(*operation, runningOperation) {
		t.Errorf("GetOperationFromConn returned wrong operation, got %v, expected %v", *operation, runningOperation)
	}
	if err != nil {
		t.Errorf("GetOperationFromConn errored out on valid instances")
	}

	operationPool[0].Status = "running"
	_, err = ReadOperationHashFromConn(ws, &operationPool)
	if err.Error() != fmt.Sprintf(operationRunning, "hash") {
		t.Errorf("GetOperationFromConn didn't error from missing values, got %v, expected %v", err.Error(), fmt.Sprintf(operationRunning, "hash"))
	}
}

func TestListenForCmd(t *testing.T) {
	message := []byte(nil)
	messageErr := errors.New(testErr)

	readOnce := false
	readMessage := func() (messageType int, p []byte, err error) {
		if !readOnce {
			readOnce = true
			return websocket.TextMessage, message, messageErr
		}
		return 0, nil, errors.New(testErr)

	}

	writeJSON := func(v interface{}) error {
		return nil
	}

	closeFunc := func() error {
		return nil
	}

	ws := newMockWebSocket(readMessage, writeJSON, closeFunc)

	endChan := make(chan bool)
	operationRunning := mockOperationToRun
	operationRunning.Status = "running"

	go listenForEndCmd(ws, &operationRunning, endChan)
	if end := <-endChan; end != true {
		t.Errorf("listenForCmd didn't set quit channel out on readmessage error")
	}

	messageErr = nil

	message = []byte(`{"cmd": "end_operation", "hash":"hash"}`)
	endChan = make(chan bool)
	readOnce = false
	go listenForEndCmd(ws, &operationRunning, endChan)
	if end := <-endChan; end != true {
		log.Println(end)
		t.Errorf("listenForCmd didn't set quit channel out on end cmd")
	}
}

func TestSendOutputToConn(t *testing.T) {
	var socketOutput socketMessage

	readMessage := func() (messageType int, p []byte, err error) {
		return 0, nil, nil
	}

	writeJSON := func(v interface{}) error {
		socketOutput = *(v.(*socketMessage))
		return nil
	}

	closeFunc := func() error {
		return nil
	}

	ws := newMockWebSocket(readMessage, writeJSON, closeFunc)

	expectedOutput := "test output"

	scanner := bufio.NewScanner(strings.NewReader(expectedOutput))

	var wg sync.WaitGroup
	wg.Add(1)

	go sendOutputToConn(ws, scanner, true, &wg)
	wg.Wait()
	if socketOutput.Stdout != expectedOutput {
		t.Errorf("sendOutputToConn didn't send proper stdout message, got %v, expected %v", socketOutput.Stdout, expectedOutput)
	}
	if socketOutput.Stderr != "" || socketOutput.Err != "" || socketOutput.ServerMessage != "" {
		t.Errorf("sendOutputToConn sent a bad websocket message")
	}

	wg.Add(1)

	scanner = bufio.NewScanner(strings.NewReader(expectedOutput))

	go sendOutputToConn(ws, scanner, false, &wg)
	wg.Wait()
	if socketOutput.Stderr != expectedOutput {
		log.Println(socketOutput.Stderr)
		log.Println(expectedOutput)
		t.Errorf("sendOutputToConn didn't send proper stderr message, got %v, expected %v", socketOutput.Stderr, expectedOutput)
	}
	if socketOutput.Stdout != "" || socketOutput.Err != "" || socketOutput.ServerMessage != "" {
		t.Errorf("sendOutputToConn sent a bad websocket message")
	}
}

func TestExecuteOperation(t *testing.T) {
	var socketOutput []socketMessage

	readMessage := func() (messageType int, p []byte, err error) {
		return websocket.TextMessage, nil, nil
	}

	writeJSON := func(v interface{}) error {
		socketOutput = append(socketOutput, *(v.(*socketMessage)))
		return nil
	}

	closeFunc := func() error {
		return nil
	}

	ws := newMockWebSocket(readMessage, writeJSON, closeFunc)

	adminExecutor := NewAdminExecutor(&mockShell{})
	operation := mockOperationToRun
	ctx, _ := context.WithTimeout(context.Background(), operationContextTimeout)
	operationDoneChan := make(chan bool)
	operation.Operation = testErr

	go adminExecutor.executeOperation(ctx, ws, &operation, operationDoneChan)
	operationDone := <-operationDoneChan

	if operationDone != true {
		t.Errorf("executeOperation didn't set operation done channel to true on error")
	}
	if socketOutput[0].Err != testErr {
		t.Errorf("executeOperation didn't write proper error to socket, got %v, expected %v", socketOutput[0].Err, testErr)
	}

	operationDoneChan = make(chan bool)
	operation.Operation = "output"

	socketOutput = []socketMessage{}

	go adminExecutor.executeOperation(ctx, ws, &operation, operationDoneChan)
	operationDone = <-operationDoneChan
	if operationDone != true {
		t.Errorf("executeOperation didn't set operation done channel to true after output waitgroup finished")
	}

	var expectedStdout bool
	var expectedStderr bool
	for _, output := range socketOutput {
		if output.Stdout == "stdout" {
			expectedStdout = true
		} else if output.Stderr == "stderr" {
			expectedStderr = true
		}
	}

	if !expectedStdout {
		t.Errorf("executeOperation didn't write proper stdout message to socket, expected %v", "stdout")
	}
	if !expectedStderr {
		t.Errorf("executeOperation didn't write proper stderr message to socket, expected %v", "stderr")
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), operationContextTimeout)
	go adminExecutor.executeOperation(ctx, ws, &operation, operationDoneChan)
	cancelFunc()
	operationDone = <-operationDoneChan
	if operationDone != true {
		t.Errorf("executeOperation didn't set operation done channel to true after ctx cancelFunc called")
	}
}

func TestRunOperation(t *testing.T) {
	var socketOutput []socketMessage

	readMessage := func() (messageType int, p []byte, err error) {
		return websocket.TextMessage, nil, nil
	}

	writeJSON := func(v interface{}) error {
		socketOutput = append(socketOutput, *(v.(*socketMessage)))
		return nil
	}

	closeFunc := func() error {
		return nil
	}

	ws := newMockWebSocket(readMessage, writeJSON, closeFunc)

	adminExecutor := NewAdminExecutor(&mockShell{})
	operation := mockOperationToRun
	operation.Operation = testErr

	adminExecutor.RunOperation(ws, &operation)

	if socketOutput[0].ServerMessage != fmt.Sprintf(serverReceivedOperation, testErr) {
		t.Errorf("RunOperation didn't write proper acknowledgement to socket, got %v, expected %v", socketOutput[0].ServerMessage, fmt.Sprintf(serverReceivedOperation, testErr))
	}

	if socketOutput[1].Err != testErr {
		t.Errorf("RunOperation didn't write proper error to socket, got %v, expected %v", socketOutput[1].Err, testErr)
	}

	if socketOutput[2].ServerMessage != fmt.Sprintf(operationEnded, "hash") {
		t.Errorf("RunOperation didn't write proper end message to socket, got %v, expected %v", socketOutput[2].ServerMessage, fmt.Sprintf(operationEnded, "hash"))
	}

	socketOutput = []socketMessage{}

	operation.Operation = "output"
	adminExecutor.RunOperation(ws, &operation)

	var expectedServerMessage bool
	var expectedStdout bool
	var expectedStderr bool

	for _, output := range socketOutput {
		if output.ServerMessage == fmt.Sprintf(serverReceivedOperation, "output") {
			expectedServerMessage = true
		} else if output.Stdout == "stdout" {
			expectedStdout = true
		} else if output.Stderr == "stderr" {
			expectedStderr = true
		}
	}

	if !expectedServerMessage {
		t.Errorf("RunOperation didn't write proper acknowledgement to socket, expected %v", fmt.Sprintf(serverReceivedOperation, "output"))
	}

	if !expectedStdout {
		t.Errorf("RunOperation didn't write proper error to socket, expected %v", "stdout")
	}

	if !expectedStderr {
		t.Errorf("RunOperation didn't write proper end message to socket, expected %v", "stderr")
	}
}
