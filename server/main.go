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
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/admin"
	"github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/gcloud"
	"github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server/shell"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

const (
	projectContextTimeout time.Duration = 20 * time.Second
	configNotLoaded       string        = "Unable to load configuration file from server, try refreshing the page."
	authError             string        = "Error authorizing user using Google oAuth, reason: %s. \n The extension uses the account signed in to Chrome to authenticate."
)

var (
	allowedOrigins = []string{"chrome-extension://oanplklbjoeneghmjkkodflcgkhggldm"}
	configPath     *string
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

var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

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
	router.HandleFunc("/verifyidtoken", verifyIdToken).Methods("POST")
	router.HandleFunc("/gcloud/compute-instances", sessionMiddleware(getComputeInstances)).Methods("POST")
	router.HandleFunc("/gcloud/start-private-rdp", sessionMiddleware(startPrivateRdp))
	router.HandleFunc("/admin/get-config", sessionMiddleware(getConfigFileAndSendJson)).Methods("GET")
	router.HandleFunc("/admin/get-project", sessionMiddleware(getProjectFromParameters)).Methods("POST")
	router.HandleFunc("/admin/operation-to-run", sessionMiddleware(validateAdminOperationParams)).Methods("POST")
	router.HandleFunc("/admin/instance-operation-to-run", sessionMiddleware(validateInstanceOperationParams)).Methods("POST")
	router.HandleFunc("/admin/run-operation", sessionMiddleware(runAdminOperation))

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
	})

	handler := c.Handler(router)
	log.Println("AdminOPs server has started on port 23966")
	log.Fatal(http.ListenAndServeTLS(":23966", "localhost.pem", "localhost-key.pem", handler))
}

func sessionMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "adminops")
		if err != nil {
			json.NewEncoder(w).Encode(newErrorRequest(errors.New("auth error")))
			return
		}

		if val, inSession := session.Values["auth"]; !inSession || val != true {
			json.NewEncoder(w).Encode(newErrorRequest(errors.New("auth error")))
			return
		}

		h(w, r)
	}
}

// health is a HTTP route that prints a simple string to check if the server is running.
func health(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Status string `json:"status"`
	}

	resp := response{Status: "server is running"}

	session, _ := store.Get(r, "adminops")
	session.Values["auth"] = true
	session.Save(r, w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func verifyIdToken(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Token string `json:"token"`
	}

	type token struct {
		Email string `json:"email"`
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	var reqBody request
	if err := json.Unmarshal(body, &reqBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var tokenInfo token

	resp, err := http.Get(fmt.Sprintf("https://www.googleapis.com/oauth2/v3/tokeninfo?access_token=%s", reqBody.Token))
	if err != nil {
		json.NewEncoder(w).Encode(newErrorRequest(fmt.Errorf(authError, "Error verifying access token")))
		return
	}

	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		json.NewEncoder(w).Encode(newErrorRequest(fmt.Errorf(authError, "Error verifying access token")))
		return
	}

	json.Unmarshal(body, &tokenInfo)

	if tokenInfo.Email == "" || tokenInfo == (token{}) {
		json.NewEncoder(w).Encode(newErrorRequest(fmt.Errorf(authError, "Didn't receive valid email in verification from Google")))
		return
	}

	if !strings.Contains(tokenInfo.Email, "@google.com") {
		json.NewEncoder(w).Encode(newErrorRequest(fmt.Errorf(authError, "You need a @google.com email to use this application")))
		return
	}

	session, _ := store.Get(r, "adminops")
	session.Options = &sessions.Options{SameSite: http.SameSiteNoneMode, Secure: true}
	session.Values["auth"] = true
	session.Save(r, w)
	return
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
	ctx, _ := context.WithTimeout(context.Background(), projectContextTimeout)

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

	log.Println(fmt.Sprintf("Server running project command: %s ", operation))
	shell := &shell.CmdShell{}
	output, err := shell.ExecuteCmdWithContext(ctx, operation)
	log.Println(fmt.Sprintf("Server received output: %s ", string(output)))

	if err != nil {
		errOutput := fmt.Errorf("Error: %s, Output: %s", err.Error(), string(output))
		json.NewEncoder(w).Encode(newErrorRequest(errOutput))
		return
	}

	if reqBody.Type == "validate" {
		if strings.Contains(string(output), reqBody.ProjectName) {
			json.NewEncoder(w).Encode(response{ProjectName: reqBody.ProjectName})
		} else {
			json.NewEncoder(w).Encode(newErrorRequest(fmt.Errorf("Project %s not found in validateProjectOperation output", reqBody.ProjectName)))
		}
		return
	}

	projectOutput := string(output)

	if (loadedConfig.ProjectOperationRegex) != "" {
		r := regexp.MustCompile(loadedConfig.ProjectOperationRegex)
		match := r.FindStringSubmatch(projectOutput)
		if len(match) > 1 {
			projectOutput = match[1]
		}
	}

	json.NewEncoder(w).Encode(response{ProjectName: strings.TrimSuffix(projectOutput, "\n")})
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

	var configuredAdminOperation admin.ConfigAdminOperation

	for _, configCommand := range loadedConfig.InstanceOperations {
		if configCommand.Name == reqBody.Name {
			configuredAdminOperation = configCommand
			break
		}
	}

	if configuredAdminOperation.Name == "" {
		json.NewEncoder(w).Encode(newErrorRequest(fmt.Errorf("%s operation was not found in the config", reqBody.Name)))
		return
	}

	operationReady, err := admin.ReadInstanceOperation(reqBody, configuredAdminOperation)
	if err != nil {
		json.NewEncoder(w).Encode(newErrorRequest(err))
		return
	}

	operationPool = append(operationPool, operationReady)

	json.NewEncoder(w).Encode(operationReady)
	return
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

	gcloudExecutor.StartPrivateRdp(ws, loadedConfig)
}
