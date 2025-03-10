// Copyright 2024 Coral Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +k8s:deepcopy-gen=package,register
//go:generate controller-gen crd:trivialVersions=true paths=. output:dir=.

// +groupName=coral.ctx.sh
// +versionName=v1beta1

package v1beta1

// +kubebuilder:docs-gen:collapse=Apache License
