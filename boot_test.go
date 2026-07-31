package hrobot_test

import (
	"errors"
	"net/http"
	"slices"
	"testing"

	hrobot "github.com/Foresee-Security/hrobot-go"
)

func TestBootRescueGetInactive(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "boot_rescue_get_inactive.json"))

	rescue, err := c.BootRescueGet(t.Context(), testServerID)
	if err != nil {
		t.Fatalf("BootRescueGet: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodGet, "/boot/321/rescue")

	if rescue.Active {
		t.Error("Active = true, want false")
	}
	// Inactive means the API lists every choice rather than naming one.
	wantOS := []string{"linux", "freebsd", "vkvm"}
	if !slices.Equal([]string(rescue.OS), wantOS) {
		t.Errorf("OS = %v, want %v", rescue.OS, wantOS)
	}
	if !slices.Equal([]int(rescue.Arch), []int{64, 32}) {
		t.Errorf("Arch = %v, want [64 32]", rescue.Arch)
	}
	if rescue.Password != "" {
		t.Errorf("Password = %q, want empty for an inactive rescue system", rescue.Password)
	}
}

func TestBootRescueGetActive(t *testing.T) {
	t.Parallel()

	c, _ := newServer(t, serveFixture(t, "boot_rescue_get_active.json"))

	rescue, err := c.BootRescueGet(t.Context(), testServerID)
	if err != nil {
		t.Fatalf("BootRescueGet: %v", err)
	}

	if !rescue.Active {
		t.Error("Active = false, want true")
	}
	// Active means the same fields arrive as bare scalars. A one-element list
	// is how this client presents that.
	if !slices.Equal([]string(rescue.OS), []string{"linux"}) {
		t.Errorf("OS = %v, want [linux]", rescue.OS)
	}
	if !slices.Equal([]int(rescue.Arch), []int{64}) {
		t.Errorf("Arch = %v, want [64]", rescue.Arch)
	}
	if rescue.Password != "qwertz1234" {
		t.Errorf("Password = %q, want %q", rescue.Password, "qwertz1234")
	}
	if len(rescue.AuthorizedKeys) != 1 || rescue.AuthorizedKeys[0].Key.Name != "admin" {
		t.Errorf("AuthorizedKeys = %+v, want one key named admin", rescue.AuthorizedKeys)
	}
}

func TestBootRescueSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fixture  string
		input    *hrobot.RescueSetInput
		wantBody string
	}{
		{
			name:     "os only",
			fixture:  "boot_rescue_set.json",
			input:    &hrobot.RescueSetInput{OS: "linux"},
			wantBody: "os=linux",
		},
		{
			name:     "os and arch",
			fixture:  "boot_rescue_set.json",
			input:    &hrobot.RescueSetInput{OS: "linux", Arch: 64},
			wantBody: "arch=64&os=linux",
		},
		{
			name:     "os and authorized key",
			fixture:  "boot_rescue_set_with_key.json",
			input:    &hrobot.RescueSetInput{OS: "linux", AuthorizedKey: "fingerprint"},
			wantBody: "authorized_key=fingerprint&os=linux",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, rec := newServer(t, serveFixture(t, tc.fixture))

			rescue, err := c.BootRescueSet(t.Context(), testServerID, tc.input)
			if err != nil {
				t.Fatalf("BootRescueSet: %v", err)
			}

			got := rec.only(t)
			wantRequest(t, got, http.MethodPost, "/boot/321/rescue")
			if got.Body != tc.wantBody {
				t.Errorf("body = %q, want %q", got.Body, tc.wantBody)
			}
			if !rescue.Active {
				t.Error("Active = false, want true after arming")
			}
		})
	}
}

