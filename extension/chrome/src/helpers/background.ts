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
      if (instancesData.error) {
        throw new Error(instancesData.error);
      }
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
    // if (instanceRequest.status === 401) {
    //   throw new Error('gCloud Auth');
    // }
    // if (!instanceRequest.ok) {
    //   throw new Error('server error');
    // }
    return await instanceRequest.json();
  },
};

// computeInstances is used to contain all the compute instances received from the server.
let computeInstances = [] as Instance[];
// rdpInstancesList contains the current instances that are being RDP'ed into.
let rdpInstancesList = [];

let adminTabId;

let getInstancesError;

const createAdminTab = () => {
  chrome.tabs.create({url: chrome.extension.getURL('index.html?#/admin')}, (tab) => {
    adminTabId = tab.id;
  })
}


// adminTabIconClickListener listens for an icon click to open the admin tab.
const adminTabIconClickListener = () => {
  chrome.browserAction.onClicked.addListener((activeTab) => {
    if (!adminTabId) {
      // if no adminTabId set, create a new one
      createAdminTab();
    } else {
      // check if tab is still open, if it is, switch to it, else open new one.
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

const renableRdpButton = (instanceName: string) => {
  for (let i = 0; i < computeInstances.length; i++) {
    if (computeInstances[i].name === instanceName) {
      computeInstances[i].rdpRunning = false;
      break;
    }
  }
}

const getInstancesUsingProjectInUrl = (url: string) => {
  const urlParams = new URLSearchParams(url.split('?')[1]);
  const projectName = urlParams.get('project');

  // If project name is parsed from URL, get all the compute instances for the project.
  if (projectName) {
    instanceFunctions
      .getComputeInstances(projectName)
      .then(instances => {
        getInstancesError = null;
        computeInstances = instances;
        chrome.browserAction.setBadgeText({text: ''})
      })
      .catch(error => {
        computeInstances = [];
        chrome.browserAction.setBadgeText({text: 'error'})
        console.log(error.toString())
        getInstancesError = error.toString();
      });
  }
}

// Tab listener listens for tab changes.
const tabUpdatedListener = () => {
  chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
    // Check if tab that is being updated is created by the extension for RDP
    if (changeInfo.status === 'complete' && rdpInstancesList.filter(element => element.tabId === tabId).length > 0) {
      console.log(tabId)

      // If the tab is being refreshed by the user, delete the tab.
      for (let i = 0; i < rdpInstancesList.length; i++) {
        if (rdpInstancesList[i].tabId === tabId && rdpInstancesList[i].status === 'ready') {
          chrome.tabs.remove(tabId);
          //renableRdpButton(rdpInstancesList[i].instance.name)
          //rdpInstancesList.splice(i, 1);
        }
      }
      
    } else if (
      changeInfo.status === 'complete' &&
      tab.status === 'complete' &&
      tab.url != undefined
    ) {

      // If tab matches Pantheon instances list page.
      if (tab.url.match(pantheonPageRegex) && tab.url.indexOf('?') !== -1) {
        getInstancesUsingProjectInUrl(tab.url);
      }
    }
  });
};

const tabRemovedListener = () => {
  chrome.tabs.onRemoved.addListener((tabId, removed) => {
    if (rdpInstancesList.filter(element => element.tabId === tabId).length > 0) {
      for (let i = 0; i < rdpInstancesList.length; i++) {
        if (rdpInstancesList[i].tabId === tabId && rdpInstancesList[i].status === 'ready') {
          renableRdpButton(rdpInstancesList[i].instance.name)
          rdpInstancesList.splice(i, 1);
        }
      }
    }
  })
}

// Listener that listens for multiple types of messages
const messageListener = () => {
  chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    // Send instances if received get instances from popup
    if (request.type === popupGetInstances) {
      chrome.tabs.query({active: true, lastFocusedWindow: true}, tabs => {
        getInstancesUsingProjectInUrl(tabs[0].url);
      })
      sendResponse({instances: computeInstances, error: getInstancesError});
    } else if (request.type === startPrivateRdp) {

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
    } else if (request.type === rdpGetInstances) {
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
    } else if (request.type === "rdpEnded") {
      for (let i = 0; i < computeInstances.length; i++) {
        if (computeInstances[i].name === request.instance.name) {
          computeInstances[i].rdpRunning = false;
          break;
        }
      }
    }
  });
}

export {enablePopup, tabUpdatedListener, tabRemovedListener, messageListener, instanceFunctions, adminTabIconClickListener};
