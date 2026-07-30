package hrobot_test

import (
	"errors"
	"net/http"
	"slices"
	"testing"

	hrobot "github.com/Foresee-Security/hrobot-go"
)

func TestServerGetList(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "server_list.json"))

	servers, err := c.ServerGetList(t.Context())
	if err != nil {
		t.Fatalf("ServerGetList: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodGet, "/server")

	if len(servers) != 2 {
		t.Fatalf("got %d servers, want 2", len(servers))
	}
	if servers[0].ServerNumber != testServerID {
		t.Errorf("servers[0].ServerNumber = %d, want %d", servers[0].ServerNumber, testServerID)
	}
	if servers[1].ServerNumber != testServerID2 {
		t.Errorf("servers[1].ServerNumber = %d, want %d", servers[1].ServerNumber, testServerID2)
	}
	if servers[0].DC != "NBG1-DC1" {
		t.Errorf("servers[0].DC = %q, want %q", servers[0].DC, "NBG1-DC1")
	}
	// The second entry sends "subnet":null, which must decode to nil rather
	// than failing the whole listing.
	if servers[1].Subnet != nil {
		t.Errorf("servers[1].Subnet = %v, want nil", servers[1].Subnet)
	}
}

func TestServerGet(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "server_get.json"))

	server, err := c.ServerGet(t.Context(), testServerID)
	if err != nil {
		t.Fatalf("ServerGet: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodGet, "/server/321")

	if server.ServerNumber != testServerID {
		t.Errorf("ServerNumber = %d, want %d", server.ServerNumber, testServerID)
	}
	if server.Name != "server1" {
		t.Errorf("Name = %q, want %q", server.Name, "server1")
	}
	if server.Product != "EQ 8" {
		t.Errorf("Product = %q, want %q", server.Product, "EQ 8")
	}
	if !server.Rescue || !server.Reset {
		t.Errorf("Rescue = %v, Reset = %v, want both true", server.Rescue, server.Reset)
	}
	if len(server.Subnet) != 1 || server.Subnet[0].Mask != "64" {
		t.Errorf("Subnet = %+v, want one entry with mask 64", server.Subnet)
	}
}

func TestServerGetNotFound(t *testing.T) {
	t.Parallel()

	c, _ := newServer(t, serveBody(http.StatusNotFound, string(fixture(t, "server_get_404.json"))))

	_, err := c.ServerGet(t.Context(), testServerID)
	if err == nil {
		t.Fatal("want an error, got nil")
	}

	if !hrobot.IsError(err, hrobot.ErrorCodeServerNotFound) {
		t.Errorf("IsError(err, ErrorCodeServerNotFound) = false, err = %v", err)
	}

	var apiErr hrobot.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("errors.As did not find an hrobot.Error in %v", err)
	}
	if apiErr.Message != "server not found" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "server not found")
	}
}

func TestServerSetName(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "server_get.json"))

	input := &hrobot.ServerSetNameInput{Name: "server-name-123456"}
	if _, err := c.ServerSetName(t.Context(), testServerID, input); err != nil {
		t.Fatalf("ServerSetName: %v", err)
	}

	got := rec.only(t)
	wantRequest(t, got, http.MethodPost, "/server/321")
	if got.Body != "server_name=server-name-123456" {
		t.Errorf("body = %q, want %q", got.Body, "server_name=server-name-123456")
	}
}

// TestServerCancellationWithdrawUsesTheDocumentedEndpoint pins the routing.
//
// This method previously issued POST /server/{id}/reversal, a path the Robot
// Webservice does not define. The old test could not catch it because its
// fixture server answered every path identically.
func TestServerCancellationWithdrawUsesTheDocumentedEndpoint(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "server_cancellation.json"))

	cancellation, err := c.ServerCancellationWithdraw(t.Context(), testServerID)
	if err != nil {
		t.Fatalf("ServerCancellationWithdraw: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodDelete, "/server/321/cancellation")

	if cancellation.ServerNumber != testServerID {
		t.Errorf("ServerNumber = %d, want %d", cancellation.ServerNumber, testServerID)
	}
	// The API sends "reserved". The field was previously tagged "reservation",
	// so it silently stayed false whatever the API said.
	if !cancellation.Reserved {
		t.Error("Reserved = false, want true from the fixture's \"reserved\":true")
	}
	if cancellation.ReservationPossible {
		t.Error("ReservationPossible = true, want false")
	}
	want := []string{"Upgrade to a new server", "Server too expensive"}
	if !slices.Equal([]string(cancellation.CancellationReason), want) {
		t.Errorf("CancellationReason = %v, want %v", cancellation.CancellationReason, want)
	}
}

func TestServerMethodsValidateArgumentsLocally(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		call    func(*hrobot.Client) error
		wantErr error
	}{
		{
			name:    "ServerGet rejects a zero id",
			call:    func(c *hrobot.Client) error { _, err := c.ServerGet(t.Context(), 0); return err },
			wantErr: hrobot.ErrInvalidServerID,
		},
		{
			name:    "ServerGet rejects a negative id",
			call:    func(c *hrobot.Client) error { _, err := c.ServerGet(t.Context(), -1); return err },
			wantErr: hrobot.ErrInvalidServerID,
		},
		{
			name: "ServerSetName rejects a nil input",
			call: func(c *hrobot.Client) error {
				_, err := c.ServerSetName(t.Context(), testServerID, nil)
				return err
			},
			wantErr: hrobot.ErrNilInput,
		},
		{
			name: "ServerSetName rejects a zero id",
			call: func(c *hrobot.Client) error {
				_, err := c.ServerSetName(t.Context(), 0, &hrobot.ServerSetNameInput{Name: "x"})
				return err
			},
			wantErr: hrobot.ErrInvalidServerID,
		},
		{
			name: "ServerCancellationWithdraw rejects a zero id",
			call: func(c *hrobot.Client) error {
				_, err := c.ServerCancellationWithdraw(t.Context(), 0)
				return err
			},
			wantErr: hrobot.ErrInvalidServerID,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, rec := newServer(t, unreachable(t))

			if err := tc.call(c); !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			// The point of validating at the boundary is that no request is
			// made at all.
			if n := rec.count(); n != 0 {
				t.Errorf("sent %d requests, want 0", n)
			}
		})
	}
}
