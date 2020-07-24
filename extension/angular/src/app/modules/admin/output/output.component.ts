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

import { Component, NgZone, Input, Output, EventEmitter } from '@angular/core';
import { AdminOperationSocketOutput, AdminOperationInterface, SocketCmd, Instance} from 'src/classes';
import { runOperationSocketEndpoint, readyForRdpCommandSocket, loginRdpCmd, endRdpCmd, rdpShutdownMessage, rdpGetInstances, rdpSocketEndpoint, endOperationCmd } from 'src/constants';
import { bindCallback, BehaviorSubject, Subscription } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MAT_HAMMER_OPTIONS } from '@angular/material/core';

@Component({
  selector: 'output',
  templateUrl: 'output.component.html',
  styleUrls: ['output.component.scss']
})

export class OutputComponent {
    messages = [] as AdminOperationSocketOutput[];
    socket: WebSocketSubject<any> = webSocket(runOperationSocketEndpoint);
    rdpSocket: WebSocketSubject<any> = webSocket(rdpSocketEndpoint);

    @Input() operationToRun: AdminOperationInterface;
    @Input() set close(close: boolean) {
        if (close) {
          if (this.operationToRun.type === 'rdp') {
            this.endRdp();
          } else {
            this.endOperation();
          }
        }
    }
    @Output() outputClosed = new EventEmitter<boolean>();
    @Output() instanceUpdate = new EventEmitter<Instance>();

    constructor(private zone: NgZone) {};

    ngOnInit() {
        if (this.operationToRun.type === 'rdp') {
          this.rdpConnection();
        } else {
          this.socketConnection();
        }
    }

    endRdp() {
      const msg = new SocketCmd()
      msg.cmd = endRdpCmd;
      msg.name = this.operationToRun.instance.name;
      this.rdpSocket.next(msg);
      this.operationToRun.instance.portRunning = "";
    }

    // endOperation sends an end command to websocket
    endOperation() {
        const msg = new SocketCmd()
        msg.cmd = endOperationCmd;
        msg.hash = this.operationToRun.hash;
        console.log(msg)
        this.socket.next(msg);
        this.outputClosed.emit(true);
    }

    // socketConnection manages the socket connection
    socketConnection() {
        console.log('starting conn')
        this.socket.next(this.operationToRun)
        this.socket.subscribe(
          (msg) => {
            // Handle messages from the connection
            const receivedMessage = msg as AdminOperationSocketOutput;
            this.messages.push(receivedMessage);
          },
          (err) => {
            // Handle error from connection
            console.log(err);
            
          },
          () => {
            // Handle connection closed from server
            console.log("closed")
          },
       );
      }

    rdpConnection() {
        console.log('starting conn')
        this.rdpSocket.next(this.operationToRun.instance)
        this.rdpSocket.subscribe(
          (msg) => {
            // Handle messages from the connection
            console.log('message received: ' + JSON.stringify(msg));
    
            this.operationToRun.instance.rdpStatus = "Connected"
    
            const receivedMessage = msg as AdminOperationSocketOutput;
            this.messages.push(receivedMessage);
    
            if (receivedMessage.message.includes("Started IAP tunnel for " + this.operationToRun.instance.name)) {
              const port = receivedMessage.message.split(": ")[1];
              console.log("Yoo port")
              console.log(port)
              this.operationToRun.instance.portRunning = port;
            }

            if (receivedMessage.message === readyForRdpCommandSocket) {
              this.operationToRun.instance.rdpStatus = "Ready";
            }
    
            if (receivedMessage.message === rdpShutdownMessage) {
              this.operationToRun.instance.rdpStatus = "Shut down";
              this.outputClosed.emit(true);
            }
          },
          (err) => {
            // Handle error from connection
            console.log(err);
            this.operationToRun.instance.rdpStatus = "Closed";
            this.outputClosed.emit(true);
          },
          () => {
            // Handle connection closed from server
            this.operationToRun.instance.rdpStatus = "Closed from server";
            this.outputClosed.emit(true);
          },
       );
      }
}
