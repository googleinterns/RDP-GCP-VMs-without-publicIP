package sockets

import (
	"bufio"
	"fmt"
	"github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/shell"
	"github.com/gorilla/websocket"
	"log"
)

func PrintAsync(cmd string) {
	stdout, err := shell.AsynchronousCmd(cmd)
	if err != nil {
		log.Println(err)
		return
	}
	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
		}
	}()
}

func SendAsyncCmd(ws *websocket.Conn, cmd string) {
	stdout, err := shell.AsynchronousCmd(cmd)
	if err != nil {
		log.Println(err)
		ws.Close()
		return
	}

	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
			if err := ws.WriteMessage(websocket.TextMessage, scanner.Bytes()); err != nil {
				log.Println(err)
				ws.Close()
				break
			}
		}
	}()
}
