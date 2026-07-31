package hrobot_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	hrobot "github.com/Foresee-Security/hrobot-go"
)

const (
	testUser = "user"
	testPass = "pass"

	testServerID  = 321
	testServerID2 = 421

	testIP  = "123.123.123.123"
	testIP2 = "124.124.124.124"
)

// recordedRequest is what the client actually put on the wire.
//
// Asserting on it is the point of this harness. A test server that answers
// every request with the same fixture proves only that the response decodes,
// which is how a method pointed at an endpoint that does not exist can still
// pass its test.
type recordedRequest struct {
	Method string
	// Path is the decoded URL path, which is what a handler routes on.
	Path string
	// RequestURI is the raw request target exactly as it arrived, before any
	// percent-decoding. Escaping assertions belong here, because Path has
	// already turned %2F back into a slash.
	RequestURI string
	RawQuery   string
	Body       string
	UserAgent  string
	AuthUser   string
	AuthPass   string
	AuthOK     bool
}

// recorder collects the requests a test server received.
//
// The handler runs on the server's goroutine while assertions run on the test's,
// so access is mutex-guarded rather than relying on the round trip to order it.
type recorder struct {
	mu       sync.Mutex
	requests []recordedRequest
}

func (rec *recorder) record(r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	user, pass, ok := r.BasicAuth()

	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.requests = append(rec.requests, recordedRequest{
		Method:     r.Method,
		Path:       r.URL.Path,
		RequestURI: r.RequestURI,
		RawQuery:   r.URL.RawQuery,
		Body:       string(body),
		UserAgent:  r.Header.Get("User-Agent"),
		AuthUser:   user,
		AuthPass:   pass,
		AuthOK:     ok,
	})
}

// count reports how many requests reached the server. Zero is the assertion
// that proves an argument was rejected locally, before any network call.
func (rec *recorder) count() int {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return len(rec.requests)
}

// only returns the single request the server received, failing if the count is
// anything other than one.
func (rec *recorder) only(t *testing.T) recordedRequest {
	t.Helper()

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.requests) != 1 {
		t.Fatalf("want exactly 1 request, got %d", len(rec.requests))
	}
	return rec.requests[0]
}

// newServer starts a test server that records every request before handing it
// to respond, and returns a client aimed at it.
//
// Caller options are applied after the base URL, so a test can add a user agent
// or a timeout without having to rebuild the client itself.
func newServer(t *testing.T, respond http.HandlerFunc, opts ...hrobot.Option) (*hrobot.Client, *recorder) {
	t.Helper()

	url, rec := newRecordingServer(t, respond)
	all := append([]hrobot.Option{hrobot.WithBaseURL(url)}, opts...)
	return hrobot.NewBasicAuthClient(testUser, testPass, all...), rec
}

// newRecordingServer starts the recording server and hands back its URL, for
// the few tests that need to build the client themselves rather than take the
// one newServer configures.
func newRecordingServer(t *testing.T, respond http.HandlerFunc) (string, *recorder) {
	t.Helper()

	rec := &recorder{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		respond(w, r)
	}))
	t.Cleanup(ts.Close)

	return ts.URL, rec
}

// serveFixture replies 200 with the named file from testdata/response.
func serveFixture(t *testing.T, name string) http.HandlerFunc {
	t.Helper()
	body := fixture(t, name)

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// serveBody replies with a fixed status and body.
func serveBody(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

// unreachable is the responder for a test that expects no request at all. It
// fails the test if the client contacts the server anyway.
func unreachable(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("client sent an unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusTeapot)
	}
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "response", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

// wantRequest asserts the method and path the client used.
func wantRequest(t *testing.T, got recordedRequest, method, path string) {
	t.Helper()

	if got.Method != method {
		t.Errorf("method = %q, want %q", got.Method, method)
	}
	if got.Path != path {
		t.Errorf("path = %q, want %q", got.Path, path)
	}
}
