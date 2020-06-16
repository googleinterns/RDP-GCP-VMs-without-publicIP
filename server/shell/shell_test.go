package shell

import (
	"reflect"
	"strings"
	"testing"
)

const (
	invalidShell string = "`"
	invalidSyncCmd  string = "foo"
	validSyncCmd string = `echo hello`
)

func TestParseCmd(t *testing.T) {
	if invalidShell, err := parseCmd(invalidSyncCmd); reflect.DeepEqual(invalidShell, []string{""}) && err == nil {
		t.Errorf("SynchronousCmd didn't error on invalid cmd")
	}

	if validCmd, err := parseCmd(validSyncCmd); !reflect.DeepEqual(validCmd, strings.Fields(validSyncCmd)) || err != nil {
		t.Errorf("SynchronousCmd failed, expected %v, got %v", strings.Fields(validSyncCmd), validCmd)
	}
}

func TestSynchronousCmd(t *testing.T) {
	if invalidCmd, err := SynchronousCmd(invalidSyncCmd); invalidCmd != "" && err == nil {
		t.Errorf("SynchronousCmd didn't error on invalid cmd")
	}

	if validCmd, err := SynchronousCmd(validSyncCmd); validCmd == "" || err != nil {
		t.Errorf("SynchronousCmd failed, expected %v, got %v", "hello", err)
	}
}
