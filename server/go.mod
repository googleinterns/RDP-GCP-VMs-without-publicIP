// Copyright 2020 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file is the module file for this package. It contains the module's path
// as well as the dependencies this module will use.

module github.com/googleinterns/RDP-GCP-VMs-without-publicIP/server

go 1.14

require (
	github.com/Wing924/shellwords v1.0.0
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/sessions v1.2.0 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/mattn/go-shellwords v1.0.10 // indirect
	github.com/rs/cors v1.7.0
	github.com/spf13/cast v1.3.0
	github.com/spf13/viper v1.7.0
	google.golang.org/api v0.30.0 // indirect
)
