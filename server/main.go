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

// Package main implements a simple HTTP server that will take requests
// from a Chrome Extension which is used for administration on GCP.
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/gcloud"
	"github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/shell"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const (
	allowedOrigin  string = "chrome-extension://ljejdkiepkafbpnbacemjjcleckglnjl"
	allowedMethods string = "POST, GET, OPTIONS"
	allowedHeaders string = "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization"
)

// main defines the routes of the HTTP server and starts listening on port 23966
func main() {
	router := mux.NewRouter()
	router.HandleFunc("/health", health).Methods("GET")
	router.HandleFunc("/health", setCorsHeaders).Methods("OPTIONS")
	router.HandleFunc("/gcloud/compute-instances", getComputeInstances).Methods("POST")
	router.HandleFunc("/gcloud/compute-instances", setCorsHeaders).Methods("OPTIONS")
	router.HandleFunc("/gcloud/start-private-rdp", startPrivateRdp)

	log.Fatal(http.ListenAndServe(":23966", router))
}

// setCorsHeaders is used to set the headers for CORS requests from the Chrome Extension.
// All preflight requests are handled by this function and it is also used in the HTTP functions.
func setCorsHeaders(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
	w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
}

// health is a HTTP route that prints a simple string to check if the server is running.
func health(w http.ResponseWriter, _ *http.Request) {
	type response struct {
		Status string `json:"status"`
	}

	resp := response{Status: "server is running"}

	w.Header().Set("Content-Type", "application/json")
	setCorsHeaders(w, nil)
	json.NewEncoder(w).Encode(resp)
}

// getComputeInstances gets the current compute instances for the project passed in.
func getComputeInstances(w http.ResponseWriter, r *http.Request) {
	setCorsHeaders(w, nil)
	type request struct {
		ProjectName string `json:"project"`
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var reqBody request
	if err := json.Unmarshal(body, &reqBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shell := &shell.CmdShell{}
	gcloudExecutor := gcloud.NewGcloudExecutor(shell)

	instances, err := gcloudExecutor.GetComputeInstances(reqBody.ProjectName)
	if err != nil {
		log.Println(err)
		switch err.Error() {
		case gcloud.SdkAuthError:
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(gcloud.SdkAuthError))
		case gcloud.SdkProjectError:
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(gcloud.SdkProjectError))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instances)
}

func startPrivateRdp(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}

	// To-do, make sure origin for websocket is only chrome extension
	upgrader.CheckOrigin = func(_ *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("Starting RDP socket connection")
	defer ws.Close()

	shell := &shell.CmdShell{}
	gcloudExecutor := gcloud.NewGcloudExecutor(shell)
	gcloudExecutor.StartPrivateRdp(ws)

}
