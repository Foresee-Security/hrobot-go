package hrobot

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// ErrorCode is the machine-readable error identifier the Robot Webservice
// returns alongside a human-readable message. Match it with [IsError] rather
// than comparing message text, which is not part of the API contract.
type ErrorCode string

// Error codes documented by the Robot Webservice. The list is not exhaustive.
// An unrecognised code is preserved verbatim in [Error.Code] rather than being
// collapsed into a catch-all, so a code added upstream stays matchable.
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

// Error is an error reported by the Robot Webservice in its own error document.
// Methods return it wrapped with the operation that produced it, so match it
// with [IsError] or [errors.As] rather than by comparison.
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// Error implements the error interface.
func (e Error) Error() string {
	return fmt.Sprintf("%s (%s)", e.Message, e.Code)
}

// StatusError reports an HTTP failure whose body was not a Robot error
// document, which is what an intermediary such as a proxy or load balancer
// produces. It carries the status code so a caller can still distinguish a
// retryable 5xx from a terminal 4xx.
type StatusError struct {
	StatusCode int
}

// Error implements the error interface.
func (e StatusError) Error() string {
	return "server responded with status code " + strconv.Itoa(e.StatusCode)
}

// IsError reports whether err, or any error it wraps, is a Robot API [Error]
// carrying the given code.
//
// This unwraps rather than type-asserting directly. A caller that adds context
// with %w is doing the right thing, and it must not cause the code match to
// silently start returning false.
func IsError(err error, code ErrorCode) bool {
	var apiErr Error
	return errors.As(err, &apiErr) && apiErr.Code == code
}

// errorResponse is the envelope the Robot Webservice wraps an error document in.
type errorResponse struct {
	Error Error `json:"error"`
}

// errorFromResponse converts a failed response into the most specific error the
// body supports. A body that does not decode into a Robot error document, or
// that decodes into an empty one, yields a [StatusError] rather than a
// misleadingly blank [Error].
func errorFromResponse(statusCode int, body []byte) error {
	var resp errorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return StatusError{StatusCode: statusCode}
	}
	if resp.Error.Code == "" && resp.Error.Message == "" {
		return StatusError{StatusCode: statusCode}
	}
	return resp.Error
}
