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

import { AdminRoutingModule } from './app/modules/admin/admin-routing.module';

// File used to contain classes and interfaces used in the extension.

interface GuestOsFeature {
  type: string;
}

interface Disk {
  guestOsFeatures: GuestOsFeature[];
}

interface NetworkInterface {
  name: string;
  network: string;
  networkIP: string;
}

class Instance {
  networkInterfaces: NetworkInterface[];
  description: string;
  disks: Disk[];
  id: string;
  name: string;
  status: string;
  zone: string;
  displayPrivateRdpDom: boolean;
  project: string;
  rdpRunning: boolean;
  portRunning: string;
  rdpStatus: string;
  rdpError: string;
  firewallNetwork: string;
}

interface SocketMessageInterface {
  message: string;
  error: string;
}

class SocketMessage implements SocketMessageInterface {
  constructor(socketMessage: SocketMessageInterface) {
    this.message = socketMessage.message;
    this.error = socketMessage.error;
  }

  message: string;
  error: string;
}

class SocketCmd {
  cmd: string;
  name: string;
  username: string;
  password: string;
  hash: string;
}

interface ConfigParamInterface {
  default: string;
  type: string;
  optional: boolean;
  description: string;
  sample: string
  choices: string[];
  dependencies: Map<string, string>;
}

interface ConfigAdminOperationInterface {
  name: string;
  operation: string;
  params: Map<string, ConfigParamInterface>;
  description: string;
}

interface ConfigInterface {
  instance_operations: ConfigAdminOperationInterface[];
  operations: ConfigAdminOperationInterface[];
  common_params: Map<string, ConfigParamInterface>;
  pre_rdp_operations: string[];
  project_operation: string;
}

interface AdminOperationInterface {
  type: string;
  instance: Instance;
  operation: string;
  hash: string;
  status: string;
}

interface AdminOperationSocketOutput {
  message: string;
  stdout: string;
  stderr: string;
  error: string;
}

class Config implements ConfigInterface {
  constructor(config: ConfigInterface) {
    this.instance_operations = config.instance_operations;
    this.operations = config.operations;
    this.common_params = config.common_params;
    this.pre_rdp_operations = config.pre_rdp_operations;
    this.project_operation = config.project_operation;
  }

  instance_operations: ConfigAdminOperationInterface[];
  operations: ConfigAdminOperationInterface[];
  common_params = new Map<string, ConfigParamInterface>();
  pre_rdp_operations: string[];
  project_operation: string;
}

const canDisplayRdpDom = (instance: Instance) => {
  for (let i = 0; i < instance.disks.length; i++) {
    for (let j = 0; j < instance.disks[i].guestOsFeatures.length; j++) {
      if (instance.disks[i].guestOsFeatures[j].type === 'WINDOWS') {
        return true;
      }
    }
  }
  return false;
}

export {Instance, canDisplayRdpDom, SocketMessageInterface, SocketMessage, SocketCmd, AdminOperationInterface, Config, ConfigInterface, ConfigParamInterface, AdminOperationSocketOutput, ConfigAdminOperationInterface};
