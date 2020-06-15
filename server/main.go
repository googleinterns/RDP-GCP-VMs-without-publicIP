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
	"fmt"
	"log"
	"net/http"

	"github.com/googleinterns/RDP-GCP-VMs-without-publicIP/shell"
	"github.com/gorilla/mux"
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

	fmt.Println(string(shell.RunSimpleCmd("ls")))

	resp := response{Status: "server is running"}

	w.Header().Set("Content-Type", "application/json")
	setCorsHeaders(w, nil)
	json.NewEncoder(w).Encode(resp)
}
