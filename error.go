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

// Error codes that appear in the Robot Webservice error tables, checked against
// the published documentation on 2026-07-31. The list is not exhaustive, and an
// unrecognised code is preserved verbatim in [Error.Code] rather than collapsed
// into a catch-all, so a code added upstream stays matchable without a release
// here.
const (
	// ErrorCodeRateLimitExceeded arrives with HTTP 403, not 429. The same
	// response carries max_request and interval fields that this client does
	// not currently decode.
	ErrorCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"

	ErrorCodeConflict     ErrorCode = "CONFLICT"
	ErrorCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrorCodeInvalidInput ErrorCode = "INVALID_INPUT"

	ErrorCodeServerNotFound ErrorCode = "SERVER_NOT_FOUND"

	ErrorCodeIPNotFound ErrorCode = "IP_NOT_FOUND"

	ErrorCodeSubnetNotFound ErrorCode = "SUBNET_NOT_FOUND"

	ErrorCodeReverseDNSNotFound ErrorCode = "RDNS_NOT_FOUND"

	ErrorCodeResetNotAvailable ErrorCode = "RESET_NOT_AVAILABLE"
	ErrorCodeResetManualActive ErrorCode = "RESET_MANUAL_ACTIVE"
	ErrorCodeResetFailed       ErrorCode = "RESET_FAILED"

	ErrorCodeBootNotAvailable       ErrorCode = "BOOT_NOT_AVAILABLE"
	ErrorCodeBootActivationFailed   ErrorCode = "BOOT_ACTIVATION_FAILED"
	ErrorCodeBootDeactivationFailed ErrorCode = "BOOT_DEACTIVATION_FAILED"

	ErrorCodeKeyAlreadyExists ErrorCode = "KEY_ALREADY_EXISTS"
	ErrorCodeKeyCreateFailed  ErrorCode = "KEY_CREATE_FAILED"
	ErrorCodeKeyUpdateFailed  ErrorCode = "KEY_UPDATE_FAILED"
	ErrorCodeKeyDeleteFailed  ErrorCode = "KEY_DELETE_FAILED"
)

// Codes inherited from upstream that do NOT appear in any published Robot error
// table as of 2026-07-31.
//
// They are kept because they are plausible and deleting one would break a
// caller that does receive it, but treat a match on these as unproven. If you
// observe one in a real response, move it into the block above and say so.
// Matching on a code the API never sends fails silently and forever, which is
// the failure mode these two blocks exist to keep visible.
const (
	// ErrorCodeUnauthorized is the presumed code for the documented 401. The
	// documentation describes the status but never names the code string.
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"

	ErrorCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrorCodeBootAlreadyEnabled ErrorCode = "BOOT_ALREADY_ENABLED"
	ErrorCodeBootBlocked        ErrorCode = "BOOT_BLOCKED"
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
