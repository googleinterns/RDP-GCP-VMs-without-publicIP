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

/* A file that contains a simple function that doubles the number passed in.
Used solely for testing purposes and will be removed in the future. */

const enablePopup = (hosts: string[]) => {
    chrome.runtime.onInstalled.addListener(function() {
        chrome.declarativeContent.onPageChanged.removeRules(undefined, function() {
            for (let i = 0; i < hosts.length; i++) {
                chrome.declarativeContent.onPageChanged.addRules([{
                    conditions: [new chrome.declarativeContent.PageStateMatcher({
                        pageUrl: {hostContains: hosts[i]},
                    })
                    ],
                    actions: [new chrome.declarativeContent.ShowPageAction()]
                }]);
            }
        });
    });
}

const getServerStatus = async (projectName: string): Promise<any> => {
    const statusRequest = await fetch("http://localhost:23966/compute-instances", {
        method: 'POST',
        mode: 'cors',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ project: projectName })
    });

    return await statusRequest.json();
}

const getComputeInstances = (pages: string[]) => {
    chrome.tabs.query({currentWindow: true, active: true}, (tabs) => {
        const tab = tabs[0];

    })
}

export { enablePopup }
