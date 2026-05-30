package ierrors

import "fmt"

// ErrorCode represents a domain-specific error category.
type ErrorCode string

const (
	ErrVendorNotFound   ErrorCode = "VENDOR_NOT_FOUND"
	ErrPricingNotFound  ErrorCode = "PRICING_NOT_FOUND"
	ErrSessionNotFound  ErrorCode = "SESSION_NOT_FOUND"
	ErrInvalidStrategy  ErrorCode = "INVALID_STRATEGY"
	ErrInvalidArgument  ErrorCode = "INVALID_ARGUMENT"
	ErrNegotiationLimit ErrorCode = "NEGOTIATION_LIMIT"
	ErrInternal         ErrorCode = "INTERNAL_ERROR"
)

// DomainError is a structured error with code and context.
type DomainError struct {
	Code    ErrorCode
	Message string
	Context map[string]any
}

func (e *DomainError) Error() string {
	if len(e.Context) > 0 {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Context)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error { return nil }

// New creates a new DomainError.
func New(code ErrorCode, msg string, ctx map[string]any) error {
	return &DomainError{Code: code, Message: msg, Context: ctx}
}

// Code extracts the error code if the error is a DomainError.
func Code(err error) ErrorCode {
	var _ *DomainError
	if err == nil {
		return ""
	}
	if as, ok := err.(*DomainError); ok {
		return as.Code
	}
	return ErrInternal
}
