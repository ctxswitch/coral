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

func GetImageLabelValue(image string) string {
	md5Hash := md5.Sum([]byte(image)) // #nosec G401
	return hex.EncodeToString(md5Hash[:])
}
