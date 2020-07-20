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
import { Instance, SocketMessage, SocketMessageInterface, SocketCmd, Config, ConfigInterface, ConfigParamInterface } from 'src/classes';
import { readyForRdpCommandSocket, loginRdpCmd, endRdpCmd, rdpShutdownMessage, rdpGetInstances, rdpSocketEndpoint, sendOperationEndpoint } from 'src/constants';
import { bindCallback, BehaviorSubject, Subscription } from 'rxjs';
import {AdminService} from './admin.service';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { MatSnackBar } from '@angular/material/snack-bar';

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
  constructor(private zone: NgZone, private snackbar: MatSnackBar, private adminService: AdminService) {};

  ngOnInit() {
   this.loadConfig();
  };

  setCommonParams() {
    for (const [name, paramValue] of Object.entries(this.config.common_params)) {
      paramValue.name = name;
      paramValue.value = paramValue.default;
      this.commonParams.push(paramValue);
    }
  }

  setOperations() {
    if (this.config.operations) {
      this.config.operations.forEach((operation) => {
        let paramsToSet = {};
        let paramsToLoad = [];
        for (const [name, paramValue] of Object.entries(operation.params)) {
          paramsToSet[name] = paramValue.default;
          paramValue.name = name;
          paramsToLoad.push(paramValue);
        }
        this.operations.push({name: operation.name, description: operation.description, paramsToSet, paramsToLoad});
    })

    console.log(this.operations)
    }
  }

  loadCommonParams(variables: any) {
    this.commonParams.forEach((commonParam) => {
      variables[commonParam.name] = commonParam.value
    })
  }

  sendOperation(operation: any) {
    const data = {name: operation.name, variables: operation.paramsToSet}
    console.log(this.commonParams)
    this.loadCommonParams(data.variables)
    this.adminService.sendOperation(data)
    .subscribe((response: any) => {
      console.log(response)

      if (response.operation) {
        operation.error = "";
        operation.loadedOperation = response;
        console.log(operation)
      }

      if (response.error) {
        operation.error = response.error;
      }

    }, error => {
      operation.error = error;
    })
  }

  clearLoadedOperation(operation: any) {
    operation.loadedOperation = null;
  }

  startLoadedOperation(operation: any) {
    Object.keys(operation.paramsToSet).forEach(function(param) {
      operation.paramsToSet[param] = null
  });
    operation.loadedOperation = null;
  }

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
        if (!this.config.operations && !this.config.enable_rdp) {
          this.configError = "Your configuration file is empty."
        }
      }

      this.loading = false;
    }, error => {
      console.log(error)
      this.configError = error;
    })
  }

  refreshConfig() {
    this.config = null;
    this.operations = [];
    this.configError = null;
    this.commonParams = [];
    this.loading = true;
    this.loadConfig();
  }
}
