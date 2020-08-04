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

import { Component } from '@angular/core';
import { canDisplayRdpDom, Config, ConfigInterface, Instance } from 'src/classes';
import { errorConnectingToServer } from 'src/constants';
import { MatSnackBar } from '@angular/material/snack-bar';
import { AdminService } from './admin.service';
import { ResizeEvent } from 'angular-resizable-element';

@Component({
  selector: 'app-admin',
  templateUrl: 'admin.component.html',
  providers: [AdminService],
  styleUrls: ['admin.component.scss']
})


export class AdminComponent {
  config: Config;
  configError: string;
  commonParams = [];
  operations = [];
  loading = true;
  operationsRunning = [];
  outputTabIndex: number;
  instanceToUpdate: Instance;
  style: {};
  useCommonParameters: boolean;
  project: string;
  projectToValidate: string;
  instancesLoading: boolean;
  instances = [] as Instance[];
  getProjectError: string;
  initializeInstances = false;
  preRdpError: string;

  constructor(private snackbar: MatSnackBar, private adminService: AdminService) {};

  ngOnInit() {
   this.loadConfig();
  };

  // onResizeEnd(event: ResizeEvent): void {
  //   this.style = {
  //     position: 'fixed',
  //     top: `${event.rectangle.top}px`,
  //     height: `${event.rectangle.height}px`,
  //   };
  // }

  // setCommonParams sets up a commonParams array consisting of name-value pairs using the configuration common params.
  setCommonParams() {
    if (this.config.common_params) {
      this.useCommonParameters = true;
      for (const [name, paramValue] of Object.entries(this.config.common_params)) {
        paramValue.name = name;
        paramValue.value = paramValue.default;
        this.commonParams.push(paramValue);
      }
    } else {
      this.useCommonParameters = false;
    }
  }

  // setOperations sets up an operations array using config.operations
  setOperations() {
    if (this.config.operations) {
      this.config.operations.forEach((operation) => {
        const params = [];

        if (operation.params) {
          for (const [name, paramValue] of Object.entries(operation.params)) {
            paramValue.value = paramValue.default;
            paramValue.name = name;
            params.push(paramValue);
          }
        }
        this.operations.push({name: operation.name, description: operation.description, params});
    })

    console.log(this.operations)
    }
  }

  getProject() {
    const data = {type: "", project_name: "", variables: {}};
    if (this.useCommonParameters) {
      data.type = "get";
    } else {
      data.type = "validate";
      data.project_name = this.projectToValidate;
    }

    this.loadCommonParams(data.variables)

    this.adminService.sendProjectOperation(data)
    .subscribe((response: any) => {
      console.log(response)

      // If operation returned, set loadedOperation to response
      if (response.project) {
        this.project = response.project;
        console.log(this.project)
        this.getComputeInstances();
      }

      // If error returned, set operation.error to error
      if (response.error) {
        this.getProjectError = response.error;
      }

    }, error => {
      if (error.status === 0) {
        this.getProjectError  = errorConnectingToServer;
      } else {
        this.getProjectError  = error.error.error;
      }
    })
  }

  getComputeInstances() {
    this.instancesLoading = true;
    const data = {project: this.project}
    this.adminService.getComputeInstances(data)
    .subscribe((response: any) => {
      console.log(response)

      // If error returned, set getInstancesError to error
      if (response.error) {
        this.getProjectError = response.error;
        this.instances = [];
      } else {
        let instances = response;
        instances.forEach((instance) => {
          instance.project = this.project;
          instance.zone = instance.zone.split('/').pop();
          instance.networkInterfaces[0].network = instance.networkInterfaces[0].network.split('/').pop();
          instance.displayPrivateRdpDom = canDisplayRdpDom(instance);
        })
        this.instances = instances;
        console.log(this.instances);
        if (instances.length > 0) {
          this.initializeInstances = true;
        }
        this.getProjectError = null;
        this.instancesLoading = false;

      }

    }, error => {
      if (error.status === 0) {
        this.getProjectError = errorConnectingToServer;
        this.instancesLoading = false;
      } else {
        this.getProjectError = error.error.error;
        this.instancesLoading = false;
      }
    })
  }


  // loadCommonParams adds the commonParams to a variables object.
  loadCommonParams(variables: any) {
    this.commonParams.forEach((commonParam) => {
      variables[commonParam.name] = commonParam.value
    })
  }

