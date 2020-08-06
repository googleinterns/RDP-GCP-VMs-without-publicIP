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

import { Output, EventEmitter, Component, Input } from '@angular/core';
import { InstancesService } from './instances.service';
import { MatTableDataSource } from '@angular/material/table';

import { Instance, ConfigAdminOperationInterface } from 'src/classes';

@Component({
  selector: 'app-instances',
  templateUrl: 'instances.component.html',
  providers: [InstancesService],
  styleUrls: ['instances.component.scss']
})

export class InstancesComponent {
  displayedColumns: string[] = ['name', 'zone', 'networkIp', 'port', 'rdp-button', 'operations'];
  dataSource: MatTableDataSource<Instance>;
  operationError: string;
  loadedOperation: any;

  @Input() instanceOperations: ConfigAdminOperationInterface[];
  @Input() instances: Instance[];
  @Input() commonParameters: any[];

  @Output() instance = new EventEmitter<Instance>();
  @Output() startOperation = new EventEmitter<any>();

  constructor(private instancesService: InstancesService) { };

  ngOnInit() {
    console.log(this.instances)
    this.dataSource = new MatTableDataSource(this.instances)
  }


  rdp(instance: Instance) {
    this.instance.emit(instance);
  }

  // loadCommonParams adds the commonParams to a variables object.
  loadCommonParams(variables: any) {
    this.commonParameters.forEach((commonParam) => {
      variables[commonParam.name] = commonParam.value
    })
  }

  startInstanceOperation(instance: Instance, instanceOperation: ConfigAdminOperationInterface) {
    console.log(instance)
    console.log(instanceOperation)
    const data = { name: instanceOperation.name, instance, variables: {} };
    this.loadCommonParams(data.variables);

    this.instancesService.sendOperation(data)
      .subscribe((response: any) => {
        console.log(response)

        // If operation returned, set loadedOperation to response
        if (response.operation) {
          this.operationError = '';
          this.loadedOperation = response;
          this.loadedOperation.name = instanceOperation.name;
          this.loadedOperation.instanceName = instance.name;
        }

        // If error returned, set operation.error to error
        if (response.error) {
          this.operationError = response.error;
        }

      }, error => {
        this.operationError = error;
      })
  }

  clearLoadedOperation() {
    this.loadedOperation = null;
  }

  startLoadedOperation(operation: any) {
    this.startOperation.emit(operation);
    this.clearLoadedOperation();
  }
}

