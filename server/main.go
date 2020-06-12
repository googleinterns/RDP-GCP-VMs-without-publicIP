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
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// main defines the routes of the HTTP server and starts listening on port 23966
func main() {
	router := mux.NewRouter()
	router.HandleFunc("/health", Health).Methods("GET")
	log.Fatal(http.ListenAndServe(":23966", router))
}

// Health is a HTTP route that prints a simple string to check if the server is running.
func Health(w http.ResponseWriter, _ *http.Request) {
	type response struct {
		Status string `json:"status"`
	}

	resp := response{Status: "server is running"}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
