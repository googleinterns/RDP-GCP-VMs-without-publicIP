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
import { Instance, AdminOperationInterface, SocketCmd} from 'src/classes';
import { runOperationSocketEndpoint, loginRdpCmd, endRdpCmd, rdpShutdownMessage, rdpGetInstances, rdpSocketEndpoint, endOperationCmd } from 'src/constants';
import { bindCallback, BehaviorSubject, Subscription } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { MatSnackBar } from '@angular/material/snack-bar';
import { SubRdpService } from './subrdp.service';
//import {MatPaginator} from '@angular/material/paginator';
import {MatTableDataSource} from '@angular/material/table';

@Component({
  selector: 'sub-rdp',
  templateUrl: 'subrdp.component.html',
  providers: [SubRdpService],
  styleUrls: ['subrdp.component.scss']
})

export class SubRdpComponent {
    project: string;
    getInstancesError: string;
    instances: Instance[];
    displayedColumns: string[] = ['name', 'status', 'port', 'start-rdp', 'end-rdp'];
    dataSource: MatTableDataSource<Instance>;

    @Input() set updateInstance(updateInstance: Instance) {
      console.log("haha")
      // if (updateInstance) {
      //   console.log("received updateinstance")
      //   console.log(updateInstance)
      //   this.instances.forEach((instance) => {
      //     if (instance.id === updateInstance.id) {
      //       instance = updateInstance;
      //     }
      //   });
      //   this.dataSource = new MatTableDataSource<Instance>(this.instances);
      // }
    }

    @Output() instance = new EventEmitter<Instance>();

    constructor(private zone: NgZone, private subRdpService: SubRdpService) {};

    ngOnInit() {

    }

    getComputeInstances() {
      const data = {project: this.project}
      this.subRdpService.getComputeInstances(data)
      .subscribe((response: any) => {
        console.log(response)

        // If error returned, set getInstancesError to error
        if (response.error) {
          this.getInstancesError = response.error;
        } else {
          this.instances = response;
          this.instances.forEach((instance) => {
            instance.project = this.project;
          })
          console.log(this.instances)
          this.project = "";
          this.dataSource = new MatTableDataSource<Instance>(this.instances);
        }
  
      }, error => {
        this.getInstancesError = error;
      })
    }

    rdp(instance: Instance) {
      this.instance.emit(instance);
    }
}
