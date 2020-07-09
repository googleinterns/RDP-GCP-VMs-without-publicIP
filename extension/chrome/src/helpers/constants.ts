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

// A file that contains constants for the extension.

const pantheonInstancesListRegex = /.*pantheon.corp.google.com\/compute\/.*/;
const pantheonPageRegex = /.*pantheon.corp.google.com\/.*/;

const serverUrl = "http://localhost:23966";
const getComputeInstancesEndpoint = serverUrl + "/gcloud/compute-instances";

// Need to change in Angular constants as well.
const popupGetInstances = 'popup-get-instances';
const startPrivateRdp = 'start-private-rdp';
const rdpGetInstances = 'rdp-get-instance';

export {pantheonInstancesListRegex, pantheonPageRegex, getComputeInstancesEndpoint, popupGetInstances, startPrivateRdp, rdpGetInstances};
