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

package shell

import (
	"bufio"
	"bytes"
	"context"
	"regexp"
	"testing"
	"time"
)

const (
	invalidCmd      string = "foo"
	validCmd        string = `echo hello`
	validContextCmd string = `bash -c 'while true; do "echo hello"; sleep 1; done'`
	validReaderCmd  string = `echo stdout; echo stderr >&2;`
)

// TestExecuteCmd tests the ExecuteCmd method which runs a shell cmd and waits for it output before testing
func TestExecuteCmd(t *testing.T) {
	shell := CmdShell{}
	if _, err := shell.ExecuteCmd(invalidCmd); err == nil {
		t.Errorf("Cmd didn't error on invalid cmd")
	}

	if validOutput, err := shell.ExecuteCmd(validCmd); validOutput == nil || err != nil || bytes.Equal(validOutput, []byte("hello")) {
		t.Errorf("Cmd failed, expected %v, got %v", "hello", string(validOutput))
	}
}

func TestExecuteCmdWithContext(t *testing.T) {
	shell := CmdShell{}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	if invalidOutput, err := shell.ExecuteCmdWithContext(ctx, invalidCmd); invalidOutput != nil && err == nil {
		t.Errorf("ExecuteCmdWithContext didn't error on invalid cmd")
	}

	r := regexp.MustCompile("hello")

	if output, err := shell.ExecuteCmdWithContext(ctx, validContextCmd); len(r.FindAllStringIndex(string(output), -1)) != 2 && err != nil {
		t.Errorf("ExecuteCmdWithContext didn't stop execution on context expiry, got %v occurences of hello, expecting %v", len(r.FindAllStringIndex(string(output), -1)), 2)
	}

	ctx, _ = context.WithTimeout(context.Background(), 2*time.Second)

	if validOutput, err := shell.ExecuteCmdWithContext(ctx, validCmd); validOutput == nil || err != nil || bytes.Equal(validOutput, []byte("hello")) {
		t.Errorf("Cmd failed, expected %v, got %v", "hello", string(validOutput))
	}
}

// TestExecuteCmdReader tests the ExecuteCmdReader method which outputs stdout/stderr as a ReadCloser
func TestExecuteCmdReader(t *testing.T) {
	shell := CmdShell{}
	if invalidOutput, _, err := shell.ExecuteCmdReader(invalidCmd); invalidOutput != nil && err == nil {
		t.Errorf("CmdReader didn't error on invalid cmd")
	}

	output, _, err := shell.ExecuteCmdReader(validReaderCmd)
	if err != nil {
		t.Errorf("Valid CmdReader cmd error'd out %v", err)
	}

	stdout, stderr := output[0], output[1]

	stdoutScanner := bufio.NewScanner(stdout)
	go func() {
		for stdoutScanner.Scan() {
			if line := stdoutScanner.Text(); line != "stdout" {
				t.Errorf("CmdReader failed, expected %v, got %v", "stdout", line)
			}
		}
	}()

	stderrScanner := bufio.NewScanner(stderr)
	go func() {
		for stderrScanner.Scan() {
			if line := stderrScanner.Text(); line != "stderr" {
				t.Errorf("CmdReader failed, expected %v, got %v", "stderr", line)
			}
		}
	}()
}
