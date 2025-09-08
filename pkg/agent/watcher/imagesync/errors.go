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

package imagesync

type WatchError string

func (e WatchError) Error() string {
	return string(e)
}

const (
	ErrNodeNotFound        WatchError = "node not found"
	ErrNodeNotReady        WatchError = "node not ready"
	ErrNodeMatch           WatchError = "node does not match match"
	ErrImageSyncNotFound   WatchError = "imagesync notfound"
	ErrPullSecretsNotFound WatchError = "pull secrets not found"
)
