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

import "time"

const (
	DefaultCertDir              string        = "/etc/webhook/tls"
	DefaultEnableLeaderElection bool          = false
	DefaultSkipInsecureVerify   bool          = false
	DefaultLogLevel             int8          = 0
	DefaultPollInterval         time.Duration = 10 * time.Second
	DefaultContainerdAddr       string        = "unix:///kubelet/containerd/containerd.sock"
	DefaultNamespace            string        = ""
	DefaultScope                string        = ""
	DefaultLabels               string        = "app=coral,component=mirror"
	DefaultParallel             int           = 1

	ConnectionTimeout  time.Duration = 30 * time.Second
	MaxCallRecvMsgSize int           = 1024 * 1024 * 32
)
