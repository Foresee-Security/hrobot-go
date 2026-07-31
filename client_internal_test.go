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

// TestTimeoutAndHTTPClientOptionsCommute pins that the two options can be
// written in either order.
//
// Previously WithTimeout wrote straight through to the http.Client, so the
// order decided the outcome: after WithHTTPClient it retimed the caller's
// client, and before it the timeout was discarded entirely and the request ran
// unbounded. The failing order was the one that failed open.
func TestTimeoutAndHTTPClientOptionsCommute(t *testing.T) {
	t.Parallel()

	const want = 3 * time.Second

	supplied := func() *http.Client { return &http.Client{Timeout: 90 * time.Second} }

	first := NewBasicAuthClient("user", "pass",
		WithTimeout(want),
		WithHTTPClient(supplied()),
	)
	second := NewBasicAuthClient("user", "pass",
		WithHTTPClient(supplied()),
		WithTimeout(want),
	)

	if first.httpClient.Timeout != want {
		t.Errorf("timeout before client: got %v, want %v", first.httpClient.Timeout, want)
	}
	if second.httpClient.Timeout != want {
		t.Errorf("timeout after client: got %v, want %v", second.httpClient.Timeout, want)
	}
}

// TestWithTimeoutLeavesTheSuppliedClientAlone pins that retiming happens on a
// copy. A caller's http.Client is frequently shared with the rest of their
// program, and silently changing its timeout is not this library's business.
func TestWithTimeoutLeavesTheSuppliedClientAlone(t *testing.T) {
	t.Parallel()

	supplied := &http.Client{Timeout: 90 * time.Second}

	c := NewBasicAuthClient("user", "pass",
		WithHTTPClient(supplied),
		WithTimeout(3*time.Second),
	)

	if supplied.Timeout != 90*time.Second {
		t.Errorf("supplied client was retimed to %v, want it untouched at 90s", supplied.Timeout)
	}
	if c.httpClient == supplied {
		t.Error("client was retimed in place rather than on a copy")
	}
	if c.httpClient.Transport != supplied.Transport {
		t.Error("the copy lost the supplied transport")
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
