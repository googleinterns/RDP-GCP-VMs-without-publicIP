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
import { AdminOperationSocketOutput, AdminOperationInterface, SocketCmd} from 'src/classes';
import { runOperationSocketEndpoint, loginRdpCmd, endRdpCmd, rdpShutdownMessage, rdpGetInstances, rdpSocketEndpoint, endOperationCmd } from 'src/constants';
import { bindCallback, BehaviorSubject, Subscription } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { MatSnackBar } from '@angular/material/snack-bar';

@Component({
  selector: 'output',
  templateUrl: 'output.component.html',
  styleUrls: ['output.component.scss']
})

export class OutputComponent {
    messages = [] as AdminOperationSocketOutput[];
    socket: WebSocketSubject<any> = webSocket(runOperationSocketEndpoint);

    @Input() operationToRun: AdminOperationInterface;
    @Input() set close(close: boolean) {
        if (close) {
            this.endOperation();
        }
    }
    @Output() outputClosed = new EventEmitter<boolean>();

    constructor(private zone: NgZone) {};

    ngOnInit() {
        this.socketConnection();
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
}
