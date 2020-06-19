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

import { pantheonPageRegex, pantheonInstancesListRegex } from "./constants";
import { Instance, InstanceInterface } from "../classes";

const enablePopup = (hosts: string[]) => {
    chrome.tabs.onUpdated.addListener(function(tabId, changeInfo, tab) {
        if (hosts.some(host => tab.url.includes(host))) {
            chrome.browserAction.setPopup({
                tabId: tabId,
                popup: 'popup.html'
            });
        }
    });
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

const getInstancesApi = async (projectName: string): Promise<any> => {
    const instanceRequest = await fetch("http://localhost:23966/compute-instances", {
        method: 'POST',
        mode: 'cors',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ project: projectName })
    });
    if (instanceRequest.status === 401) {
        throw new Error("gCloud Auth");
    }
    if (!instanceRequest.ok) {
        throw new Error("server error");
    }
    return await instanceRequest.json();
}

const getComputeInstances = async (projectName: string): Promise<Instance[]> => {
    try {
        const instancesData = await getInstancesApi(projectName);
        const instances = [] as Array<Instance>
        for (let i = 0; i < instancesData.length; i++) {
            console.log(<InstanceInterface> instancesData[i]);
            console.log(new Instance(<InstanceInterface> instancesData[i]))
            instances.push(new Instance(<InstanceInterface> instancesData[i]));
        }

        return instances;
    } catch (error) {
        throw error;
    }
}

let computeInstances = [] as Instance[];
const pantheonListener = () => {
    chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
        if (changeInfo.status === "complete" && tab.status === "complete" && tab.url) {
            chrome.tabs.query({lastFocusedWindow: true, active: true}, (tabs) => {
                if (tabs[0].url === tab.url && tab.url.match(pantheonPageRegex) && tab.url.indexOf("?") !== -1) {
                    const urlParams = new URLSearchParams(tab.url.split("?")[1]);
                    const projectName = urlParams.get("project");
                    console.log(projectName)
                    if (projectName) {
                        getComputeInstances(projectName).then((instances) => {
                            computeInstances = instances;
                            console.log(computeInstances)
                        }).catch(error => chrome.browserAction.setBadgeText({text: "error"}));
                    }
                }
            })
        }
    })
}



export { enablePopup, pantheonListener };
