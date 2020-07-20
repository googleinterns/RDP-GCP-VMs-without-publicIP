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

import {pantheonInstancesListRegex, pantheonPageRegex, getComputeInstancesEndpoint, popupGetInstances, startPrivateRdp, rdpGetInstances} from './constants';
import {Instance, InstanceInterface} from '../classes';

// Enable chrome extension popup on matching hosts.
const enablePopup = (hosts: string[]) => {
  chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
    if (hosts.some(host => tab.url.includes(host))) {
      chrome.browserAction.setPopup({
        tabId: tabId,
        popup: 'index.html?#/popup',
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
        instances.push(new Instance(<InstanceInterface>instancesData[i], projectName));
      }

      return instances;
    } catch (error) {
      throw error;
    }
  },
  getInstancesApi: async (projectName: string): Promise<any> => {
    const instanceRequest = await fetch(
      getComputeInstancesEndpoint,
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

// computeInstances is used to contain all the compute instances received from the server.
let computeInstances = [] as Instance[];
// rdpInstancesList contains the current instances that are being RDP'ed into.
let rdpInstancesList = [];

let adminTabId;

const createAdminTab = () => {
  chrome.tabs.create({url: chrome.extension.getURL('index.html?#/admin')}, (tab) => {
    adminTabId = tab.id;
  })
}

const adminTabIconClickListener = () => {
  chrome.browserAction.onClicked.addListener((activeTab) => {
    if (!adminTabId) {
      createAdminTab();
    } else {
      chrome.tabs.get(adminTabId, () => {
        if (chrome.runtime.lastError) {
          createAdminTab();
        } else {
          chrome.tabs.update(adminTabId, {highlighted: true});
        }
      })
    }
  })
}

// Tab listener listens for tab changes.
const tabListener = () => {
  chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
    // Check if tab that is being updated is created by the extension for RDP
    if (changeInfo.status === 'complete' && rdpInstancesList.filter(element => element.tabId === tabId).length > 0) {
      console.log(tabId)

      // If the tab is being refreshed by the user, delete the tab.
      for (let i = 0; i < rdpInstancesList.length; i++) {
        if (rdpInstancesList[i].tabId === tabId && rdpInstancesList[i].status === 'ready') {
          rdpInstancesList.splice(i, 1);
          chrome.tabs.remove(tabId);
        }
      }
      
    } else if (
      changeInfo.status === 'complete' &&
      tab.status === 'complete' &&
      tab.url != undefined
    ) {

      // If tab matches Pantheon instances list page.
      if (tab.url.match(pantheonInstancesListRegex) && tab.url.indexOf('?') !== -1) {
        const urlParams = new URLSearchParams(tab.url.split('?')[1]);
        const projectName = urlParams.get('project');

        // If project name is parsed from URL, get all the compute instances for the project.
        if (projectName) {
          instanceFunctions
            .getComputeInstances(projectName)
            .then(instances => {
              computeInstances = instances;

              chrome.tabs.sendMessage(tabId, { type: 'get-compute-instances', computeInstances })
            })
            .catch(error => chrome.browserAction.setBadgeText({text: 'error'}));
        }
      }
    }
  });
};

// Listener that listens for multiple types of messages
const messageListener = () => {
  chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    // Send instances if received get instances from popup
    if (request.type == popupGetInstances) {
      sendResponse({instances: computeInstances, projectName: computeInstances[0].project});
    } else if (request.type == startPrivateRdp) {

      let instanceToRdp;
      // set rdpRunning to true for instances that sent start request
      for (let i = 0; i < computeInstances.length; i++) {
        if (computeInstances[i].name === request.instance.name) {
          computeInstances[i].rdpRunning = true;
          instanceToRdp = computeInstances[i];
          break;
        }
      }
      sendResponse({instances: computeInstances});

      // open new tab for RDP, add the instance and its tab id to keep track.
      chrome.tabs.create({url: chrome.extension.getURL('index.html?#/rdp')}, (tab) => {
        rdpInstancesList.push({instance: instanceToRdp, tabId: tab.id, status: 'created'});
      });
    } else if (request.type == rdpGetInstances) {
      console.log(rdpInstancesList)
      let instance;
      
      // Set rdp instance status to ready, RDP page in new tab has been created and is now starting connection.
      for (let i = 0; i < rdpInstancesList.length; i++) {
        if (rdpInstancesList[i].tabId === sender.tab.id) {
          rdpInstancesList[i].status = 'ready';
          instance = rdpInstancesList[i].instance;
        }
      }

      if (instance) {
        sendResponse({instance});
      }
    }
  });
}

export {enablePopup, tabListener, messageListener, instanceFunctions, adminTabIconClickListener};
