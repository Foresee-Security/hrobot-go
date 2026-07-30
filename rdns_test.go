package hrobot_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	hrobot "github.com/Foresee-Security/hrobot-go"
)

func TestRDNSGetList(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "rdns_list.json"))

	entries, err := c.RDNSGetList(t.Context())
	if err != nil {
		t.Fatalf("RDNSGetList: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodGet, "/rdns")

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].IP != testIP || entries[0].PTR != "testen.de" {
		t.Errorf("entries[0] = %+v, want %s/testen.de", entries[0], testIP)
	}
	if entries[1].IP != testIP2 || entries[1].PTR != "your-server.de" {
		t.Errorf("entries[1] = %+v, want %s/your-server.de", entries[1], testIP2)
	}
}

func TestRDNSGet(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "rdns_get.json"))

	entry, err := c.RDNSGet(t.Context(), testIP)
	if err != nil {
		t.Fatalf("RDNSGet: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodGet, "/rdns/123.123.123.123")

	if entry.IP != testIP || entry.PTR != "testen.de" {
		t.Errorf("entry = %+v, want %s/testen.de", entry, testIP)
	}
}

func TestRDNSGetRejectsEmptyIP(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, unreachable(t))

	// An empty address would otherwise build "/rdns/", which the API answers
	// with the whole collection. The caller would then get a decode failure
	// naming a type mismatch instead of the mistake they actually made.
	_, err := c.RDNSGet(t.Context(), "")
	if !errors.Is(err, hrobot.ErrEmptyIP) {
		t.Fatalf("error = %v, want ErrEmptyIP", err)
	}
	if n := rec.count(); n != 0 {
		t.Errorf("sent %d requests, want 0", n)
	}
}

// TestRDNSGetEscapesThePathSegment pins that a caller-supplied address cannot
// steer the request at a different endpoint.
//
// The assertion is on RequestURI, the bytes actually sent. An unescaped
// "../server/321" would put "/rdns/../server/321" on the wire, which any server
// that normalizes paths resolves to /server/321.
func TestRDNSGetEscapesThePathSegment(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "rdns_get.json"))

	if _, err := c.RDNSGet(t.Context(), "../server/321"); err != nil {
		t.Fatalf("RDNSGet: %v", err)
	}

	got := rec.only(t)
	if got.RequestURI != "/rdns/..%2Fserver%2F321" {
		t.Errorf("request target = %q, want the separators escaped", got.RequestURI)
	}
	if strings.Contains(got.RequestURI, "/../") {
		t.Errorf("request target %q carries an unescaped traversal", got.RequestURI)
	}
}

func TestRDNSGetNotFound(t *testing.T) {
	t.Parallel()

	body := `{"error":{"code":"RDNS_NOT_FOUND","message":"rdns entry not found","status":404}}`
	c, _ := newServer(t, serveBody(http.StatusNotFound, body))

	_, err := c.RDNSGet(t.Context(), testIP)
	if !hrobot.IsError(err, hrobot.ErrorCodeReverseDNSNotFound) {
		t.Fatalf("IsError(err, ErrorCodeReverseDNSNotFound) = false, err = %v", err)
	}
}
