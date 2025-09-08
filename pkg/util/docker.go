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
	"crypto/md5" // #nosec G501
	"encoding/hex"
	"strings"
)

const (
	// DefaultSearchRegistry is the default search registry for Docker images.
	DefaultSearchRegistry = "docker.io"
)

func GetImageQualifiedName(search, image string) string {
	name := image

	parts := strings.SplitN(image, "/", 2)
	if len(parts) == 1 {
		name = search + "/library/" + parts[0]
	} else if len(parts) == 2 {
		// Handle cases like docker.io/ubuntu
		if !strings.Contains(parts[0], ".") && !strings.Contains(parts[0], ":") && parts[0] != "localhost" {
			name = "docker.io/" + image
		}
	}

	parts = strings.SplitN(image, ":", 2)
	if len(parts) == 1 {
		name += ":latest"
	}

	return name
}

func ExtractImageHostname(image string) string {
	parts := strings.Split(image, "/")
	if len(parts) == 1 {
		return DefaultSearchRegistry
	}

	// Check if the first part looks like a hostname (contains . or :)
	firstPart := parts[0]
	if strings.Contains(firstPart, ".") || strings.Contains(firstPart, ":") {
		return firstPart
	}

	return DefaultSearchRegistry
}

func ExtractImageName(image string) string {
	parts := strings.SplitN(image, "/", 2)
	if len(parts) == 1 {
		return parts[0]
	}

	imageName := parts[1]
	// Handle case where image doesn't have a tag
	if !strings.Contains(imageName, ":") {
		imageName += ":latest"
	}

	return imageName
}

func GetImageLabelValue(image string) string {
	md5Hash := md5.Sum([]byte(image)) // #nosec G401
	return hex.EncodeToString(md5Hash[:])
}
