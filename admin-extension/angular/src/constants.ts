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
const rdpShutdownMessage = 'Shutdown private RDP for ';
const rdpFirewallDeletedMessage = 'Deleting firewall for ';

// Need to change in Chrome Extension constants as well.
const errorConnectingToServer = 'Error connecting to server, are you sure the companion server is running?';

const statusUrl = 'https://localhost:23966/health'
const getConfigEndpoint = 'https://localhost:23966/admin/get-config'
const rdpSocketEndpoint = 'wss://localhost:23966/gcloud/start-private-rdp';
const sendOperationEndpoint = 'https://localhost:23966/admin/operation-to-run';
const runOperationSocketEndpoint = 'wss://localhost:23966/admin/run-operation';
const getComputeInstancesEndpoint = 'https://localhost:23966/gcloud/compute-instances';
const sendInstanceOperationEndpoint = 'https://localhost:23966/admin/instance-operation-to-run';
const sendProjectOperationEndpoint = 'https://localhost:23966/admin/get-project';
const runPreRDPOperationsEndpoint = 'https://localhost:23966/admin/run-prerdp';
const verifyTokenEndpoint = 'https://localhost:23966/verifyidtoken'


export {verifyTokenEndpoint, endRdpCmd, loginRdpCmd, endOperationCmd, readyForRdpCommandSocket, rdpFirewallDeletedMessage, rdpShutdownMessage, errorConnectingToServer, rdpSocketEndpoint, statusUrl, getConfigEndpoint, sendOperationEndpoint, runOperationSocketEndpoint, getComputeInstancesEndpoint, sendInstanceOperationEndpoint, sendProjectOperationEndpoint, runPreRDPOperationsEndpoint };
