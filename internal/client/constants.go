package client

import "time"

const (
	// DefaultAPIVersion is the default API version used for requests
	DefaultAPIVersion = "25r1"
	// DefaultTimeout is the default timeout for HTTP requests
	DefaultTimeout = 30 * time.Second
	// DefaultDeleteTimeout is the default timeout for waiting for delete operations to complete
	DefaultDeleteTimeout = 5 * time.Minute
	// DefaultDeletePollInterval is the default interval between polling attempts during delete operations
	DefaultDeletePollInterval = 2 * time.Second

	// Retry configuration for handling concurrent operations
	// DefaultMaxRetries is the default number of retry attempts
	DefaultMaxRetries     = 8
	DefaultBaseRetryDelay = 100 * time.Millisecond
	DefaultMaxRetryDelay  = 5 * time.Second
	DefaultMaxJitter      = 500 * time.Millisecond
)
