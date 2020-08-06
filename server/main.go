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
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/admin"
	"github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/gcloud"
	"github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/shell"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

const (
	configNotLoaded string = "Unable to load configuration file from server, try refreshing the page."
)

var (
	allowedOrigins = []string{"chrome-extension://aibhgfeeenaelgkgefjmlmdiehldgekn"}
	configPath *string
	// loadedConfig points to the config currently in use.
	loadedConfig *admin.Config
	// operationPool keeps track of all the custom commands that are setup and running
	operationPool []admin.OperationToRun
)

type errorRequest struct {
	Error string `json:"error"`
}

func newErrorRequest(err error) errorRequest {
	return errorRequest{Error: err.Error()}
}

// main defines the routes of the HTTP server and starts listening on port 23966
func main() {
	configPath = flag.String("path", ".", "Path of the config file")
	enableLogs := flag.Bool("v", false, "Enable logging")
	flag.Parse()

	if !*enableLogs {
		log.SetOutput(ioutil.Discard)
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", health).Methods("GET")
	router.HandleFunc("/gcloud/compute-instances", getComputeInstances).Methods("POST")
	router.HandleFunc("/gcloud/start-private-rdp", startPrivateRdp)
	router.HandleFunc("/admin/get-config", getConfigFileAndSendJson).Methods("GET")
	router.HandleFunc("/admin/get-project", getProjectFromParameters).Methods("POST")
	router.HandleFunc("/admin/run-prerdp", runPreRDPOperations).Methods("POST")
	router.HandleFunc("/admin/operation-to-run", validateAdminOperationParams).Methods("POST")
	router.HandleFunc("/admin/instance-operation-to-run", validateInstanceOperationParams).Methods("POST")
	router.HandleFunc("/admin/run-operation", runAdminOperation)
	
	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowCredentials: true,
	})

	handler := c.Handler(router)	
	log.Println("AdminOPs server has started on port 23966")
	log.Fatal(http.ListenAndServe(":23966", handler))
}

