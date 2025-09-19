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

	// DefaultMaxRetries is the default number of retry attempts (reduced from 20 for parallelism=1)
	DefaultMaxRetries     = 5
	DefaultBaseRetryDelay = 200 * time.Millisecond
	DefaultMaxRetryDelay  = 5 * time.Second
	DefaultMaxJitter      = 500 * time.Millisecond
)
