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

import { Component, NgZone } from '@angular/core';
import { Instance, SocketMessage, SocketMessageInterface, SocketCmd } from 'src/classes';
import { readyForRdpCommandSocket, loginRdpCmd, endRdpCmd, rdpShutdownMessage, rdpGetInstances, rdpSocketEndpoint } from 'src/constants';
import { bindCallback, BehaviorSubject, Subscription } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { MatSnackBar } from '@angular/material/snack-bar';

@Component({
  selector: 'app-rdp',
  templateUrl: 'rdp.component.html',
  styleUrls: ['rdp.component.scss']
})


export class RdpComponent {
  rdpInstance: Instance;
  socketStatus = 'Getting instance for private RDP';
  credsReady = false;
  endButtonReady = false;
  socketMessageList = [] as SocketMessage[];
  getInstances: any;
  socket: WebSocketSubject<any> = webSocket(rdpSocketEndpoint);
  username: string;
  password: string;

  constructor(private zone: NgZone, private snackbar: MatSnackBar) {};

  ngOnInit() {
    this.getInstanceFromBackground();
  };


  getInstanceFromBackground() {
    chrome.runtime.sendMessage({type: rdpGetInstances}, (resp) => {
      if (resp.instance) {
        this.zone.run(() => {
          this.rdpInstance = resp.instance;
          this.socketStatus = 'Connecting to service with instance '  + this.rdpInstance.name;
          this.socketConnection();
        });
      }
    });
  }

  loginToRdp() {
    const msg = new SocketCmd()
    msg.cmd = loginRdpCmd;
    msg.username = this.username;
    msg.password = this.password;

    this.username = ''
    this.password = ''

    this.socket.next(msg);
    this.snackbar.open('Sent credentials', '', { duration: 3000 });
  }

  disableButtonsAndInput() {
    this.zone.run(() => {
      this.credsReady = false;
      this.endButtonReady = false;
    });
  }

  endRdp() {
    const msg = new SocketCmd()
    msg.cmd = endRdpCmd;
    msg.name = this.rdpInstance.name;
    this.socket.next(msg);

    this.disableButtonsAndInput();
    this.snackbar.open('Sent end command', '', { duration: 3000 });

    chrome.runtime.sendMessage({type: "rdpEnded", instance: this.rdpInstance});
  }

  connectionClosed() {
    this.zone.run(() => {
      this.socketStatus = 'Connection to service with instance ' + this.rdpInstance.name + ' closed';
    });
    this.disableButtonsAndInput();
    this.snackbar.open('Connection to service was closed', '', { duration: 3000 });
  }

  socketConnection() {
    console.log('starting conn')
    this.socket.next(this.rdpInstance)
    this.socket.subscribe(
      (msg) => {
        // Handle messages from the connection
        console.log('message received: ' + JSON.stringify(msg));

        this.zone.run(() => {
          this.socketStatus = 'Connected to service with instance ' + this.rdpInstance.name;
        });

        const receivedMessage = new SocketMessage(msg as SocketMessageInterface);
        this.socketMessageList.push(receivedMessage);


        if (receivedMessage.message === readyForRdpCommandSocket) {
          this.zone.run(() => {
            this.credsReady = true;
            this.endButtonReady = true;
          });
        }

        if (receivedMessage.message === rdpShutdownMessage) {
          this.disableButtonsAndInput();
        }
      },
      (err) => {
        // Handle error from connection
        console.log(err);
        this.connectionClosed();
      },
      () => {
        // Handle connection closed from server
        this.connectionClosed();
      },
   );
  }

}
