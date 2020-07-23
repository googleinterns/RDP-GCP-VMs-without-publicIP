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
  selector: 'sub-rdp',
  templateUrl: 'subrdp.component.html',
  styleUrls: ['subrdp.component.scss']
})

export class SubRdpComponent {

    constructor(private zone: NgZone) {};

    ngOnInit() {

    }
}
