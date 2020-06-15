package shell

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"

	"github.com/mattn/go-shellwords"
)

func parseCmd(cmd string) []string {
	p := shellwords.NewParser()
	args, err := p.Parse(cmd)
	if err != nil {
		log.Println(err)
	}

	return args
}

// SynchronousCmd runs a shell command and waits for its output before returning
func SynchronousCmd(cmd string) string {
	parsedCmd := parseCmd(cmd)

	out, err := exec.Command(parsedCmd[0], parsedCmd[1:]...).CombinedOutput()
	if err != nil {
		log.Println(err)
	}
	return string(out)
}

// AsynchronousCmd runs a shell command in the background
func AsynchronousCmd(cmd string) {
	parsedCmd := parseCmd(cmd)

	asyncCmd := exec.Command(parsedCmd[0], parsedCmd[1:]...)
	stdout, err := asyncCmd.StdoutPipe()
	if err != nil {
		log.Println(err)
	}

	asyncCmd.Stderr = asyncCmd.Stdout

	if err = asyncCmd.Start(); err != nil {
		log.Println(err)
	}

	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
		}
	}()

	fmt.Println("Stand by to read..")
}
