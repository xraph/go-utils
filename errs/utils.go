package errs

import "context"

// ErrorHandler handles errors from HTTP handlers.
type ErrorHandler interface {
	// HandleError handles an error and returns the formatted error response
	HandleError(ctx context.Context, err error) error
}
