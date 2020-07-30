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

import { Component, Input, Output, EventEmitter } from '@angular/core';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';

import { AdminOperationSocketOutput, AdminOperationInterface, SocketCmd} from 'src/classes';
import { runOperationSocketEndpoint, readyForRdpCommandSocket, endRdpCmd, rdpShutdownMessage, rdpSocketEndpoint, endOperationCmd, rdpFirewallDeletedMessage } from 'src/constants';

@Component({
  selector: 'output',
  templateUrl: 'output.component.html',
  styleUrls: ['output.component.scss']
})

export class OutputComponent {
    messages = [] as AdminOperationSocketOutput[];
    socket: WebSocketSubject<any> = webSocket(runOperationSocketEndpoint);
    rdpSocket: WebSocketSubject<any> = webSocket(rdpSocketEndpoint);
    rdpFirewallDeleted = false;

    @Input() operationToRun: AdminOperationInterface;

    // Close input from the red close button
    @Input() set close(close: boolean) {
        if (close) {
          if (this.operationToRun.type === 'rdp') {
            if (!this.operationToRun.instance.rdpRunning) {
              this.outputClosed.emit(true);
            } else {
              this.endRdp();
            }
            this.outputClosed.emit(true);
          } else {
            this.endOperation();
          }
        }
    }

    // Close input from the end RDP button
    @Input() set closeRdp(end: boolean) {
      if (end) {
        this.endRdp();
      }
    }

    @Output() outputClosed = new EventEmitter<boolean>();

    constructor() {};

    ngOnInit() {
        if (this.operationToRun.type === 'rdp') {
          this.rdpConnection();
        } else {
          this.socketConnection();
        }
    }

    // Send end RDP command and remove port
    endRdp() {
      const msg = new SocketCmd()
      msg.cmd = endRdpCmd;
      msg.name = this.operationToRun.instance.name;
      this.rdpSocket.next(msg);
      this.operationToRun.instance.portRunning = '';
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
            console.log(msg)
            const receivedMessage = msg as AdminOperationSocketOutput;
            msg.message = msg.message.replace(/\s/g, '&#160;');
            msg.stdout = msg.stdout.replace(/\s/g, '&#160;')
            msg.stderr = msg.stderr.replace(/\s/g, '&#160;');
            this.messages.push(receivedMessage);
            console.log(this.messages)
          },
          (err) => {
            // Handle error from connection
            console.log(err);

          },
          () => {
            // Handle connection closed from server
            console.log('closed')
          },
       );
      }


    // rdpConnection handles the websocket connection for private RDP
    rdpConnection() {
        console.log('starting conn')
        this.rdpSocket.next(this.operationToRun.instance)
        this.rdpSocket.subscribe(
          (msg) => {

            console.log(msg)
            // Handle messages from the connection
            this.operationToRun.instance.rdpStatus = 'Connected to server'

            const receivedMessage = msg as AdminOperationSocketOutput;
            this.messages.push(receivedMessage);

            // If started IAP tunnel message, display port in table
            if (receivedMessage.message.includes('Started IAP tunnel for ' + this.operationToRun.instance.name)) {
              const port = receivedMessage.message.split(': ')[1];
              this.operationToRun.instance.portRunning = port;
            }

            // Set status to ready if received ready for RDP message
            if (receivedMessage.message === readyForRdpCommandSocket) {
              this.operationToRun.instance.rdpStatus = 'Ready to RDP';
            }

            if (receivedMessage.message === rdpFirewallDeletedMessage + this.operationToRun.instance.name) {
              this.operationToRun.instance.rdpStatus = 'Firewall has closed, restart RDP if you wish to connect';
              this.rdpFirewallDeleted = true;
              this.operationToRun.instance.portRunning = null;
            }

            // Set instance to not running when shut down message received.
            if (receivedMessage.message === rdpShutdownMessage + this.operationToRun.instance.name) {
              this.operationToRun.instance.rdpStatus = 'Shut down';
              this.operationToRun.instance.rdpRunning = false;
            }

            if (receivedMessage.error) {
              this.operationToRun.instance.rdpError = receivedMessage.error;
              this.operationToRun.instance.rdpStatus = null;
            }

          },
          (err) => {
            // Handle error from connection
            console.log(err);
            if (!this.rdpFirewallDeleted) {
              this.operationToRun.instance.rdpError = 'Server couldn\'t delete firewall rule for ' + this.operationToRun.instance.name + ', please delete manually';
              this.operationToRun.instance.rdpStatus = null;
            } else {
              this.operationToRun.instance.rdpStatus = 'Connection closed';
            }
            this.operationToRun.instance.rdpRunning = false;
            this.operationToRun.instance.portRunning = null;
          },
          () => {
            if (!this.rdpFirewallDeleted) {
              this.operationToRun.instance.rdpError = 'Server couldn\'t delete firewall rule for ' + this.operationToRun.instance.name + ', please delete manually';
              this.operationToRun.instance.rdpStatus = null;
            } else {
              this.operationToRun.instance.rdpStatus = 'Closed from server';
            }
            this.operationToRun.instance.rdpRunning = false;
            this.operationToRun.instance.portRunning = null;
          },
       );
      }
}
