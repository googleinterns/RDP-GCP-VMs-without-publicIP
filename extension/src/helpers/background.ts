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

/* A file that contains functions used in the background script */

import {pantheonPageRegex} from './constants';
import {Instance, InstanceInterface} from '../classes';

// Enable chrome extension popup on matching hosts.
const enablePopup = (hosts: string[]) => {
  chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
    if (hosts.some(host => tab.url.includes(host))) {
      chrome.browserAction.setPopup({
        tabId: tabId,
        popup: 'popup.html',
      });
    }
  });
};

// Object that contains functions used to get instances, needed for unit testing.
const instanceFunctions = {
  getComputeInstances: async function (
    projectName: string
  ): Promise<Instance[]> {
    try {
      const instancesData = await this.getInstancesApi(projectName);
      const instances = [] as Array<Instance>;
      for (let i = 0; i < instancesData.length; i++) {
        instances.push(new Instance(<InstanceInterface>instancesData[i]));
      }

      return instances;
    } catch (error) {
      throw error;
    }
  },
  getInstancesApi: async (projectName: string): Promise<any> => {
    const instanceRequest = await fetch(
      'http://localhost:23966/compute-instances',
      {
        method: 'POST',
        mode: 'cors',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({project: projectName}),
      }
    );
    if (instanceRequest.status === 401) {
      throw new Error('gCloud Auth');
    }
    if (!instanceRequest.ok) {
      throw new Error('server error');
    }
    return await instanceRequest.json();
  },
};

let computeInstances = [] as Instance[];
// Pantheon listener listens for pantheon pages and gets the GCP Compute instances to display buttons.
const pantheonListener = () => {
  chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
    if (
      changeInfo.status === 'complete' &&
      tab.status === 'complete' &&
      tab.url != undefined
    ) {
      if (tab.url.match(pantheonPageRegex) && tab.url.indexOf('?') !== -1) {
        const urlParams = new URLSearchParams(tab.url.split('?')[1]);
        const projectName = urlParams.get('project');
        console.log(projectName);
        if (projectName) {
          instanceFunctions
            .getComputeInstances(projectName)
            .then(instances => {
              computeInstances = instances;
              console.log(computeInstances);
            })
            .catch(error => chrome.browserAction.setBadgeText({text: 'error'}));
        }
      }
    }
  });
};

export {enablePopup, pantheonListener, instanceFunctions};
