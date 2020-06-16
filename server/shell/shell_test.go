package shell

import (
	"bufio"
	"reflect"
	"strings"
	"testing"
)

const (
	invalidShell string = "`"
	invalidCmd  string = "foo"
	validSyncCmd string = `echo hello`
	validAsyncCmd string = `echo stdout; echo stderr >&2;`
)

// TestParseCmd tests the parseCmd function which turns strings into shell arguments to pass into os.exec
func TestParseCmd(t *testing.T) {
	if invalidShell, err := parseCmd(invalidCmd); reflect.DeepEqual(invalidShell, []string{""}) && err == nil {
		t.Errorf("SynchronousCmd didn't error on invalid cmd")
	}

	if validCmd, err := parseCmd(validSyncCmd); !reflect.DeepEqual(validCmd, strings.Fields(validSyncCmd)) || err != nil {
		t.Errorf("SynchronousCmd failed, expected %v, got %v", strings.Fields(validSyncCmd), validCmd)
	}
}

// TestSynchronousCmd tests the SynchronousCmd which runs a shell cmd and waits for it output before testing
func TestSynchronousCmd(t *testing.T) {
	if invalidSyncCmd, err := SynchronousCmd(invalidCmd); invalidSyncCmd != "" && err == nil {
		t.Errorf("SynchronousCmd didn't error on invalid cmd")
	}

	if validCmd, err := SynchronousCmd(validSyncCmd); validCmd == "" || err != nil {
		t.Errorf("SynchronousCmd failed, expected %v, got %v", "hello", err)
	}
}

// TestAsynchronousCmd tests the AsynchronousCmd which outputs stdout/stderr as a ReadCloser
func TestAsynchronousCmd(t *testing.T) {
	if invalidAsyncCmd, err := AsynchronousCmd(invalidCmd); invalidAsyncCmd != nil && err == nil {
		t.Errorf("SynchronousCmd didn't error on invalid cmd")
	}

	output, err := AsynchronousCmd(validAsyncCmd)
	if err != nil {
		t.Errorf("Valid async cmd error'd out %v", err)
	}

	stdout := output[0]; stderr := output[1]

	stdoutScanner := bufio.NewScanner(stdout)
	go func() {
		for stdoutScanner.Scan() {
			if line := stdoutScanner.Text(); line != "stdout" {
				t.Errorf("AsyncCmd failed, expected %v, got %v", "stdout", line)
			}
		}
	}()

	stderrScanner := bufio.NewScanner(stderr)
	go func() {
		for stderrScanner.Scan() {
			if line := stderrScanner.Text(); line != "stderr" {
				t.Errorf("AsyncCmd failed, expected %v, got %v", "stderr", line)
			}
		}
	}()
}
