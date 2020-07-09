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

import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, Subject } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { switchAll, catchError, tap } from 'rxjs/operators';

@Injectable()
export class RdpService {
    private socket: WebSocketSubject<any>;
    private messageSubject = new Subject();
    messages = this.messageSubject.pipe(switchAll(), catchError(e => { throw e }));
    
    connect() {
        if (!this.socket || this.socket.closed) {
            this.socket = webSocket('ws://localhost:23966/gcloud/start-private-rdp');
            const messages = this.socket.pipe(
                tap({
                    error: e => console.log(e),
                })
            );
            this.messageSubject.next(messages)
        }
    }

    sendMessage(msg: any) {
        this.socket.next(msg);
    }

    close() {
        this.socket.complete();
    }

}