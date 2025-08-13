package storage

import "errors"

// Common storage errors
var (
	ErrBackendNotFound  = errors.New("storage backend not found")
	ErrNoDefaultBackend = errors.New("no default storage backend configured")
	ErrKeyNotFound      = errors.New("key not found in storage")
	ErrInvalidConfig    = errors.New("invalid storage configuration")
	ErrBackendNotReady  = errors.New("storage backend not ready")
	ErrUnsupportedOp    = errors.New("operation not supported by this backend")
)
