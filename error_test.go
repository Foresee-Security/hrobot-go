package hrobot_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	hrobot "github.com/Foresee-Security/hrobot-go"
)

func TestIsErrorMatchesThroughWrapping(t *testing.T) {
	t.Parallel()

	base := hrobot.Error{Code: hrobot.ErrorCodeServerNotFound, Message: "server not found"}

	tests := []struct {
		name string
		err  error
		code hrobot.ErrorCode
		want bool
	}{
		{name: "direct", err: base, code: hrobot.ErrorCodeServerNotFound, want: true},
		{
			name: "wrapped once",
			err:  fmt.Errorf("load server: %w", base),
			code: hrobot.ErrorCodeServerNotFound,
			want: true,
		},
		{
			name: "wrapped twice",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", base)),
			code: hrobot.ErrorCodeServerNotFound,
			want: true,
		},
		{name: "different code", err: base, code: hrobot.ErrorCodeNotFound, want: false},
		{name: "unrelated error", err: errors.New("boom"), code: hrobot.ErrorCodeServerNotFound, want: false},
		{name: "nil error", err: nil, code: hrobot.ErrorCodeServerNotFound, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := hrobot.IsError(tc.err, tc.code); got != tc.want {
				t.Errorf("IsError = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestErrorMessage(t *testing.T) {
	t.Parallel()

	err := hrobot.Error{Code: hrobot.ErrorCodeBootBlocked, Message: "boot blocked"}
	if got, want := err.Error(), "boot blocked (BOOT_BLOCKED)"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestNonAPIErrorBodiesBecomeStatusErrors covers the responses an intermediary
// produces, which are not Robot error documents and previously collapsed into
// an untyped message a caller could only string-match.
func TestNonAPIErrorBodiesBecomeStatusErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "html from a proxy", status: http.StatusBadGateway, body: "<html>502 Bad Gateway</html>"},
		{name: "empty body", status: http.StatusInternalServerError, body: ""},
		{name: "valid json that is not an error document", status: http.StatusForbidden, body: `{"foo":"bar"}`},
		{name: "error document with blank fields", status: http.StatusTeapot, body: `{"error":{"code":"","message":""}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, _ := newServer(t, serveBody(tc.status, tc.body))

			_, err := c.ServerGetList(t.Context())
			if err == nil {
				t.Fatal("want an error, got nil")
			}

			var statusErr hrobot.StatusError
			if !errors.As(err, &statusErr) {
				t.Fatalf("errors.As did not find a StatusError in %v", err)
			}
			if statusErr.StatusCode != tc.status {
				t.Errorf("StatusCode = %d, want %d", statusErr.StatusCode, tc.status)
			}
		})
	}
}

func TestAPIErrorBodiesBecomeTypedErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
		want   hrobot.ErrorCode
	}{
		{
			name:   "unauthorized",
			status: http.StatusUnauthorized,
			body:   `{"error":{"code":"UNAUTHORIZED","message":"Unauthorized","status":401}}`,
			want:   hrobot.ErrorCodeUnauthorized,
		},
		{
			name:   "rate limited",
			status: http.StatusTooManyRequests,
			body:   `{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"rate limit exceeded","status":429}}`,
			want:   hrobot.ErrorCodeRateLimitExceeded,
		},
		{
			// A code this library has no constant for must still round-trip,
			// so a code added upstream stays matchable without a release here.
			name:   "unknown code is preserved",
			status: http.StatusBadRequest,
			body:   `{"error":{"code":"SOMETHING_NEW","message":"new failure","status":400}}`,
			want:   hrobot.ErrorCode("SOMETHING_NEW"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, _ := newServer(t, serveBody(tc.status, tc.body))

			_, err := c.ServerGetList(t.Context())
			if !hrobot.IsError(err, tc.want) {
				t.Fatalf("IsError(err, %q) = false, err = %v", tc.want, err)
			}
		})
	}
}
