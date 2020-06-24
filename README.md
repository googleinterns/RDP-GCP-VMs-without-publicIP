
# Project Template

This repository is a Google open source project that owns a chrome extension
to allow RDP to GCP VMs that do not have public IP. This is not an officially
supported Google product. 


## Source Code Headers

Every file containing source code must include copyright and license
information. This includes any JS/CSS files that you might be serving out to
browsers. (This is to help well-intentioned people avoid accidental copying that
doesn't comply with the license.)

Apache header:

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


## How to get started with the Chrome Extension  
The Chrome Extension is written in TypeScript and uses Angular for pages.

To get started with the extension:  

Switch to the extension's directory  
`cd extensions`  
Use npm to install the dependencies  
`npm install`  
The extension requires a extension key in the environment variables, please set a key before building the extension  
`export EXTENSION_DEV_KEY = "your key"`  
To run the tests, cd into the extensions directory and use the `test` command  
`npm test`  
To start the bundler  
`npm start`  
Then load the `dist` folder as an unpacked extension in `chrome://extensions`  

NOTE: Whenever an edit is made to any of the `.ts` or `.html` files, the extension will automatically reload  

## How to run the Go companion server
To run the Go server, cd into the server directory and run `go run main.go`  

To test the Go server and its packages, run `go test ./...`
