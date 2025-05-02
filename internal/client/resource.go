package client

// Resource represents any API resource that can have an ETag
type Resource interface {
	// GetID returns the resource's unique identifier
	GetID() string

	// GetETag returns the resource's ETag value
	GetETag() string

	// SetETag sets the resource's ETag value
	SetETag(etag string)

	// GetResource returns the underlying resource object
	GetResource() any
}
