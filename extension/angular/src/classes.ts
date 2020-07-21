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
  NetworkInterfaces: NetworkInterface[];
  description: string;
  disks: Disk[];
  id: string;
  name: string;
  status: string;
  zone: string;
  displayPrivateRdpDom: boolean;
  project: string;
  rdpRunning: boolean;
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
}

interface ConfigParamInterface {
  default: string;
  type: string;
  optional: boolean;
  description: string;
  sample: string
  choices: string[];
}

interface ConfigAdminOperationInterface {
  name: string;
  operation: string;
  params: Map<string, ConfigParamInterface>;
  description: string;
}

interface ConfigInterface {
  operations: ConfigAdminOperationInterface[];
  common_params: Map<string, ConfigParamInterface>;
  enable_rdp: boolean;
}

class Config implements ConfigInterface {
  constructor(config: ConfigInterface) {
    this.operations = config.operations;
    this.common_params = config.common_params;
    this.enable_rdp = config.enable_rdp;
  }

  operations: ConfigAdminOperationInterface[];
  common_params = new Map<string, ConfigParamInterface>();
  enable_rdp: boolean;
}

export {Instance, SocketMessageInterface, SocketMessage, SocketCmd, Config, ConfigInterface, ConfigParamInterface};
