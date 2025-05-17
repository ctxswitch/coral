package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetImageQualifiedName(t *testing.T) {
	tests := []struct {
		search string
		image  string
		want   string
	}{
		{"docker.io", "ubuntu", "docker.io/library/ubuntu:latest"},
		{"docker.io", "ubuntu:latest", "docker.io/library/ubuntu:latest"},
		{"docker.io", "repo/ubuntu", "docker.io/repo/ubuntu:latest"},
		{"docker.io", "repo/ubuntu:latest", "docker.io/repo/ubuntu:latest"},
		{"docker.io", "docker.io/repo/ubuntu:latest", "docker.io/repo/ubuntu:latest"},
		{"docker.io", "localhost:5000/repo/ubuntu:latest", "localhost:5000/repo/ubuntu:latest"},
		{"docker.io", "us-west1-docker.pkg.dev/project/repo", "us-west1-docker.pkg.dev/project/repo:latest"},
		{"docker.io", "ubuntu:noble", "docker.io/library/ubuntu:noble"},
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			got := GetImageQualifiedName(tt.search, tt.image)
			assert.Equal(t, tt.want, got, "GetImageQualifiedName(%q, %q) = %q; want %q", tt.search, tt.image, got, tt.want)
		})
	}
}

func TestGetImageLabelValue(t *testing.T) {
	tests := []struct {
		image string
		want  string
	}{
		{"docker.io/library/ubuntu:latest", "942849ee1519b80488bad583231163a5"},
		{"docker.io/repo/ubuntu:latest", "25bf9cd5dac1489ad385fde5d2c6c17e"},
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			got := GetImageLabelValue(tt.image)
			assert.Equal(t, tt.want, got, "GetImageLabelValue(%q) = %q; want %q", tt.image, got, tt.want)
		})
	}
}
