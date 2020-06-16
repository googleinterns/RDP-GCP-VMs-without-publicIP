package shell

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/mattn/go-shellwords"
)

// parseCmd parses a single string and splits it into shell arguments, "ls -l" -> ["ls, -l"]
func parseCmd(cmd string) ([]string, error) {
	p := shellwords.NewParser()
	args, err := p.Parse(cmd)
	if err != nil {
		return nil, err
	}

	return args, nil
}

// SynchronousCmd runs a shell command and waits for its output before returning the output
func SynchronousCmd(cmd string) (string, error) {
	parsedCmd, err := parseCmd(cmd)
	if err != nil {
		return "", err
	}

	out, err := exec.Command(parsedCmd[0], parsedCmd[1:]...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// AsynchronousCmd runs a shell command and pipes the stdout and stderr into a ReadCloser
func AsynchronousCmd(cmd string) ([]io.ReadCloser, error) {
	parsedCmd, err := parseCmd(cmd)
	if err != nil {
		return nil, err
	}

	asyncCmd := exec.Command(parsedCmd[0], parsedCmd[1:]...)

	stdout, err := asyncCmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := asyncCmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	//asyncCmd.Stderr = asyncCmd.Stdout

	if err = asyncCmd.Start(); err != nil {
		return nil, err
	}

	fmt.Println("Stand by to read..")
	return []io.ReadCloser{stdout, stderr}, nil
}
