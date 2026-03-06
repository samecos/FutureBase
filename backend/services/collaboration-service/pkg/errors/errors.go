package errors

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CollaborationError represents a collaboration service error
type CollaborationError struct {
	Code      codes.Code
	Message   string
	Details   map[string]interface{}
	Retryable bool
	Cause     error
}

// Error implements the error interface
func (e *CollaborationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the wrapped error
func (e *CollaborationError) Unwrap() error {
	return e.Cause
}

// GRPCStatus converts the error to a gRPC status
func (e *CollaborationError) GRPCStatus() *status.Status {
	st := status.New(e.Code, e.Message)
	if len(e.Details) > 0 {
		// Add details if needed
		// This is a simplified version
	}
	return st
}

// New creates a new CollaborationError
func New(code codes.Code, message string) *CollaborationError {
	return &CollaborationError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// Wrap wraps an existing error
func Wrap(err error, code codes.Code, message string) *CollaborationError {
	return &CollaborationError{
		Code:    code,
		Message: message,
		Cause:   err,
		Details: make(map[string]interface{}),
	}
}

// WithDetail adds a detail to the error
func (e *CollaborationError) WithDetail(key string, value interface{}) *CollaborationError {
	e.Details[key] = value
	return e
}

// WithRetryable sets whether the error is retryable
func (e *CollaborationError) WithRetryable(retryable bool) *CollaborationError {
	e.Retryable = retryable
	return e
}

// Predefined errors
var (
	// Session errors
	ErrSessionNotFound     = New(codes.NotFound, "session not found")
	ErrSessionExpired      = New(codes.DeadlineExceeded, "session has expired")
	ErrSessionClosed       = New(codes.FailedPrecondition, "session is closed")
	ErrSessionFull         = New(codes.ResourceExhausted, "session has reached maximum participants")
	ErrSessionAlreadyExists = New(codes.AlreadyExists, "session already exists")
	
	// Permission errors
	ErrPermissionDenied    = New(codes.PermissionDenied, "permission denied")
	ErrInvalidPermission   = New(codes.InvalidArgument, "invalid permission level")
	ErrUserNotInSession    = New(codes.NotFound, "user is not in session")
	ErrUserAlreadyInSession = New(codes.AlreadyExists, "user is already in session")
	
	// Operation errors
	ErrInvalidOperation    = New(codes.InvalidArgument, "invalid operation")
	ErrOperationNotFound   = New(codes.NotFound, "operation not found")
	ErrOperationTooLarge   = New(codes.ResourceExhausted, "operation data too large")
	ErrOperationConflict   = New(codes.Aborted, "operation conflict detected")
	ErrOperationTimeout    = New(codes.DeadlineExceeded, "operation timed out")
	
	// CRDT errors
	ErrYjsApplyFailed      = New(codes.Internal, "failed to apply Yjs update")
	ErrYjsStateCorrupted   = New(codes.DataLoss, "Yjs state is corrupted")
	ErrInvalidStateVector  = New(codes.InvalidArgument, "invalid state vector")
	
	// Rate limiting errors
	ErrRateLimitExceeded   = New(codes.ResourceExhausted, "rate limit exceeded")
	ErrTooManyRequests     = New(codes.ResourceExhausted, "too many requests")
	
	// General errors
	ErrInternalError       = New(codes.Internal, "internal server error")
	ErrNotImplemented      = New(codes.Unimplemented, "not implemented")
	ErrInvalidArgument     = New(codes.InvalidArgument, "invalid argument")
	ErrUnauthenticated     = New(codes.Unauthenticated, "unauthenticated")
)

// ConvertToGRPCError converts an error to a gRPC error
func ConvertToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	// Check if it's already a gRPC status
	if _, ok := status.FromError(err); ok {
		return err
	}

	// Check if it's a CollaborationError
	var collabErr *CollaborationError
	if errors.As(err, &collabErr) {
		return collabErr.GRPCStatus().Err()
	}

	// Map common errors
	switch {
	case errors.Is(err, ErrSessionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrSessionExpired):
		return status.Error(codes.DeadlineExceeded, err.Error())
	case errors.Is(err, ErrPermissionDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, ErrInvalidOperation):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrOperationConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, ErrRateLimitExceeded):
		return status.Error(codes.ResourceExhausted, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	var collabErr *CollaborationError
	if errors.As(err, &collabErr) {
		return collabErr.Retryable
	}

	// Check gRPC status codes
	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case codes.DeadlineExceeded,
		codes.Unavailable,
		codes.Aborted,
		codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

// ErrorCode returns the gRPC error code
func ErrorCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}

	st, ok := status.FromError(err)
	if !ok {
		return codes.Unknown
	}

	return st.Code()
}
