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
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/google/shlex"
)

const cmdReaderContextTimeout time.Duration = 1 * time.Hour

// CmdShell implements Shell interface, contains functions that run commands.
type CmdShell struct{}

// ExecuteCmd runs a shell command and waits for its output before returning the output
func (*CmdShell) ExecuteCmd(cmd string) ([]byte, error) {
	parsedCmd := strings.Fields(cmd)
	out, err := exec.Command(parsedCmd[0], parsedCmd[1:]...).CombinedOutput()
	return out, err
}

// ExecuteCmdReader runs a shell command and pipes the stdout and stderr into ReadClosers
func (*CmdShell) ExecuteCmdReader(cmd string) ([]io.ReadCloser, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdReaderContextTimeout)
	parsedCmd, err := shlex.Split(cmd)

	if err != nil {
		cancel()
		return nil, nil, err
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
