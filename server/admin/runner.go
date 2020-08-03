package admin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

const (
	endOperationCmd         string        = "end_operation"
	operationContextTimeout time.Duration = 5 * time.Minute
	operationNotFound       string        = "operation with hash %v not found"
	operationRunning        string        = "operation with hash %v already running"
	operationEnded          string        = "operation with hash %v ended"
	serverReceivedOperation string        = "server received operation: %v"
)

type shell interface {
	ExecuteCmd(string) ([]byte, error)
	ExecuteCmdWithContext(context.Context, string) ([]byte, error)
	ExecuteCmdReader(string) ([]io.ReadCloser, context.CancelFunc, error)
}

// NewAdminExecutor is used to call gcloud functions with the shell passed in.
type AdminExecutor struct {
	shell shell
}

// socketCmd struct is used to read commands such as end-operation from the websocket
type socketCmd struct {
	Cmd  string `json:"cmd"`
	Hash string `json:"hash"`
}

// socketMessage is the struct that is sent to the websockets
type socketMessage struct {
	ServerMessage string `json:"message"`
	Stdout        string `json:"stdout"`
	Stderr        string `json:"stderr"`
	Err           string `json:"error"`
}

// newSocketMessage creates a socketMessage struct
func newSocketMessage(message, stdout, stderr string, err error) *socketMessage {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	return &socketMessage{
		ServerMessage: message,
		Stdout:        stdout,
		Stderr:        stderr,
		Err:           errorMessage,
	}
}

// WriteToSocket is a wrapper that is used to write JSON to the websocket
func WriteToSocket(ws conn, message, stdout, stderr string, err error) error {
	if err := ws.WriteJSON(newSocketMessage(message, stdout, stderr, err)); err != nil {
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

// RunOperation is a runner that listens for any commands and executes the actual operation
func (adminExecutor *AdminExecutor) RunOperation(ws conn, operationToRun *OperationToRun) {
	ctx, cancel := context.WithTimeout(context.Background(), operationContextTimeout)
	endOperationChan := make(chan bool)

	if err := WriteToSocket(ws, fmt.Sprintf(serverReceivedOperation, operationToRun.Operation), "", "", nil); err != nil {
		cancel()
		return
	}

	go listenForEndCmd(ws, operationToRun, endOperationChan)

	if operationToRun.RealtimeOutput {
		log.Println("Realtime")
		go adminExecutor.executeOperation(ctx, ws, operationToRun, endOperationChan)
	} else {
		go adminExecutor.executeOperationInstant(ctx, ws, operationToRun, endOperationChan)
	}
	<-endOperationChan
	cancel()
	WriteToSocket(ws, fmt.Sprintf(operationEnded, operationToRun.Hash), "", "", nil)
	return
}

// sendOutputToConn sends the output from the operation to the websocket
func sendOutputToConn(ws conn, scanner *bufio.Scanner, stdout bool, wg *sync.WaitGroup) {
	for scanner.Scan() {
		line := scanner.Text()
		log.Println(line)

		if stdout {
			if err := WriteToSocket(ws, "", line, "", nil); err != nil {
				log.Println(err)
				break
			}
		} else {
			if err := WriteToSocket(ws, "", "", line, nil); err != nil {
				log.Println(err)
				break
			}
		}

	}

	if err := scanner.Err(); err != nil {
		WriteToSocket(ws, "", "", "", err)
	}

	wg.Done()
}

// executeOperationInstant executes the actual operation, waits till termination and sends one output
func (adminExecutor *AdminExecutor) executeOperationInstant(ctx context.Context, ws conn, operation *OperationToRun, operationDoneChan chan<- bool) {
	log.Println("Running operation", operation.Operation)

	output, err := adminExecutor.shell.ExecuteCmdWithContext(ctx, operation.Operation)
	if err != nil {
		log.Println(err)
		WriteToSocket(ws, "", string(output), "", err)

		operationDoneChan <- true
		return
	}

	WriteToSocket(ws, "", string(output), "", nil)

	operationDoneChan <- true
}

// executeOperation executes the actual operation and pipes the stdout and stderr
func (adminExecutor *AdminExecutor) executeOperation(ctx context.Context, ws conn, operation *OperationToRun, operationDoneChan chan<- bool) {
	log.Println("Running operation", operation.Operation)

	output, cmdCancel, err := adminExecutor.shell.ExecuteCmdReader(operation.Operation)
	if err != nil {
		log.Println(err)
		WriteToSocket(ws, "", "", "", err)

		operationDoneChan <- true
		return
	}

	stdoutScanner, stderrScanner := bufio.NewScanner(output[0]), bufio.NewScanner(output[1])

	var wg sync.WaitGroup
	wg.Add(2)
	go sendOutputToConn(ws, stdoutScanner, true, &wg)
	go sendOutputToConn(ws, stderrScanner, false, &wg)

	done := make(chan bool)

	// These goroutines run concurrently waiting for either the waitgroup to finish or the
	// context to be done before ending the operation.
	go func() {
		// Wait for cmd to finish outputting
		wg.Wait()
		done <- true
	}()

	go func() {
		// Wait for context to be done
		<-ctx.Done()
		log.Println("calling cancel func")
		if cmdCancel != nil {
			cmdCancel()
		}
		done <- true
	}()

	<-done
	operationDoneChan <- true
}

// listenForEndCmd listens for commands from the extension
func listenForEndCmd(ws conn, operationRunning *OperationToRun, endOperationChan chan<- bool) {
	for {
		log.Println("listening for end cmd")

		_, message, err := ws.ReadMessage()

		log.Printf("listenForEndCmd got message %v", string(message))

		if err != nil {
			log.Println(err)
			endOperationChan <- true
			return
		}

		var cmd socketCmd
		if err := json.Unmarshal(message, &cmd); err != nil {
			log.Printf("listenForEndCmd for %v failed due to %v", operationRunning.Operation, err)
		}

		if cmd.Cmd == endOperationCmd && cmd.Hash == operationRunning.Hash {
			log.Printf("listenForEndCmd for %v successfully read", operationRunning.Operation)
			endOperationChan <- true
			return
		}
	}
}

// ReadOperationHashFromConn reads the instance that is sent at the start of the websocket connection
func ReadOperationHashFromConn(ws conn, operationPool *[]OperationToRun) (*OperationToRun, error) {
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

		// Check if operation hash exists in pool and if status is not running
		for _, operation := range *operationPool {
			if operation.Hash == reqBody.Hash {
				if operation.Status == "running" {
					return nil, fmt.Errorf(operationRunning, reqBody.Hash)
				}
				operation.Status = "running"
				log.Println(operation)
				return &operation, nil
			}
		}

		return nil, fmt.Errorf(operationNotFound, reqBody.Hash)
	}
}
