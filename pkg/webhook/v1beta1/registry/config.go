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

package registry

import (
	"fmt"

	"github.com/distribution/distribution/v3/configuration"
)

type Configuration configuration.Configuration

// NewConfiguration creates a new storage configurator.
func NewConfiguration(options *Options) *Configuration {
	config := &Configuration{}
	config = config.WithLogConfiguration(options).
		WithHTTPConfiguration(options).
		WithStorageConfiguration(options)

	return config
}

func (c *Configuration) WithLogConfiguration(options *Options) *Configuration {
	c.Log = configuration.Log{
		Formatter: options.LogFormat,
		AccessLog: configuration.AccessLog{
			Disabled: !options.EnableAccessLog,
		},
		Fields: map[string]interface{}{
			"service": "coral-registry",
		},
		Level: configuration.Loglevel(options.LogLevel),
	}

	return c
}

func (c *Configuration) WithHTTPConfiguration(options *Options) *Configuration {
	// Configure HTTP server
	addr := fmt.Sprintf(":%d", options.Port)
	httpConfig := configuration.HTTP{
		Addr: addr,
		H2C: configuration.H2C{
			Enabled: options.EnableH2C,
		},
		HTTP2: configuration.HTTP2{
			Disabled: !options.EnableHTTP2,
		},
		DrainTimeout: options.DrainTimeout,
	}
	c.HTTP = httpConfig

	return c
}

func (c *Configuration) RegistryConfig() *configuration.Configuration {
	cfg := configuration.Configuration(*c)
	return &cfg
}

func (c *Configuration) WithStorageConfiguration(options *Options) *Configuration {
	sc := configuration.Storage{
		"delete": {
			"enabled": true,
		},
		"maintenance": {
			"uploadpurging": map[interface{}]interface{}{
				"enabled":  options.UploadPurgingEnabled,
				"age":      options.UploadPurgingAge,
				"interval": options.UploadPurgingInterval,
				"dryrun":   options.UploadPurgingDryRun,
			},
			"readonly": map[interface{}]interface{}{
				"enabled": options.ReadOnlyMode,
			},
			"redirect": map[interface{}]interface{}{
				"disable": options.DisableRedirects,
			},
		},
	}

	switch options.StorageDriver {
	case "s3":
		sc[options.StorageDriver] = c.WithS3StorageConfiguration(options)
	case "inmemory":
		sc[options.StorageDriver] = c.WithInMemoryStorageConfiguration(options)
	default:
		sc[options.StorageDriver] = options.StorageConfig
	}

	c.Storage = sc
	return c
}

// buildS3Configuration creates S3-specific storage configuration with validation.
func (c *Configuration) WithS3StorageConfiguration(options *Options) configuration.Parameters {
	// Set sensible defaults for S3 if not provided
	cfg := configuration.Parameters{
		"chunksize":                   5242880,  // 5MB
		"multipartcopychunksize":      33554432, // 32MB
		"multipartcopymaxconcurrency": 100,
		"multipartcopythresholdsize":  33554432, // 32MB
		"secure":                      true,
		"v4auth":                      true,
		"encrypt":                     false,
		"forcepathstyle":              false,
		"accelerate":                  false,
		"usedualstack":                false,
	}

	for key, value := range options.StorageConfig {
		cfg[key] = value
	}

	return cfg
}

func (c *Configuration) WithInMemoryStorageConfiguration(options *Options) configuration.Parameters {
	cfg := configuration.Parameters{}
	return cfg
}
