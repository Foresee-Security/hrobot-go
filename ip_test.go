package hrobot_test

import (
	"net/http"
	"testing"
)

func TestIPGetList(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "ip_list.json"))

	ips, err := c.IPGetList(t.Context())
	if err != nil {
		t.Fatalf("IPGetList: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodGet, "/ip")

	if len(ips) != 2 {
		t.Fatalf("got %d addresses, want 2", len(ips))
	}
	if ips[0].IP != testIP {
		t.Errorf("ips[0].IP = %q, want %q", ips[0].IP, testIP)
	}
	if ips[1].IP != testIP2 {
		t.Errorf("ips[1].IP = %q, want %q", ips[1].IP, testIP2)
	}
	if ips[0].ServerNumber != testServerID {
		t.Errorf("ips[0].ServerNumber = %d, want %d", ips[0].ServerNumber, testServerID)
	}
	// The API sends "separate_mac":null, which must decode to an empty string
	// rather than failing the listing.
	if ips[0].SeparateMAC != "" {
		t.Errorf("ips[0].SeparateMAC = %q, want empty", ips[0].SeparateMAC)
	}
	if ips[1].TrafficMonthly != 20 {
		t.Errorf("ips[1].TrafficMonthly = %d, want 20", ips[1].TrafficMonthly)
	}
}

func TestIPGetListInvalidJSON(t *testing.T) {
	t.Parallel()

	c, _ := newServer(t, serveBody(http.StatusOK, "invalid JSON"))

	if _, err := c.IPGetList(t.Context()); err == nil {
		t.Fatal("want a decode error, got nil")
	}
}

func TestIPGetListServerError(t *testing.T) {
	t.Parallel()

	c, _ := newServer(t, serveBody(http.StatusInternalServerError, ""))

	if _, err := c.IPGetList(t.Context()); err == nil {
		t.Fatal("want an error, got nil")
	}
}
