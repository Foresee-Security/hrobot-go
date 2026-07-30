package models

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrorCodeUnauthorized      ErrorCode = "UNAUTHORIZED"
	ErrorCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"

	ErrorCodeConflict      ErrorCode = "CONFLICT"
	ErrorCodeNotFound      ErrorCode = "NOT_FOUND"
	ErrorCodeInvalidInput  ErrorCode = "INVALID_INPUT"
	ErrorCodeInternalError ErrorCode = "INTERNAL_ERROR"

	ErrorCodeServerNotFound ErrorCode = "SERVER_NOT_FOUND"

	ErrorCodeIPNotFound ErrorCode = "IP_NOT_FOUND"

	ErrorCodeSubnetNotFound ErrorCode = "SUBNET_NOT_FOUND"

	ErrorCodeReverseDNSNotFound ErrorCode = "RDNS_NOT_FOUND"

	ErrorCodeResetNotAvailable ErrorCode = "RESET_NOT_AVAILABLE"
	ErrorCodeResetManualActive ErrorCode = "RESET_MANUAL_ACTIVE"
	ErrorCodeResetFailed       ErrorCode = "RESET_FAILED"

	ErrorCodeBootNotAvailable       ErrorCode = "BOOT_NOT_AVAILABLE"
	ErrorCodeBootAlreadyEnabled     ErrorCode = "BOOT_ALREADY_ENABLED"
	ErrorCodeBootBlocked            ErrorCode = "BOOT_BLOCKED"
	ErrorCodeBootActivationFailed   ErrorCode = "BOOT_ACTIVATION_FAILED"
	ErrorCodeBootDeactivationFailed ErrorCode = "BOOT_DEACTIVATION_FAILED"

	ErrorCodeKeyAlreadyExists ErrorCode = "KEY_ALREADY_EXISTS"
	ErrorCodeKeyCreateFailed  ErrorCode = "KEY_CREATE_FAILED"
	ErrorCodeKeyUpdateFailed  ErrorCode = "KEY_UPDATE_FAILED"
	ErrorCodeKeyDeleteFailed  ErrorCode = "KEY_DELETE_FAILED"
)

type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%s (%s)", e.Message, e.Code)
}

type ErrorResponse struct {
	Error Error `json:"error"`
}

// IsError reports whether err, or any error it wraps, is an API error carrying
// the given code.
//
// This unwraps rather than type-asserting directly. A caller that adds context
// with %w is doing the right thing, and it must not cause the code match to
// silently start returning false.
func IsError(err error, code ErrorCode) bool {
	var apiErr Error
	return errors.As(err, &apiErr) && apiErr.Code == code
}
