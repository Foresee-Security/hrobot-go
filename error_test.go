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

// TestErrorCodesRoundTrip covers every exported ErrorCode through the real
// decode path.
//
// What it proves: each constant is unique, is shaped like a Robot code, and
// matches via [hrobot.IsError] when that literal arrives in an error document.
// A constant that is silently a duplicate of another, or that gets lowercased
// by an editor, fails here.
//
// What it cannot prove: that the string is the one Hetzner actually sends. Only
// the published error tables or live traffic settle that, and the constants
// were checked against the tables by hand. The four in error.go's second block
// are not in those tables at all and are marked as unproven there.
func TestErrorCodesRoundTrip(t *testing.T) {
	t.Parallel()

	codes := []hrobot.ErrorCode{
		hrobot.ErrorCodeRateLimitExceeded,
		hrobot.ErrorCodeConflict,
		hrobot.ErrorCodeNotFound,
		hrobot.ErrorCodeInvalidInput,
		hrobot.ErrorCodeServerNotFound,
		hrobot.ErrorCodeIPNotFound,
		hrobot.ErrorCodeSubnetNotFound,
		hrobot.ErrorCodeReverseDNSNotFound,
		hrobot.ErrorCodeResetNotAvailable,
		hrobot.ErrorCodeResetManualActive,
		hrobot.ErrorCodeResetFailed,
		hrobot.ErrorCodeBootNotAvailable,
		hrobot.ErrorCodeBootActivationFailed,
		hrobot.ErrorCodeBootDeactivationFailed,
		hrobot.ErrorCodeKeyAlreadyExists,
		hrobot.ErrorCodeKeyCreateFailed,
		hrobot.ErrorCodeKeyUpdateFailed,
		hrobot.ErrorCodeKeyDeleteFailed,
		hrobot.ErrorCodeUnauthorized,
		hrobot.ErrorCodeInternalError,
		hrobot.ErrorCodeBootAlreadyEnabled,
		hrobot.ErrorCodeBootBlocked,
	}

	seen := make(map[hrobot.ErrorCode]bool, len(codes))
	for _, code := range codes {
		if seen[code] {
			t.Errorf("duplicate ErrorCode %q", code)
		}
		seen[code] = true

		if code == "" {
			t.Error("empty ErrorCode in the list")
			continue
		}
		for _, r := range string(code) {
			if (r < 'A' || r > 'Z') && r != '_' {
				t.Errorf("ErrorCode %q contains %q, want upper case and underscores", code, r)
				break
			}
		}
	}

	for _, code := range codes {
		t.Run(string(code), func(t *testing.T) {
			t.Parallel()

			body := fmt.Sprintf(`{"error":{"code":%q,"message":"x","status":400}}`, code)
			c, _ := newServer(t, serveBody(http.StatusBadRequest, body))

			// Driven through a single-get rather than a list, because the
			// collection endpoints deliberately absorb NOT_FOUND as an empty
			// result and would swallow one of the codes under test.
			_, err := c.ServerGet(t.Context(), testServerID)
			if !hrobot.IsError(err, code) {
				t.Fatalf("IsError(err, %q) = false, err = %v", code, err)
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
			// Robot answers a rate limit with 403, not 429. The status is part
			// of the fixture so this stays honest about what the API sends.
			name:   "rate limited",
			status: http.StatusForbidden,
			body:   `{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"rate limit exceeded","status":403,"max_request":200,"interval":3600}}`,
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
