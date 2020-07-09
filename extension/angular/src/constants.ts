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

const readyForRdpCommandSocket = 'Ready for command';
const rdpShutdownMessage = 'Shutdown private RDP for %v'

// Need to change in Chrome Extension constants as well.
const popupGetInstances = 'popup-get-instances';
const startPrivateRdp = 'start-private-rdp';
const rdpGetInstances = 'rdp-get-instance';

export {endRdpCmd, loginRdpCmd, readyForRdpCommandSocket, rdpShutdownMessage, popupGetInstances, startPrivateRdp, rdpGetInstances};
