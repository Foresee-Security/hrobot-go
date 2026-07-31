package hrobot

import (
	"net/http"
	"testing"
	"time"
)

// The rest of the suite is an external test package, which is the right default
// because it exercises the library the way a caller does. These three cases
// cannot be written that way. They assert how the constructor wires the
// http.Client, and that wiring is only observable from inside the package.
//
// Each one corresponds to a mutation that previously shipped with the whole
// gate green: removing the default timeout, and making WithHTTPClient a no-op.

func TestConstructorWiresTheDefaultTimeout(t *testing.T) {
	t.Parallel()

	c := NewBasicAuthClient("user", "pass")

	if c.httpClient.Timeout != defaultTimeout {
		t.Errorf("Timeout = %v, want %v", c.httpClient.Timeout, defaultTimeout)
	}
	if c.httpClient.Timeout == 0 {
		t.Error("Timeout is zero, which in net/http means no timeout at all")
	}
}

func TestWithTimeoutReplacesTheDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{name: "replaces the default", timeout: 5 * time.Second, want: 5 * time.Second},
		{name: "ignores zero", timeout: 0, want: defaultTimeout},
		{name: "ignores negative", timeout: -1, want: defaultTimeout},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := NewBasicAuthClient("user", "pass", WithTimeout(tc.timeout))

			if c.httpClient.Timeout != tc.want {
				t.Errorf("Timeout = %v, want %v", c.httpClient.Timeout, tc.want)
			}
		})
	}
}

func TestWithHTTPClientSubstitutesTheClient(t *testing.T) {
	t.Parallel()

	supplied := &http.Client{Timeout: 9 * time.Second}

	c := NewBasicAuthClient("user", "pass", WithHTTPClient(supplied))
	if c.httpClient != supplied {
		t.Error("WithHTTPClient did not install the supplied client")
	}

	// A nil client must leave the default in place rather than disabling
	// every request.
	d := NewBasicAuthClient("user", "pass", WithHTTPClient(nil))
	if d.httpClient == nil {
		t.Fatal("WithHTTPClient(nil) left the client without an http.Client")
	}
	if d.httpClient.Timeout != defaultTimeout {
		t.Errorf("Timeout = %v, want the default %v", d.httpClient.Timeout, defaultTimeout)
	}
}
