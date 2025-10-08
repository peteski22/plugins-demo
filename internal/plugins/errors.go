package plugins

import "errors"

var (
	// ErrInvalidRequestType is returned when a request payload is not the expected type (e.g. []byte).
	ErrInvalidRequestType = errors.New("invalid request type for plugin")

	// ErrInvalidResponseType is returned when a response payload is not the expected type (e.g. []byte).
	ErrInvalidResponseType = errors.New("invalid response type for plugin")

	// ErrRequiredPluginFailed is returned when a required plugin fails to handle its request.
	ErrRequiredPluginFailed = errors.New("required plugin failed to handle request")
)
