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
import { readyForRdpCommandSocket, loginRdpCmd, endRdpCmd, rdpShutdownMessage } from 'src/constants';
import { bindCallback, BehaviorSubject, Subscription } from 'rxjs';
import { RdpService } from './rdp.service';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';

@Component({
  selector: 'app-rdp',
  templateUrl: 'rdp.component.html',
  providers: [RdpService],
  styleUrls: ['rdp.component.scss']
})


export class RdpComponent {
  rdpInstance: Instance;
  socketStatus = "Getting instance for private RDP";
  credsReady = false;
  endButtonReady = false;
  socketMessageList = [] as SocketMessage[];
  getInstances: any;
  socket: WebSocketSubject<any> = webSocket('ws://localhost:23966/gcloud/start-private-rdp');
  username: string;
  password: string;

  constructor(private zone: NgZone, private rdpService: RdpService) {};

  ngOnInit() {
    this.getInstanceFromBackground();
  };


  getInstanceFromBackground() {
    chrome.runtime.sendMessage({type: 'rdp-get-instance'}, (resp) => {
      if (resp.instance) {
        this.zone.run(() => {
          this.rdpInstance = resp.instance;
          this.socketStatus = "Connecting to service with instance "  + this.rdpInstance['name'];
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

    this.socket.next(msg);
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
  }

  socketConnection() {
    console.log("starting conn")
    this.socket.next(this.rdpInstance)
    this.socket.subscribe(
      msg => {
        console.log('message received: ' + JSON.stringify(msg));
        const receivedMessage = new SocketMessage(<SocketMessageInterface> msg);
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

      // Called whenever there is a message from the server
      err => this.disableButtonsAndInput(),
      // Called if WebSocket API signals some kind of error
      () => this.disableButtonsAndInput()
      // Called when connection is closed (for whatever reason)
   );
    //this.rdpService.connect();
    //this.rdpService.messages.subscribe(msg => {console.log(msg)})
  }

}
