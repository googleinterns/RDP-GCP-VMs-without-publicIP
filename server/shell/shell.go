package shell

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Cmd runs a shell command and waits for its output before returning the output
func Cmd(cmd string) (string, error) {
	parsedCmd := strings.Fields(cmd)

	out, err := exec.Command(parsedCmd[0], parsedCmd[1:]...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// CmdReader runs a shell command and pipes the stdout and stderr into ReadClosers
func CmdReader(cmd string) ([]io.ReadCloser, error) {
	parsedCmd := strings.Fields(cmd)
	asyncCmd := exec.Command(parsedCmd[0], parsedCmd[1:]...)

	stdout, err := asyncCmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := asyncCmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err = asyncCmd.Start(); err != nil {
		return nil, err
	}

	fmt.Println("Stand by to read..")
	return []io.ReadCloser{stdout, stderr}, nil
}
