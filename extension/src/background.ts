import { doubleNumber } from "./double_number";
let number = 1;
function polling() {
    number = doubleNumber(number)
    console.log('polling 2 ' + number);
    setTimeout(polling, 1000 * 5);
}

polling()

chrome.runtime.onInstalled.addListener(function() {
    chrome.declarativeContent.onPageChanged.removeRules(undefined, function() {
      chrome.declarativeContent.onPageChanged.addRules([{
        conditions: [new chrome.declarativeContent.PageStateMatcher({
          pageUrl: {hostContains: 'pantheon.corp.google.com'},
        })
        ],
            actions: [new chrome.declarativeContent.ShowPageAction()]
      }]);
    });
});