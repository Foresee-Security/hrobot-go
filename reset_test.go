package hrobot_test

import (
	"errors"
	"net/http"
	"slices"
	"testing"

	hrobot "github.com/Foresee-Security/hrobot-go"
)

func TestResetGet(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "reset_get.json"))

	reset, err := c.ResetGet(t.Context(), testServerID)
	if err != nil {
		t.Fatalf("ResetGet: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodGet, "/reset/321")

	if reset.ServerNumber != testServerID {
		t.Errorf("ServerNumber = %d, want %d", reset.ServerNumber, testServerID)
	}
	want := []hrobot.ResetType{
		hrobot.ResetTypeSoftware,
		hrobot.ResetTypeHardware,
		hrobot.ResetTypeManual,
	}
	if !slices.Equal(reset.Type, want) {
		t.Errorf("Type = %v, want %v", reset.Type, want)
	}
	if reset.OperatingStatus != "not supported" {
		t.Errorf("OperatingStatus = %q, want %q", reset.OperatingStatus, "not supported")
	}
}

func TestResetSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		resetType hrobot.ResetType
		wantBody  string
	}{
		{name: "hardware", resetType: hrobot.ResetTypeHardware, wantBody: "type=hw"},
		{name: "software", resetType: hrobot.ResetTypeSoftware, wantBody: "type=sw"},
		{name: "power", resetType: hrobot.ResetTypePower, wantBody: "type=power"},
		{name: "power long", resetType: hrobot.ResetTypePowerLong, wantBody: "type=power_long"},
		{name: "manual", resetType: hrobot.ResetTypeManual, wantBody: "type=man"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, rec := newServer(t, serveFixture(t, "reset_post.json"))

			input := &hrobot.ResetSetInput{Type: tc.resetType}
			reset, err := c.ResetSet(t.Context(), testServerID, input)
			if err != nil {
				t.Fatalf("ResetSet: %v", err)
			}

			got := rec.only(t)
			wantRequest(t, got, http.MethodPost, "/reset/321")
			if got.Body != tc.wantBody {
				t.Errorf("body = %q, want %q", got.Body, tc.wantBody)
			}
			if reset.Type != hrobot.ResetTypeHardware {
				t.Errorf("Type = %q, want the fixture's hw", reset.Type)
			}
		})
	}
}

func TestResetSetSendsFormContentType(t *testing.T) {
	t.Parallel()

	var contentType string
	c, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture(t, "reset_post.json"))
	})

	input := &hrobot.ResetSetInput{Type: hrobot.ResetTypeHardware}
	if _, err := c.ResetSet(t.Context(), testServerID, input); err != nil {
		t.Fatalf("ResetSet: %v", err)
	}

	if contentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", contentType)
	}
}

func TestResetSetNotAvailable(t *testing.T) {
	t.Parallel()

	body := `{"error":{"code":"RESET_NOT_AVAILABLE","message":"reset not available","status":404}}`
	c, _ := newServer(t, serveBody(http.StatusNotFound, body))

	input := &hrobot.ResetSetInput{Type: hrobot.ResetTypePower}
	_, err := c.ResetSet(t.Context(), testServerID, input)
	if !hrobot.IsError(err, hrobot.ErrorCodeResetNotAvailable) {
		t.Fatalf("IsError(err, ErrorCodeResetNotAvailable) = false, err = %v", err)
	}
}

func TestResetMethodsValidateArgumentsLocally(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		call    func(*hrobot.Client) error
		wantErr error
	}{
		{
			name:    "ResetGet rejects a zero id",
			call:    func(c *hrobot.Client) error { _, err := c.ResetGet(t.Context(), 0); return err },
			wantErr: hrobot.ErrInvalidServerID,
		},
		{
			name: "ResetSet rejects a nil input",
			call: func(c *hrobot.Client) error {
				_, err := c.ResetSet(t.Context(), testServerID, nil)
				return err
			},
			wantErr: hrobot.ErrNilInput,
		},
		{
			name: "ResetSet rejects a zero id",
			call: func(c *hrobot.Client) error {
				_, err := c.ResetSet(t.Context(), 0, &hrobot.ResetSetInput{Type: hrobot.ResetTypeHardware})
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
			if n := rec.count(); n != 0 {
				t.Errorf("sent %d requests, want 0", n)
			}
		})
	}
}
