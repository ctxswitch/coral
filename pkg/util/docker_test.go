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

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetImageQualifiedName(t *testing.T) {
	tests := []struct {
		name   string
		search string
		image  string
		want   string
	}{
		{
			name:   "simple image name gets library namespace and latest tag",
			search: "docker.io",
			image:  "ubuntu",
			want:   "docker.io/library/ubuntu:latest",
		},
		{
			name:   "image with latest tag gets library namespace",
			search: "docker.io",
			image:  "ubuntu:latest",
			want:   "docker.io/library/ubuntu:latest",
		},
		{
			name:   "image with namespace gets latest tag",
			search: "docker.io",
			image:  "repo/ubuntu",
			want:   "docker.io/repo/ubuntu:latest",
		},
		{
			name:   "image with namespace and tag remains unchanged",
			search: "docker.io",
			image:  "repo/ubuntu:latest",
			want:   "docker.io/repo/ubuntu:latest",
		},
		{
			name:   "fully qualified docker.io image remains unchanged",
			search: "docker.io",
			image:  "docker.io/repo/ubuntu:latest",
			want:   "docker.io/repo/ubuntu:latest",
		},
		{
			name:   "localhost registry image remains unchanged",
			search: "docker.io",
			image:  "localhost:5000/repo/ubuntu:latest",
			want:   "localhost:5000/repo/ubuntu:latest",
		},
		{
			name:   "custom registry image gets latest tag",
			search: "docker.io",
			image:  "us-west1-docker.pkg.dev/project/repo",
			want:   "us-west1-docker.pkg.dev/project/repo:latest",
		},
		{
			name:   "image with custom tag gets library namespace",
			search: "docker.io",
			image:  "ubuntu:noble",
			want:   "docker.io/library/ubuntu:noble",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetImageQualifiedName(tt.search, tt.image)
			assert.Equal(t, tt.want, got, "GetImageQualifiedName(%q, %q) = %q; want %q", tt.search, tt.image, got, tt.want)
		})
	}
}

func TestGetImageLabelValue(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  string
	}{
		{
			name:  "docker.io library ubuntu image returns consistent MD5 hash",
			image: "docker.io/library/ubuntu:latest",
			want:  "942849ee1519b80488bad583231163a5",
		},
		{
			name:  "docker.io repo ubuntu image returns consistent MD5 hash",
			image: "docker.io/repo/ubuntu:latest",
			want:  "25bf9cd5dac1489ad385fde5d2c6c17e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetImageLabelValue(tt.image)
			assert.Equal(t, tt.want, got, "GetImageLabelValue(%q) = %q; want %q", tt.image, got, tt.want)
		})
	}
}

func TestExtractImageName(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  string
	}{
		{
			name:  "simple image name without slash returns as-is",
			image: "nginx",
			want:  "nginx",
		},
		{
			name:  "image with tag but no slash returns as-is",
			image: "nginx:1.21",
			want:  "nginx:1.21",
		},
		{
			name:  "image with latest tag but no slash returns as-is",
			image: "nginx:latest",
			want:  "nginx:latest",
		},
		{
			name:  "namespace/image without registry gets image with latest tag",
			image: "library/nginx",
			want:  "nginx:latest",
		},
		{
			name:  "namespace/image with tag but no registry gets image unchanged",
			image: "library/nginx:1.21",
			want:  "nginx:1.21",
		},
		{
			name:  "registry with image gets image part with latest tag",
			image: "docker.io/nginx",
			want:  "nginx:latest",
		},
		{
			name:  "registry with image and tag gets image part unchanged",
			image: "docker.io/nginx:1.21",
			want:  "nginx:1.21",
		},
		{
			name:  "registry with namespace and image gets namespace/image with latest tag",
			image: "docker.io/library/nginx",
			want:  "library/nginx:latest",
		},
		{
			name:  "registry with namespace image and tag gets namespace/image unchanged",
			image: "docker.io/library/nginx:1.21",
			want:  "library/nginx:1.21",
		},
		{
			name:  "custom registry with image gets image part with latest tag",
			image: "registry.example.com/nginx",
			want:  "nginx:latest",
		},
		{
			name:  "custom registry with image and tag gets image part unchanged",
			image: "registry.example.com/nginx:1.21",
			want:  "nginx:1.21",
		},
		{
			name:  "localhost registry with image gets image part with latest tag",
			image: "localhost:5000/nginx",
			want:  "nginx:latest",
		},
		{
			name:  "localhost registry with image and tag gets image part unchanged",
			image: "localhost:5000/nginx:1.21",
			want:  "nginx:1.21",
		},
		{
			name:  "deep path registry gets full path with latest tag",
			image: "registry.example.com/team/project/service",
			want:  "team/project/service:latest",
		},
		{
			name:  "deep path registry with tag gets full path unchanged",
			image: "registry.example.com/team/project/service:v1.0",
			want:  "team/project/service:v1.0",
		},
		{
			name:  "gcr.io with project and image gets project/image with latest tag",
			image: "gcr.io/my-project/nginx",
			want:  "my-project/nginx:latest",
		},
		{
			name:  "gcr.io with project image and tag gets project/image unchanged",
			image: "gcr.io/my-project/nginx:1.21",
			want:  "my-project/nginx:1.21",
		},
		{
			name:  "quay.io with org and image gets org/image with latest tag",
			image: "quay.io/organization/nginx",
			want:  "organization/nginx:latest",
		},
		{
			name:  "AWS ECR with repo gets repo with latest tag",
			image: "123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo",
			want:  "my-repo:latest",
		},
		{
			name:  "Azure ACR with image gets image with latest tag",
			image: "myregistry.azurecr.io/nginx",
			want:  "nginx:latest",
		},
		{
			name:  "image with SHA digest gets image part with digest",
			image: "registry.example.com/nginx@sha256:abc123",
			want:  "nginx@sha256:abc123",
		},
		{
			name:  "complex path with SHA digest gets full path with digest",
			image: "registry.example.com/team/project/service@sha256:abc123",
			want:  "team/project/service@sha256:abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractImageName(tt.image)
			assert.Equal(t, tt.want, got, "ExtractImageName(%q) = %q; want %q", tt.image, got, tt.want)
		})
	}
}

