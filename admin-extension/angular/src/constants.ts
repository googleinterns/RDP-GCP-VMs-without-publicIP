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

const endRdpCmd = 'end';
const loginRdpCmd = 'start-rdp';
const endOperationCmd = 'end_operation';

const readyForRdpCommandSocket = 'Ready for command';
const rdpShutdownMessage = 'Shutdown private RDP for %v'

// Need to change in Chrome Extension constants as well.
const errorConnectingToServer = 'Error connecting to server, are you sure the companion server is running?';

const statusUrl = 'http://localhost:23966/health'
const getConfigEndpoint = 'http://localhost:23966/admin/get-config'
const rdpSocketEndpoint = 'ws://localhost:23966/gcloud/start-private-rdp';
const sendOperationEndpoint = 'http://localhost:23966/admin/command-to-run';
const runOperationSocketEndpoint = 'ws://localhost:23966/admin/run-operation';
const getComputeInstancesEndpoint = 'http://localhost:23966/gcloud/compute-instances';
const sendInstanceOperationEndpoint = 'http://localhost:23966/admin/instance-operation-to-run';

export {endRdpCmd, loginRdpCmd, endOperationCmd, readyForRdpCommandSocket, rdpShutdownMessage, errorConnectingToServer, rdpSocketEndpoint, statusUrl, getConfigEndpoint, sendOperationEndpoint, runOperationSocketEndpoint, getComputeInstancesEndpoint, sendInstanceOperationEndpoint};
