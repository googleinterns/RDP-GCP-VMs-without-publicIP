package admin

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	endOperationCmd         string        = "end_operation"
	operationContextTimeout time.Duration = 5 * time.Minute
)

type shell interface {
	ExecuteCmd(string) ([]byte, error)
	ExecuteCmdReader(string) ([]io.ReadCloser, context.CancelFunc, error)
}

// NewAdminExecutor is used to call gcloud functions with the shell passed in.
type AdminExecutor struct {
	shell shell
}

type socketCmd struct {
	Cmd  string `json:"cmd"`
	Hash string `json:"hash"`
}

// socketMessage is the struct that is sent to the websockets
type socketMessage struct {
	Message string `json:"message"`
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

// WriteToSocket is a wrapper that is used to write JSON to the websocket
func WriteToSocket(ws conn, message string, err error) error {
	if err := ws.WriteJSON(newSocketMessage(message, err)); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// NewAdminExecutor creates a new gcloudExecutor struct with a struct that implements shell.
func NewAdminExecutor(shell shell) *AdminExecutor {
	return &AdminExecutor{
		shell: shell,
	}
}

// conn interface is used to mock websocket connections
type conn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteJSON(v interface{}) error
	Close() error
}

// StartPrivateRdp is a task runner that runs all the individual functions for automated RDP.
func (adminExecutor *AdminExecutor) RunOperation(ws *websocket.Conn, operationToRun *OperationToRun) {
	ctx, cancel := context.WithTimeout(context.Background(), operationContextTimeout)
	endOperationChan := make(chan bool)

	if err := WriteToSocket(ws, fmt.Sprintf("Server received operation %s", operationToRun.Operation), nil); err != nil {
		cancel()
		return
	}

	go listenForCmd(ws, operationToRun, endOperationChan)
	go adminExecutor.ExecuteOperation(ctx, ws, operationToRun, endOperationChan)
	<-endOperationChan
	cancel()
	WriteToSocket(ws, fmt.Sprintf("Ended operation"), nil)
	return

}

func sendOutputToConn(ws *websocket.Conn, scanner *bufio.Scanner, prefix string, wg *sync.WaitGroup, operationDone chan<- bool) {
	for scanner.Scan() {
		line := scanner.Text()
		if err := WriteToSocket(ws, fmt.Sprintf("%v: %v", prefix, line), nil); err != nil {
			log.Println(err)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		WriteToSocket(ws, "", err)
		operationDone <- true
	}

	wg.Done()
}

func (adminExecutor *AdminExecutor) ExecuteOperation(ctx context.Context, ws *websocket.Conn, operation *OperationToRun, operationDoneChan chan<- bool) {
	log.Println("Running operation %v", operation.Operation)

	output, cmdCancel, err := adminExecutor.shell.ExecuteCmdReader(operation.Operation)
	if err != nil {
		log.Println(err)
		WriteToSocket(ws, "", err)

		operationDoneChan <- true
		return
	}

	stdoutScanner, stderrScanner := bufio.NewScanner(output[0]), bufio.NewScanner(output[1])

	var wg sync.WaitGroup
	wg.Add(2)
	go sendOutputToConn(ws, stdoutScanner, "stdout", &wg, operationDoneChan)
	go sendOutputToConn(ws, stderrScanner, "stderr", &wg, operationDoneChan)

	done := make(chan bool)

	go func() {
		wg.Wait()
		done <- true
	}()

	go func() {
		<-ctx.Done()
		log.Println("calling cancel func")
		cmdCancel()
		done <- true
	}()

	<-done
	operationDoneChan <- true

}

func listenForCmd(ws conn, operationRunning *OperationToRun, endOperationChan chan<- bool) {
	for {
		log.Println("listening for end cmd")

		_, message, err := ws.ReadMessage()

		log.Printf("listenForCmd got message %v", string(message))

		if err != nil {
			endOperationChan <- true
			return
		}

		var cmd socketCmd
		if err := json.Unmarshal(message, &cmd); err != nil {
			log.Printf("listenForCmd for %v failed due to %v", operationRunning.Operation, err)
		}

		if cmd.Cmd == endOperationCmd && cmd.Hash == operationRunning.Hash {
			endOperationChan <- true
			return
		}
	}
}

// GetOperationFromConn reads the instance that is sent at the start of the websocket connection
func GetOperationFromConn(ws conn, operationPool *[]OperationToRun) (*OperationToRun, error) {
	type request struct {
		Hash string `json:"hash"`
	}

	for {
		_, message, err := ws.ReadMessage()

		log.Println("got message: ", string(message))

		if err != nil {
			log.Println("error reading message")
			return nil, err
		}

		var reqBody request
		if err := json.Unmarshal(message, &reqBody); err != nil {
			log.Println("error unmarshalling operation hash")
			return nil, err
		}

		for _, operation := range *operationPool {
			if operation.Hash == reqBody.Hash {
				if operation.Status == "running" {
					return nil, errors.New("operation already running")
				}
				operation.Status = "running"
				log.Println(operation)
				return &operation, nil
			}
		}

		return nil, errors.New("operation not found")
	}
}