func TestExtractImageHostname(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  string
	}{
		{
			name:  "simple image name without registry",
			image: "nginx",
			want:  "docker.io",
		},
		{
			name:  "image with tag but no registry",
			image: "nginx:latest",
			want:  "docker.io",
		},
		{
			name:  "image with user/org but no registry",
			image: "library/nginx",
			want:  "docker.io",
		},
		{
			name:  "image with user/org and tag but no registry",
			image: "library/nginx:1.21",
			want:  "docker.io",
		},
		{
			name:  "explicit docker.io registry with FQDN",
			image: "docker.io/nginx",
			want:  "docker.io",
		},
		{
			name:  "explicit docker.io registry with library namespace",
			image: "docker.io/library/nginx",
			want:  "docker.io",
		},
		{
			name:  "custom registry with FQDN",
			image: "registry.example.com/nginx",
			want:  "registry.example.com",
		},
		{
			name:  "custom registry with port",
			image: "registry.example.com:5000/nginx",
			want:  "registry.example.com:5000",
		},
		{
			name:  "localhost registry",
			image: "localhost:5000/nginx",
			want:  "localhost:5000",
		},
		{
			name:  "gcr.io registry",
			image: "gcr.io/project/image",
			want:  "gcr.io",
		},
		{
			name:  "quay.io registry",
			image: "quay.io/organization/image",
			want:  "quay.io",
		},
		{
			name:  "AWS ECR registry",
			image: "123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo",
			want:  "123456789012.dkr.ecr.us-west-2.amazonaws.com",
		},
		{
			name:  "Azure ACR registry",
			image: "myregistry.azurecr.io/nginx",
			want:  "myregistry.azurecr.io",
		},
		{
			name:  "image with deep path",
			image: "registry.example.com/team/project/service:v1.0",
			want:  "registry.example.com",
		},
		{
			name:  "registry with IP address",
			image: "192.168.1.100:5000/nginx",
			want:  "192.168.1.100:5000",
		},
		{
			name:  "registry with subdomain",
			image: "harbor.dev.example.com/library/nginx",
			want:  "harbor.dev.example.com",
		},
		{
			name:  "hyphenated registry hostname",
			image: "my-registry.example.com/nginx",
			want:  "my-registry.example.com",
		},
		{
			name:  "registry with underscores in hostname",
			image: "my_registry.example.com/nginx",
			want:  "my_registry.example.com",
		},
		{
			name:  "Docker Hub official namespace shorthand",
			image: "ubuntu/nginx",
			want:  "docker.io",
		},
		{
			name:  "single character registry name",
			image: "r/nginx",
			want:  "docker.io",
		},
		{
			name:  "complex image with SHA digest",
			image: "registry.example.com/project/image@sha256:abc123",
			want:  "registry.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractImageHostname(tt.image)
			assert.Equal(t, tt.want, got, "ExtractImageHostname(%q) = %q; want %q", tt.image, got, tt.want)
		})
	}
}
