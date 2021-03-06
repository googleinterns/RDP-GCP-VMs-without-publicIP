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

// Package shell provides helper functions to run shell commands and returns the output for use.
package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/google/shlex"
)

const cmdReaderContextTimeout time.Duration = 1 * time.Hour

// CmdShell implements Shell interface, contains functions that run commands.
type CmdShell struct{}

// ExecuteCmd runs a shell command and waits for its output before returning the output
func (*CmdShell) ExecuteCmd(cmd string) ([]byte, error) {
	parsedCmd, err := shlex.Split(cmd)
	if err != nil {
		return []byte("Operation invalid"), err
	}

	for i := 0; i < len(parsedCmd); i++ {
		parsedCmd[i] = os.ExpandEnv(parsedCmd[i])
	}

	var out []byte
	if len(parsedCmd) == 0 {
		return []byte("Operation invalid"), errors.New("Invalid operation")
	}
	if len(parsedCmd) == 1 {
		out, err = exec.Command(parsedCmd[0]).CombinedOutput()
	} else {
		out, err = exec.Command(parsedCmd[0], parsedCmd[1:]...).CombinedOutput()
	}
	return out, err
}

// ExecuteCmd runs a shell command and waits for its output before returning the output
func (*CmdShell) ExecuteCmdWithContext(endContext context.Context, cmd string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdReaderContextTimeout)
	parsedCmd, err := shlex.Split(cmd)

	if err != nil {
		cancel()
		return nil, err
	}

	for i := 0; i < len(parsedCmd); i++ {
		parsedCmd[i] = os.ExpandEnv(parsedCmd[i])
	}

	c := exec.CommandContext(ctx, parsedCmd[0], parsedCmd[1:]...)

	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = &b
	err = c.Start()
	if err != nil {
		cancel()
		return nil, err
	}

	var returnErr error
	// Use a channel to signal completion so we can use a select statement
	done := make(chan bool)
	go func() {
		if err = c.Wait(); err != nil {
			returnErr = err
		}
		done <- true
	}()

	go func() {
		// Wait for context to be done
		<-endContext.Done()
		returnErr = errors.New("Operation timed out")
		cancel()
		done <- true
	}()

	<-done
	return b.Bytes(), returnErr
}

// ExecuteCmdReader runs a shell command and pipes the stdout and stderr into ReadClosers
func (*CmdShell) ExecuteCmdReader(cmd string) ([]io.ReadCloser, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdReaderContextTimeout)
	parsedCmd, err := shlex.Split(cmd)

	if err != nil {
		cancel()
		return nil, nil, err
	}

	for i := 0; i < len(parsedCmd); i++ {
		parsedCmd[i] = os.ExpandEnv(parsedCmd[i])
	}

	asyncCmd := exec.CommandContext(ctx, parsedCmd[0], parsedCmd[1:]...)

	stdout, err := asyncCmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, nil, err
	}

	stderr, err := asyncCmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, nil, err
	}

	if err = asyncCmd.Start(); err != nil {
		cancel()
		return nil, nil, err
	}

	fmt.Println("Stand by to read..")
	return []io.ReadCloser{stdout, stderr}, cancel, nil
}

// FindOpenPort finds a free port on the system and returns the listener
func FindOpenPort() (*net.TCPListener, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}

	return listener, nil
}
