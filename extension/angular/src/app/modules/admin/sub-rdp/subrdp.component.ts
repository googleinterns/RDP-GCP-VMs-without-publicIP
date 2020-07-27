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

import { Output, EventEmitter, Component } from '@angular/core';
import { SubRdpService } from './subrdp.service';
import { MatTableDataSource } from '@angular/material/table';
import { MatDialog } from '@angular/material/dialog';

import { Instance} from 'src/classes';
import { NetworkDialog } from '../../../components/network-dialog/network-dialog.component';

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
    displayedColumns: string[] = ['name', 'zone', 'status', 'port', 'start-rdp', 'end-rdp'];
    dataSource: MatTableDataSource<Instance>;

    @Output() instance = new EventEmitter<Instance>();

    constructor(public dialog: MatDialog, private subRdpService: SubRdpService) {};

    // getComputeInstances gets the current compute instances for the credentials signed into gCloud
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
            instance.zone = instance.zone.split('/').pop();
          })
          console.log(this.instances)
          this.project = '';
          this.getInstancesError = null;
          this.dataSource = new MatTableDataSource<Instance>(this.instances);
        }

      }, error => {
        this.getInstancesError = error.error.error;
      })
    }

    rdp(instance: Instance) {
      if (!instance.rdpRunning) {
        const dialogRef = this.dialog.open(NetworkDialog, {
          width: '400px',
          data: {network: 'default'}
        });
        dialogRef.afterClosed().subscribe(result => {
          if (result) {
            instance.firewallNetwork = result;
            this.instance.emit(instance);
          }
        });
      } else {
        this.instance.emit(instance);
      }
    }
}