func TestBootRescueDelete(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "boot_rescue_delete.json"))

	rescue, err := c.BootRescueDelete(t.Context(), testServerID)
	if err != nil {
		t.Fatalf("BootRescueDelete: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodDelete, "/boot/321/rescue")

	if rescue.Active {
		t.Error("Active = true, want false after disarming")
	}
}

func TestBootLinuxGetInactive(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "boot_linux_get_inactive.json"))

	linux, err := c.BootLinuxGet(t.Context(), testServerID)
	if err != nil {
		t.Fatalf("BootLinuxGet: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodGet, "/boot/321/linux")

	if linux.Active {
		t.Error("Active = true, want false")
	}
	wantDist := []string{"CentOS 5.5 minimal", "Debian 7.8 minimal"}
	if !slices.Equal([]string(linux.Dist), wantDist) {
		t.Errorf("Dist = %v, want %v", linux.Dist, wantDist)
	}
	if !slices.Equal([]string(linux.Lang), []string{"en"}) {
		t.Errorf("Lang = %v, want [en]", linux.Lang)
	}
}

func TestBootLinuxSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fixture  string
		input    *hrobot.LinuxSetInput
		wantBody string
	}{
		{
			name:     "dist and lang",
			fixture:  "boot_linux_set.json",
			input:    &hrobot.LinuxSetInput{Dist: "CentOS 5.5 minimal", Lang: "en"},
			wantBody: "dist=CentOS+5.5+minimal&lang=en",
		},
		{
			name:    "every field",
			fixture: "boot_linux_set_with_key.json",
			input: &hrobot.LinuxSetInput{
				Dist:          "CentOS 5.5 minimal",
				Arch:          32,
				Lang:          "en",
				AuthorizedKey: "fingerprint",
			},
			wantBody: "arch=32&authorized_key=fingerprint&dist=CentOS+5.5+minimal&lang=en",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, rec := newServer(t, serveFixture(t, tc.fixture))

			linux, err := c.BootLinuxSet(t.Context(), testServerID, tc.input)
			if err != nil {
				t.Fatalf("BootLinuxSet: %v", err)
			}

			got := rec.only(t)
			wantRequest(t, got, http.MethodPost, "/boot/321/linux")
			if got.Body != tc.wantBody {
				t.Errorf("body = %q, want %q", got.Body, tc.wantBody)
			}
			if !slices.Equal([]string(linux.Dist), []string{"CentOS 5.5 minimal"}) {
				t.Errorf("Dist = %v, want the single active distribution", linux.Dist)
			}
		})
	}
}

func TestBootLinuxDelete(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "boot_linux_delete.json"))

	linux, err := c.BootLinuxDelete(t.Context(), testServerID)
	if err != nil {
		t.Fatalf("BootLinuxDelete: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodDelete, "/boot/321/linux")

	if linux.Active {
		t.Error("Active = true, want false after disarming")
	}
}

func TestBootMethodsValidateArgumentsLocally(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		call    func(*hrobot.Client) error
		wantErr error
	}{
		{
			name:    "BootRescueGet rejects a zero id",
			call:    func(c *hrobot.Client) error { _, err := c.BootRescueGet(t.Context(), 0); return err },
			wantErr: hrobot.ErrInvalidServerID,
		},
		{
			name:    "BootRescueDelete rejects a zero id",
			call:    func(c *hrobot.Client) error { _, err := c.BootRescueDelete(t.Context(), 0); return err },
			wantErr: hrobot.ErrInvalidServerID,
		},
		{
			name: "BootRescueSet rejects a nil input",
			call: func(c *hrobot.Client) error {
				_, err := c.BootRescueSet(t.Context(), testServerID, nil)
				return err
			},
			wantErr: hrobot.ErrNilInput,
		},
		{
			name: "BootRescueSet rejects a zero id",
			call: func(c *hrobot.Client) error {
				_, err := c.BootRescueSet(t.Context(), 0, &hrobot.RescueSetInput{OS: "linux"})
				return err
			},
			wantErr: hrobot.ErrInvalidServerID,
		},
		{
			name:    "BootLinuxGet rejects a zero id",
			call:    func(c *hrobot.Client) error { _, err := c.BootLinuxGet(t.Context(), 0); return err },
			wantErr: hrobot.ErrInvalidServerID,
		},
		{
			name:    "BootLinuxDelete rejects a zero id",
			call:    func(c *hrobot.Client) error { _, err := c.BootLinuxDelete(t.Context(), 0); return err },
			wantErr: hrobot.ErrInvalidServerID,
		},
		{
			name: "BootLinuxSet rejects a nil input",
			call: func(c *hrobot.Client) error {
				_, err := c.BootLinuxSet(t.Context(), testServerID, nil)
				return err
			},
			wantErr: hrobot.ErrNilInput,
		},
		{
			// BootLinuxSet arms a destructive reinstall, so its argument
			// checking is the one that most needs pinning.
			name: "BootLinuxSet rejects a zero id",
			call: func(c *hrobot.Client) error {
				_, err := c.BootLinuxSet(t.Context(), 0, &hrobot.LinuxSetInput{Dist: "Debian"})
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
