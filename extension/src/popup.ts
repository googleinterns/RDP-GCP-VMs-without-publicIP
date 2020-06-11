import { doubleNumber } from "./double_number";

chrome.tabs.query({'active': true,'currentWindow':true}, () => {
    const container = document.getElementById("container");
    container.innerText = doubleNumber(6).toString();
});
