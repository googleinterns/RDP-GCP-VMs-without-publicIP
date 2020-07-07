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
import {Instance} from "./classes";

const observer = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
        if (!mutation.addedNodes) {
            return;
        }
        for (let i = 0; i < mutation.addedNodes.length; i++) {
            const mutationNode = mutation.addedNodes[i] as HTMLElement;
            if (mutationNode != undefined && mutationNode.tagName.toLowerCase().includes("ng-view")) {
                renderRdpButtons();
            }
        }
    })
})


let computeInstances = [] as Instance[];
console.log("content script running 3");
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.type == 'get-compute-instances') {
        computeInstances = message.computeInstances;
        if (document.querySelector("ng-view")) {
            renderRdpButtons();
        } else {
            observer.observe(document.body, {
                childList: true,
                subtree: true
            });
        }
    }
});

const renderRdpButtons = () => {
    const table = document.querySelector("table");
    const rows = Array.from(table.querySelectorAll("tr"));
    const headings = rows[0];
    let nameIndex;
    console.log("rendering rdp")
    for (let i = 0; i < headings.children.length; i++) {
        const child = headings.children[i] as HTMLElement;
        if (child.innerText.toLowerCase() === "name") {
            nameIndex = i;
            break;
        }
    }

    for (let i = 1; i < rows.length; i++) {
        const nameElement = rows[i].children[nameIndex] as HTMLElement;
        console.log(nameElement.innerText)
        const instance = computeInstances.filter(instance => instance.name === nameElement.innerText.trim())[0];
        console.log(instance);
        createRdpButton(instance, rows[i]);
    }
}

const createRdpButton = (instance: Instance, element: HTMLElement) => {
    if (instance.displayPrivateRdpDom) {
        let customButton = document.createElement("button");
        customButton.innerText = "PRIVATE RDP";
        customButton.id = "private-rdp-button";
        customButton.type = "button";
        customButton.classList.add("pure-material-button-contained");
        customButton.onclick = () => {
            chrome.runtime.sendMessage({type: 'start-private-rdp', instance}, (_) => {
                customButton.disabled = true;
            })
        }

        if (instance.rdpRunning) {
            customButton.disabled = true;
        }

        element.appendChild(customButton);
    }
}
