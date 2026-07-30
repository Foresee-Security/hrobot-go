package hrobot_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	hrobot "github.com/Foresee-Security/hrobot-go"
)

func TestClientSendsDefaultUserAgent(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "server_list.json"))

	if _, err := c.ServerGetList(t.Context()); err != nil {
		t.Fatalf("ServerGetList: %v", err)
	}

	got := rec.only(t)
	want := "hrobot-client/" + hrobot.Version
	if got.UserAgent != want {
		t.Errorf("User-Agent = %q, want %q", got.UserAgent, want)
	}
	if got.UserAgent != "hrobot-client/"+c.GetVersion() {
		t.Errorf("User-Agent %q disagrees with GetVersion %q", got.UserAgent, c.GetVersion())
	}
}

func TestClientSendsCustomUserAgent(t *testing.T) {
	t.Parallel()

	const want = "hrobot-testsuite/0.0.1"

	c, rec := newServer(t, serveFixture(t, "server_list.json"), hrobot.WithUserAgent(want))

	if _, err := c.ServerGetList(t.Context()); err != nil {
		t.Fatalf("ServerGetList: %v", err)
	}

	if got := rec.only(t); got.UserAgent != want {
		t.Errorf("User-Agent = %q, want %q", got.UserAgent, want)
	}
}

func TestClientSendsBasicAuth(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "server_list.json"))

	if _, err := c.ServerGetList(t.Context()); err != nil {
		t.Fatalf("ServerGetList: %v", err)
	}

	got := rec.only(t)
	if !got.AuthOK {
		t.Fatal("request carried no basic auth")
	}
	if got.AuthUser != testUser || got.AuthPass != testPass {
		t.Errorf("basic auth = %q/%q, want %q/%q", got.AuthUser, got.AuthPass, testUser, testPass)
	}
}

func TestClientSetCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		user     string
		pass     string
		wantErr  error
		wantUser string
		wantPass string
	}{
		{
			name:     "replaces both",
			user:     "new-user",
			pass:     "new-pass",
			wantUser: "new-user",
			wantPass: "new-pass",
		},
		{
			name:     "rejects empty username and changes nothing",
			user:     "",
			pass:     "new-pass",
			wantErr:  hrobot.ErrEmptyUsername,
			wantUser: testUser,
			wantPass: testPass,
		},
		{
			name:     "rejects empty password and changes nothing",
			user:     "new-user",
			pass:     "",
			wantErr:  hrobot.ErrEmptyPassword,
			wantUser: testUser,
			wantPass: testPass,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, rec := newServer(t, serveFixture(t, "server_list.json"))

			err := c.SetCredentials(tc.user, tc.pass)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("SetCredentials error = %v, want %v", err, tc.wantErr)
			}

			if _, err := c.ServerGetList(t.Context()); err != nil {
				t.Fatalf("ServerGetList: %v", err)
			}

			got := rec.only(t)
			if got.AuthUser != tc.wantUser || got.AuthPass != tc.wantPass {
				t.Errorf("basic auth = %q/%q, want %q/%q", got.AuthUser, got.AuthPass, tc.wantUser, tc.wantPass)
			}
		})
	}
}

func TestClientValidateCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "accepts a successful listing",
			status: http.StatusOK,
			body:   `[]`,
		},
		{
			// An account with no dedicated servers answers 404. The credentials
			// were still accepted, so this must not read as a failure.
			name:   "accepts an account with no servers",
			status: http.StatusNotFound,
			body:   `{"error":{"code":"SERVER_NOT_FOUND","message":"server not found","status":404}}`,
		},
		{
			name:    "rejects bad credentials",
			status:  http.StatusUnauthorized,
			body:    `{"error":{"code":"UNAUTHORIZED","message":"Unauthorized","status":401}}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, rec := newServer(t, serveBody(tc.status, tc.body))

			err := c.ValidateCredentials(t.Context())
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateCredentials error = %v, wantErr = %v", err, tc.wantErr)
			}

			// It must probe a documented endpoint, not the undocumented root.
			wantRequest(t, rec.only(t), http.MethodGet, "/server")
		})
	}
}

func TestClientRedactsPassword(t *testing.T) {
	t.Parallel()

	c := hrobot.NewBasicAuthClient("bob", "hunter2")

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		got := c.String()
		if strings.Contains(got, "hunter2") {
			t.Errorf("String leaked the password: %s", got)
		}
		if !strings.Contains(got, "bob") {
			t.Errorf("String dropped the username: %s", got)
		}
	})

	t.Run("formatting verbs", func(t *testing.T) {
		t.Parallel()

		for _, verb := range []string{"%v", "%+v", "%s"} {
			got := fmt.Sprintf(verb, c)
			if strings.Contains(got, "hunter2") {
				t.Errorf("%s leaked the password: %s", verb, got)
			}
		}
	})

	t.Run("LogValue", func(t *testing.T) {
		t.Parallel()

		got := c.LogValue().String()
		if strings.Contains(got, "hunter2") {
			t.Errorf("LogValue leaked the password: %s", got)
		}
	})
}

func TestClientRejectsOversizeResponse(t *testing.T) {
	t.Parallel()

	// One byte past the 8 MiB cap.
	huge := strings.Repeat("a", (8<<20)+1)

	c, _ := newServer(t, serveBody(http.StatusOK, huge))

	_, err := c.ServerGetList(t.Context())
	if !errors.Is(err, hrobot.ErrResponseTooLarge) {
		t.Fatalf("error = %v, want ErrResponseTooLarge", err)
	}
}

func TestClientReportsTransportFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
	}{
		{name: "unparseable url", baseURL: "http://Not a valid URL"},
		{name: "unresolvable host", baseURL: "http://does-not-exist.invalid"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := hrobot.NewBasicAuthClient(testUser, testPass, hrobot.WithBaseURL(tc.baseURL))

			if _, err := c.ServerGetList(t.Context()); err == nil {
				t.Fatal("want an error, got nil")
			}
		})
	}
}

func TestClientHonoursContextCancellation(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	c, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		<-release
		w.WriteHeader(http.StatusOK)
	})
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := c.ServerGetList(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestClientErrorsNameTheOperation(t *testing.T) {
	t.Parallel()

	c, _ := newServer(t, serveBody(http.StatusOK, "not json"))

	_, err := c.ServerGet(t.Context(), testServerID)
	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if !strings.Contains(err.Error(), "server get") {
		t.Errorf("error %q does not name the operation", err)
	}
}

func TestWithTimeoutBoundsTheRequest(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	c, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		<-release
		w.WriteHeader(http.StatusOK)
	}, hrobot.WithTimeout(50*time.Millisecond))
	t.Cleanup(func() { close(release) })

	if _, err := c.ServerGetList(t.Context()); err == nil {
		t.Fatal("want a timeout error, got nil")
	}
}

func TestOptionsIgnoreEmptyValues(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "server_list.json"))

	// An empty base URL must not blank out the configured one, and an empty
	// user agent must not blank out the default.
	hrobot.WithBaseURL("")(c)
	hrobot.WithUserAgent("")(c)
	hrobot.WithHTTPClient(nil)(c)
	hrobot.WithTimeout(0)(c)

	if _, err := c.ServerGetList(t.Context()); err != nil {
		t.Fatalf("ServerGetList: %v", err)
	}

	if got := rec.only(t); got.UserAgent != "hrobot-client/"+hrobot.Version {
		t.Errorf("User-Agent = %q, want the default", got.UserAgent)
	}
}
