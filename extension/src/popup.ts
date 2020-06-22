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

/* A package that is the script used in the pop up of the extension */

const getServerStatus = async (): Promise<string> => {
  const statusRequest = await fetch('http://localhost:23966/health', {
    method: 'GET',
    mode: 'cors',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  const response = await statusRequest.json();
  return response['status'];
};

chrome.tabs.query({active: true, currentWindow: true}, async () => {
  const container = document.getElementById('span-status');
  container.innerText = 'connecting to server';
  try {
    container.innerText = await getServerStatus();
  } catch (error) {
    container.innerText = error;
  }
});
