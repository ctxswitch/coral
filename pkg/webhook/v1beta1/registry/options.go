package registry

import "time"

// Options contains the configuration options for the Coral registry service.
type Options struct {
	Port                  int
	StorageDriver         string
	StorageConfig         map[string]interface{}
	LogFormat             string
	LogLevel              string
	EnableRegistryLogging bool
	EnableAccessLog       bool
	DrainTimeout          time.Duration
	EnableHTTP2           bool
	EnableH2C             bool
	UploadPurgingEnabled  bool
	UploadPurgingAge      string
	UploadPurgingInterval string
	UploadPurgingDryRun   bool
	ReadOnlyMode          bool
	DisableRedirects      bool
	HealthCheckEnabled    bool
	HealthCheckInterval   time.Duration
	HealthCheckThreshold  int
}

// setDefaults applies default values to options that aren't explicitly configured.
func (o *Options) setDefaults() {
	if o.StorageDriver == "" {
		o.StorageDriver = "inmemory"
	}
	if o.StorageConfig == nil {
		o.StorageConfig = make(map[string]interface{})
	}
	if o.LogFormat == "" {
		o.LogFormat = "json"
	}
	if o.LogLevel == "" {
		o.LogLevel = "info"
	}
	if o.DrainTimeout == 0 {
		o.DrainTimeout = 10 * time.Second
	}
	if o.UploadPurgingAge == "" {
		o.UploadPurgingAge = "168h" // 7 days
	}
	if o.UploadPurgingInterval == "" {
		o.UploadPurgingInterval = "24h" // 1 day
	}
	if o.HealthCheckInterval == 0 {
		o.HealthCheckInterval = 10 * time.Second
	}
	if o.HealthCheckThreshold == 0 {
		o.HealthCheckThreshold = 3
	}
}
