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

package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// TestHealth tests the /health HTTP response with a GET Request using the health function
func TestHealth(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(health)

	handler.ServeHTTP(rr, req)

	reqBody, err := ioutil.ReadAll(rr.Body)
	if err != nil {
		t.Fatal(err)
	}

	expectedResp := map[string]string{
		"status": "server is running",
	}

	var gotResp map[string]string
	err = json.Unmarshal(reqBody, &gotResp)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(gotResp, expectedResp) {
		t.Errorf("HEALTH failed, got: %v, expected: %v", gotResp, expectedResp)
	}
}

// TestCorsHeaders tests the /health HTTP CORS Headers with a OPTIONS Request using the setCorsHeaders function
func TestCorsHeaders(t *testing.T) {
	req, err := http.NewRequest("OPTIONS", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(setCorsHeaders)

	handler.ServeHTTP(rr, req)

	if gotOrigin, expectedOrigin := rr.Header().Get("Access-Control-Allow-Origin"), "chrome-extension://ljejdkiepkafbpnbacemjjcleckglnjl"; gotOrigin != expectedOrigin {
		t.Errorf("CORSRESPONSE origin failed, got: %v, want: %v", gotOrigin, expectedOrigin)
	}

	if gotMethods, expectedMethods := rr.Header().Get("Access-Control-Allow-Methods"), "POST, GET, OPTIONS"; gotMethods != expectedMethods {
		t.Errorf("CORSRESPONSE origin failed, got: %v, want: %v", gotMethods, expectedMethods)
	}

	if gotHeaders, expectedHeaders := rr.Header().Get("Access-Control-Allow-Headers"), "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization"; gotHeaders != expectedHeaders {
		t.Errorf("CORSRESPONSE origin failed, got: %v, want: %v", gotHeaders, expectedHeaders)
	}
}