  // sendOperation sends the operation and its params to the server to get an ready operation.
  sendOperation(operation: any) {
    let variables = {};
    operation.params.forEach(param => {
      variables[param.name] = param.value;
    });
    const data = {name: operation.name, variables}

    this.loadCommonParams(data.variables)

    this.adminService.sendOperation(data)
    .subscribe((response: any) => {
      console.log(response)

      // If operation returned, set loadedOperation to response
      if (response.operation) {
        operation.error = '';
        operation.loadedOperation = response;
        console.log(operation)
      }

      // If error returned, set operation.error to error
      if (response.error) {
        operation.error = response.error;
      }

    }, error => {
      if (error.status === 0) {
        operation.error = errorConnectingToServer;
      } else {
        operation.error = error.error.error;
      }
    })
  }

  // clearLoadedOperation is triggered by the clear button, clears the operation
  clearLoadedOperation(operation: any) {
    operation.loadedOperation = null;
  }

  // startLoadedOperation will start the loadedOperation.
  startLoadedOperation(operation: any) {
    const operationFull = operation.name;
    operation.loadedOperation.label = operationFull.substr(0,20-1)+(operationFull.length>20?'...':'');
    this.operationsRunning.push(operation.loadedOperation)

    this.snackbar.open('Started operation', '', { duration: 3000 });

    console.log(this.operationsRunning)
    operation.loadedOperation = null;
  }

  // startLoadedInstanceOperation will start an instance operation from the subrdp component.
  startLoadedInstanceOperation(operation: any) {
    const operationFull = operation.operation;
    operation.label = operationFull.substr(0,20-1)+(operationFull.length>20?'...':'');
    this.operationsRunning.push(operation)
    this.snackbar.open('Started instance operation', '', { duration: 3000 });
  }

  // loadConfig loads the config file from the server and sets up all the variables needed to render page.
  loadConfig() {
    this.adminService.getConfig()
    .subscribe((response: any) => {
      console.log(response)
      if (response.error) {
        this.configError = response.error;
      } else {
        this.config = new Config(response as ConfigInterface)
        this.setCommonParams();
        this.setOperations();

        console.log(this.config)

        // If no operations defined and rdp not enabled, set a configError
        if (!this.config) {
          this.configError = 'Your configuration file is empty.'
        }
      }


    }, error => {
      console.log(error)
      console.log(error.status)
      if (error.status === 0) {
        this.configError = errorConnectingToServer;
      } else {
        this.configError = JSON.stringify(error);
      }
    })

    this.loading = false;
  }

  // refreshConfig is trigerred by the refresh icon in the top bar, resets all the variables and reloadsConfig
  refreshConfig() {
    this.config = null;
    this.operations = [];
    this.configError = null;
    this.commonParams = [];
    this.loading = true;
    this.operationsRunning = [];
    this.initializeInstances = false;
    this.preRdpError = null;
    this.getProjectError = null;
    this.loadConfig();
  }

  // closeOutputTab sends a close message to the tab running an operation to close the websocket.
  closeOutputTab() {
    console.log('close button')
    console.log(this.operationsRunning[this.outputTabIndex])
    this.operationsRunning[this.outputTabIndex].close = true;
  }

  // tabChanged changes outputTabIndex to the current tab index
  tabChanged(tabChangeEvent: any): void {
    this.outputTabIndex = tabChangeEvent.index;
  }

  // outputTabClosed removes the ended operation from the list of tabs
  outputTabClosed(i: number) {
    this.operationsRunning.splice(i, 1);
    this.snackbar.open('Operation ended', '', { duration: 3000 });
  }

  // instanceEmitted handles cases from the subrdp component including starting and ending the RDP websocket connection.
  instanceEmitted(instance: Instance) {
    if (!instance.rdpRunning) {
      if (this.config.pre_rdp_operations) {
        const data = {type: "pre_rdp", project_name: "", variables: {}};
        this.loadCommonParams(data.variables)
        this.adminService.runPreRDPOperations(data)
        .subscribe((response: any) => {
          console.log(response)
    
          // If error returned, set getInstancesError to error
          if (response.error) {
            this.preRdpError = response.error;
          } else {
            this.preRdpError = null;
            instance.rdpRunning = true;
            const operation = {type: 'rdp', label: 'RDP '+instance.name, instance}
            this.operationsRunning.push(operation);
          }
    
        }, error => {
          if (error.status === 0) {
            this.preRdpError = errorConnectingToServer;
          } else {
            this.preRdpError = error.error.error;
          }
        })
      }

    } else {
      this.operationsRunning.forEach((operation => {
        if (operation.instance === instance) {
          operation.rdpClose = true;

          this.snackbar.open('Ending RDP for '+instance.name, '', { duration: 3000 });
        }
      }))
    }
  }
}
