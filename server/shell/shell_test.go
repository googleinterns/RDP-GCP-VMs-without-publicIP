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
	"testing"
)

const (
	invalidCmd  string = "foo"
	validCmd string = `echo hello`
	validReaderCmd string = `echo stdout; echo stderr >&2;`
)

// TestCmd tests the Cmd which runs a shell cmd and waits for it output before testing
func TestCmd(t *testing.T) {
	if _, err := Cmd(invalidCmd); err == nil {
		t.Errorf("Cmd didn't error on invalid cmd")
	}

	if validCmd, err := Cmd(validCmd); validCmd == nil || err != nil || bytes.Equal(validCmd, []byte("hello")) {
		t.Errorf("Cmd failed, expected %v, got %v", "hello", string(validCmd))
	}
}

// TestCmdReader tests the CmdReader which outputs stdout/stderr as a ReadCloser
func TestCmdReader(t *testing.T) {
	if invalidCmd, err := CmdReader(invalidCmd); invalidCmd != nil && err == nil {
		t.Errorf("CmdReader didn't error on invalid cmd")
	}

	output, err := CmdReader(validReaderCmd)
	if err != nil {
		t.Errorf("Valid CmdReader cmd error'd out %v", err)
	}

	stdout := output[0]; stderr := output[1]

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