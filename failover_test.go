package hrobot_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	hrobot "github.com/Foresee-Security/hrobot-go"
)

func TestFailoverGetList(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "failover_list.json"))

	addresses, err := c.FailoverGetList(t.Context())
	if err != nil {
		t.Fatalf("FailoverGetList: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodGet, "/failover")

	if len(addresses) != 2 {
		t.Fatalf("got %d addresses, want 2", len(addresses))
	}
	if addresses[0].IP != testIP {
		t.Errorf("addresses[0].IP = %q, want %q", addresses[0].IP, testIP)
	}
	// server_ipv6_net was absent from the struct, so the API's value was
	// dropped for every failover address.
	if addresses[0].ServerIPv6Net != "2a01:4f8:d0a:2003::" {
		t.Errorf("addresses[0].ServerIPv6Net = %q, want the value the API sent", addresses[0].ServerIPv6Net)
	}
	if addresses[1].IP != "2a01:4f8:fff1::" {
		t.Errorf("addresses[1].IP = %q, want the IPv6 address", addresses[1].IP)
	}
}

func TestFailoverGet(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "failover_get.json"))

	failover, err := c.FailoverGet(t.Context(), testIP)
	if err != nil {
		t.Fatalf("FailoverGet: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodGet, "/failover/123.123.123.123")

	if failover.IP != testIP {
		t.Errorf("IP = %q, want %q", failover.IP, testIP)
	}
	// ServerIP is the owning server, ActiveServerIP the one traffic reaches.
	// They differ once the address has been switched, so a test asserting both
	// is the one that would catch them being conflated.
	if failover.ServerIP != "78.46.1.93" {
		t.Errorf("ServerIP = %q, want %q", failover.ServerIP, "78.46.1.93")
	}
	if failover.ActiveServerIP != testIP2 {
		t.Errorf("ActiveServerIP = %q, want %q", failover.ActiveServerIP, testIP2)
	}
	if failover.ServerIPv6Net != "2a01:4f8:d0a:2003::" {
		t.Errorf("ServerIPv6Net = %q, want the value the API sent", failover.ServerIPv6Net)
	}
}

func TestFailoverGetEscapesIPv6PathSegment(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "failover_get.json"))

	// A failover address may be IPv6, whose colons must survive into the path.
	if _, err := c.FailoverGet(t.Context(), "2a01:4f8:fff1::"); err != nil {
		t.Fatalf("FailoverGet: %v", err)
	}

	if got := rec.only(t); got.Path != "/failover/2a01:4f8:fff1::" {
		t.Errorf("path = %q, want /failover/2a01:4f8:fff1::", got.Path)
	}
}

// TestFailoverGetEscapesThePathSegment covers the second caller-supplied
// address, which previously had no traversal test at all. Both addressEndpoint
// callers are now pinned, so the guarantee does not rest on one of them.
func TestFailoverGetEscapesThePathSegment(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "failover_get.json"))

	if _, err := c.FailoverGet(t.Context(), "../server/321"); err != nil {
		t.Fatalf("FailoverGet: %v", err)
	}

	got := rec.only(t)
	if got.RequestURI != "/failover/..%2Fserver%2F321" {
		t.Errorf("request target = %q, want the separators escaped", got.RequestURI)
	}
	if strings.Contains(got.RequestURI, "/../") {
		t.Errorf("request target %q carries an unescaped traversal", got.RequestURI)
	}
}

func TestFailoverGetRejectsEmptyIP(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, unreachable(t))

	_, err := c.FailoverGet(t.Context(), "")
	if !errors.Is(err, hrobot.ErrEmptyIP) {
		t.Fatalf("error = %v, want ErrEmptyIP", err)
	}
	if n := rec.count(); n != 0 {
		t.Errorf("sent %d requests, want 0", n)
	}
}

func TestFailoverGetServerError(t *testing.T) {
	t.Parallel()

	c, _ := newServer(t, serveBody(http.StatusInternalServerError, ""))

	if _, err := c.FailoverGet(t.Context(), testIP); err == nil {
		t.Fatal("want an error, got nil")
	}
}
