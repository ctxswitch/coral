// Copyright 2025 Coral Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

const (
	DefaultCertDir                  string = "/etc/coral/tls"
	DefaultCACertName               string = "ca.crt"
	DefaultCertName                 string = "tls.crt"
	DefaultKeyName                  string = "tls.key"
	DefaultEnableLeaderElection     bool   = false
	DefaultSkipInsecureVerify       bool   = true
	DefaultLogLevel                 int8   = 4
	DefaultContainerdAddr           string = "unix:///run/containerd/containerd.sock"
	DefaultNamespace                string = ""
	DefaultMaxConcurrentPullers     int    = 10
	DefaultMaxConcurrentReconcilers int    = 3
	DefaultCoralHost                string = "https://coral-webhook-service.coral-system.svc"
)
