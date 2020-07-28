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

/* A script that runs in the background when the extension is initialized */

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

adminTabIconClickListener();