// health is a HTTP route that prints a simple string to check if the server is running.
func health(w http.ResponseWriter, _ *http.Request) {
	type response struct {
		Status string `json:"status"`
	}

	resp := response{Status: "server is running"}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// getConfigFileAndSendJson calls the functions to load the config file and set loadedConfig to it
func getConfigFileAndSendJson(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type request struct {
		Error string `json:"error"`
	}

	config, err := admin.LoadConfig(configPath)
	if err != nil {
		var req request
		req.Error = err.Error()
		json.NewEncoder(w).Encode(req)

		return
	}

	loadedConfig = config
	json.NewEncoder(w).Encode(config)
	return
}

func getProjectFromParameters(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ProjectName string `json:"project"`
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	var reqBody admin.ProjectOperationParams
	if err := json.Unmarshal(body, &reqBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if loadedConfig == nil {
		json.NewEncoder(w).Encode(newErrorRequest(errors.New(configNotLoaded)))
		return
	}

	if loadedConfig.ValidateProjectOperation == "" && reqBody.Type == "validate" {
		json.NewEncoder(w).Encode(response{ProjectName: reqBody.ProjectName})
		return
	}

	if reqBody.Type == "validate" && reqBody.ProjectName == "" {
		json.NewEncoder(w).Encode(newErrorRequest(errors.New("Project missing for validation")))
		return
	}

	var operation string

	if reqBody.Type == "validate" {
		operation, _, err = admin.ReadOperationFromCommonParams(reqBody, loadedConfig.ValidateProjectOperation, loadedConfig)
	} else {
		operation, _, err = admin.ReadOperationFromCommonParams(reqBody, loadedConfig.ProjectOperation, loadedConfig)
	}

	if err != nil {
		json.NewEncoder(w).Encode(newErrorRequest(err))
		return
	}

	shell := &shell.CmdShell{}
	output, err := shell.ExecuteCmd(operation)

	if err != nil{
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(newErrorRequest(errors.New(string(output))))
	}

	if (reqBody.Type == "validate") {
		if (strings.Contains(string(output), reqBody.ProjectName)) {
			json.NewEncoder(w).Encode(response{ProjectName: reqBody.ProjectName})
		} else {
			json.NewEncoder(w).Encode(newErrorRequest(fmt.Errorf("Project %s not found in validateProjectOperation output", reqBody.ProjectName)))
		}
		return
	}

	json.NewEncoder(w).Encode(response{ProjectName: strings.TrimSuffix(string(output), "\n")})
}

// validateAdminOperationParams reads requests from the server that fill in the command's variables
func validateAdminOperationParams(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	var reqBody admin.OperationToFill
	if err := json.Unmarshal(body, &reqBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if loadedConfig == nil {
		json.NewEncoder(w).Encode(newErrorRequest(errors.New(configNotLoaded)))
		return
	}

	operationReady, err := admin.ReadAdminOperation(reqBody, loadedConfig)
	if err != nil {
		json.NewEncoder(w).Encode(newErrorRequest(err))
		return
	}

	operationPool = append(operationPool, operationReady)

	json.NewEncoder(w).Encode(operationReady)
	return
}

func validateInstanceOperationParams(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	var reqBody admin.InstanceOperationToFill
	if err := json.Unmarshal(body, &reqBody); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if loadedConfig == nil {
		json.NewEncoder(w).Encode(newErrorRequest(errors.New(configNotLoaded)))
		return
	}

	operationReady, err := admin.ReadInstanceOperation(reqBody, loadedConfig)
	if err != nil {
		json.NewEncoder(w).Encode(newErrorRequest(err))
		return
	}

	operationPool = append(operationPool, operationReady)

	json.NewEncoder(w).Encode(operationReady)
	return
}

func runPreRDPOperations(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	var reqBody admin.ProjectOperationParams
	if err := json.Unmarshal(body, &reqBody); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if loadedConfig == nil {
		json.NewEncoder(w).Encode(newErrorRequest(errors.New(configNotLoaded)))
		return
	}

	shell := &shell.CmdShell{}


	for _, operation := range loadedConfig.PreRDPOperations {
		runOperation := false

		filledOperation, variables, err := admin.ReadOperationFromCommonParams(reqBody, operation.Operation, loadedConfig)
		if err != nil {
			json.NewEncoder(w).Encode(newErrorRequest(err))
			return
		}

		for dependency, value := range operation.Dependencies {
			dependency = strings.ToUpper(dependency)
			if variables[dependency] == value {
				runOperation = true
			}
		}

		if (runOperation) {
	
			output, _ := shell.ExecuteCmd(filledOperation)
			log.Println(string(output))
		}
	}

	type response struct {
		Status string `json:"status"`
	}

	json.NewEncoder(w).Encode(response{Status: "ready"})
}

func runAdminOperation(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	upgrader.CheckOrigin = func(r *http.Request) bool { 
		if origin := r.Header.Get("Origin"); origin != "" {
			log.Println(origin)
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == origin {
					return true
				}
			}
		}
		return false
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("Starting operation socket connection")
	defer ws.Close()

	operationToRun, err := admin.ReadOperationHashFromConn(ws, &operationPool)
	if err != nil {
		admin.WriteToSocket(ws, "", "", "", err)
	}

	shell := &shell.CmdShell{}
	adminExecutor := admin.NewAdminExecutor(shell)
	adminExecutor.RunOperation(ws, operationToRun)

	// Remove finished operation from pool
	for i, operation := range operationPool {
		if operation == *operationToRun {
			operationPool = append(operationPool[:i], operationPool[i+1:]...)
			break
		}
	}
}

// getComputeInstances gets the current compute instances for the project passed in.
func getComputeInstances(w http.ResponseWriter, r *http.Request) {
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
		case gcloud.SdkProjectError:
			w.WriteHeader(http.StatusUnauthorized)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
		json.NewEncoder(w).Encode(newErrorRequest(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instances)
}

func startPrivateRdp(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	upgrader.CheckOrigin = func(r *http.Request) bool { 
		if origin := r.Header.Get("Origin"); origin != "" {
			log.Println(origin)
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == origin {
					return true
				}
			}
		}
		return false
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("Starting RDP socket connection")
	defer ws.Close()

	shell := &shell.CmdShell{}
	gcloudExecutor := gcloud.NewGcloudExecutor(shell)
	gcloudExecutor.StartPrivateRdp(ws)
}
