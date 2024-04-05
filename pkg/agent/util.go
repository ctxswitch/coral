// Copyright 2024 Coral Authors
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

package agent

import (
	"context"

	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
	stvziov1 "stvz.io/coral/pkg/apis/stvz.io/v1"
)

func GetImageIdentifiers(ctx context.Context, ims runtime.ImageServiceClient) (map[string]string, error) {
	resp, err := ims.ListImages(ctx, &runtime.ListImagesRequest{})
	if err != nil {
		return nil, err
	}

	ids := make(map[string]string)
	for _, img := range resp.Images {
		for _, tag := range img.RepoTags {
			ids[tag] = img.Id
		}
	}

	return ids, nil
}

func ImageMap(ctx context.Context, ims runtime.ImageServiceClient) (map[string]string, error) {
	resp, err := ims.ListImages(ctx, &runtime.ListImagesRequest{})
	if err != nil {
		return nil, err
	}

	tags := make(map[string]string)
	for _, img := range resp.Images {
		for _, tag := range img.RepoTags {
			tags[tag] = stvziov1.HashedImageLabelKey(tag)
		}
	}

	return tags, nil
}
