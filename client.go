package hrobot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultBaseURL is the public endpoint of the Robot Webservice.
const DefaultBaseURL = "https://robot-ws.your-server.de"

// Version is this library's version. It appears in the default User-Agent and
// is reported by [Client.GetVersion].
const Version = "0.3.0"

const (
	defaultUserAgent = "hrobot-client/" + Version

	// defaultTimeout bounds a request whose caller supplied no deadline of its
	// own. A zero-value http.Client has no timeout whatsoever, so without this
	// a single unresponsive request blocks its caller until the process dies.
	// A context deadline, being the tighter of the two in practice, still wins.
	defaultTimeout = 30 * time.Second

	// maxResponseBytes caps how much of a response body is read into memory.
	// The Robot API returns small JSON documents, so anything larger is a
	// broken or hostile endpoint rather than a response worth parsing.
	maxResponseBytes = 8 << 20 // 8 MiB
)

// Sentinel errors returned for misuse detected before a request is sent, and
// for a response this client refuses to process. Match them with [errors.Is].
var (
	// ErrResponseTooLarge means the response body exceeded 8 MiB. The body is
	// discarded rather than truncated, because a truncated JSON document fails
	// to parse for a reason the caller cannot diagnose.
	ErrResponseTooLarge = errors.New("hrobot: response body exceeds the maximum size")

	// ErrEmptyUsername means SetCredentials was given an empty username.
	ErrEmptyUsername = errors.New("hrobot: username is empty")

	// ErrEmptyPassword means SetCredentials was given an empty password.
	ErrEmptyPassword = errors.New("hrobot: password is empty")

	// ErrNilInput means a method was called with a nil input struct.
	ErrNilInput = errors.New("hrobot: input is nil")

	// ErrInvalidServerID means a server number was zero or negative. Robot
	// server numbers are positive, so such a call could only ever 404.
	ErrInvalidServerID = errors.New("hrobot: server ID must be a positive integer")

	// ErrEmptyIP means an IP address argument was empty. An empty address
	// would silently address the collection endpoint instead of one member.
	ErrEmptyIP = errors.New("hrobot: IP address is empty")
)

// Option configures a [Client] at construction. See [WithBaseURL],
// [WithUserAgent], [WithHTTPClient] and [WithTimeout].
type Option func(*Client)

// WithBaseURL directs the client at a different endpoint, such as a test
// server. A trailing slash is trimmed. An empty url is ignored, which keeps
// [DefaultBaseURL] rather than producing requests to a relative path.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if baseURL != "" {
			c.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

// WithUserAgent replaces the User-Agent header sent with every request. An
// empty value is ignored. Hetzner asks that clients identify themselves, which
// also makes this library's traffic attributable in a rate-limit investigation.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		if userAgent != "" {
			c.userAgent = userAgent
		}
	}
}

// WithHTTPClient supplies the http.Client used for every request, for callers
// that need their own transport, proxy or instrumentation.
//
// The caller then owns the timeout. Nothing is imposed on a client supplied
// this way, because overriding a deliberately configured transport would be
// worse than the bound it replaces. Pair it with a context deadline, or with
// [WithTimeout] applied after this option. A nil client is ignored.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithTimeout sets the per-request timeout, replacing the 30-second default. A
// non-positive duration is ignored.
//
// Order does not matter. The timeout is recorded and applied once every option
// has run, so it takes effect whether it is written before or after
// [WithHTTPClient]. Combined with a supplied client it retimes a copy, leaving
// the caller's own client and anything else sharing it untouched.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// Client talks to the Hetzner Robot Webservice.
//
// Safe for concurrent use. The endpoint, user agent and http.Client are fixed
// at construction, and the credentials are guarded, so [Client.SetCredentials]
// may rotate them while requests are in flight.
type Client struct {
	baseURL    string
	userAgent  string
	httpClient *http.Client

	// timeout is what WithTimeout recorded, applied to httpClient once every
	// option has run so the two options commute.
	timeout time.Duration

	// mu guards username and password, the only mutable fields, so that
	// SetCredentials cannot race a request reading them.
	mu       sync.RWMutex
	username string
	password string
}

// Compile-time proof that *Client satisfies the documented surface. Consumers
// should prefer to declare their own narrow interface covering only the calls
// they make, per the Go convention of defining interfaces at the consumer.
var _ RobotClient = (*Client)(nil)

// NewBasicAuthClient returns a Client authenticating as the given Robot
// Webservice user against [DefaultBaseURL].
//
// Robot Webservice credentials are created in the Robot web interface and are
// distinct from both the Hetzner account login and any Hetzner Cloud token.
//
// The returned client carries a 30-second per-request timeout, so a caller
// passing a context without a deadline is still bounded.
//
// Empty credentials are accepted here, unlike [Client.SetCredentials] which
// rejects them. The asymmetry is deliberate. Construction has nowhere to report
// an error without forcing every caller to handle one, and an empty pair simply
// fails the first request with a 401, which is loud and immediate.
// SetCredentials rejects them because it mutates a client that is already in
// use, where a half-applied rotation would be neither the old pair nor the new.
func NewBasicAuthClient(username, password string, opts ...Option) *Client {
	c := &Client{
		baseURL:   DefaultBaseURL,
		userAgent: defaultUserAgent,
		httpClient: &http.Client{
			Timeout:       defaultTimeout,
			CheckRedirect: refuseCrossOriginRedirect,
		},
		username: username,
		password: password,
	}
	for _, opt := range opts {
		opt(c)
	}

	// Applied after every option so the result does not depend on the order
	// they were written in. The copy is what keeps a caller-supplied client
	// from being retimed underneath whatever else is sharing it.
	if c.timeout > 0 && c.httpClient.Timeout != c.timeout {
		retimed := *c.httpClient
		retimed.Timeout = c.timeout
		c.httpClient = &retimed
	}
	return c
}

// String renders the client without its password, so a stray %v or %+v cannot
// leak the credential. Redaction lives on the type rather than at call sites,
// because a call site that forgets is exactly the case that leaks.
func (c *Client) String() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return fmt.Sprintf("hrobot.Client{username: %q, password: REDACTED, baseURL: %q}", c.username, c.baseURL)
}

// LogValue is the structured-logging counterpart to [Client.String], so slog
// output redacts the password too.
func (c *Client) LogValue() slog.Value {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return slog.GroupValue(
		slog.String("username", c.username),
		slog.String("password", "REDACTED"),
		slog.String("base_url", c.baseURL),
	)
}

// GetVersion reports this library's version, the same value as [Version].
func (c *Client) GetVersion() string {
	return Version
}

// SetCredentials replaces the credentials used by subsequent requests. It
// returns [ErrEmptyUsername] or [ErrEmptyPassword] without changing anything
// if either argument is empty, so a partial rotation cannot leave the client
// authenticating with a half-updated pair.
//
// Safe to call while requests are in flight. A request that has already read
// the credentials completes with the previous pair.
func (c *Client) SetCredentials(username, password string) error {
	if username == "" {
		return ErrEmptyUsername
	}
	if password == "" {
		return ErrEmptyPassword
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.username = username
	c.password = password
	return nil
}

// ValidateCredentials reports whether the Robot Webservice accepts the
// configured credentials, by issuing one real request against the server
// listing.
//
// A nil return means the credentials authenticated, not that the account owns
// any servers. Measured against the live API, an account with no dedicated
// servers answers the listing with 200 and an empty array.
//
// A 404 is also treated as success. Reaching a not-found answer at all proves
// authentication succeeded, since the API rejects an unauthenticated request
// with 401 before it ever looks for the resource. Other Robot collections such
// as /ip and /failover do answer 404 when they are empty, so this guards
// against the server listing behaving that way on some accounts.
func (c *Client) ValidateCredentials(ctx context.Context) error {
	_, err := c.do(ctx, "validate credentials", http.MethodGet, "/server", nil)
	switch {
	case err == nil:
		return nil
	case IsError(err, ErrorCodeServerNotFound), IsError(err, ErrorCodeNotFound):
		return nil
	default:
		return err
	}
}

// ErrRedirectCrossOrigin means the server tried to redirect a request to a
// different scheme, host or port, and the client refused to follow it.
var ErrRedirectCrossOrigin = errors.New("hrobot: refusing to follow a redirect to a different origin")

// refuseCrossOriginRedirect stops the client following a redirect that leaves
// the origin the caller configured.
//
// This exists because of how the standard library decides whether to resend an
// Authorization header. It compares hostnames, which excludes the port and
// ignores the scheme, so a redirect from https to http on the same host, or to
// a different port, still carries the credentials. For a basic-auth client that
// is the whole secret travelling somewhere the caller never named, in the
// downgrade case over cleartext.
//
// The Robot Webservice is a single documented endpoint and publishes no
// redirects, so refusing costs nothing that is known to work. A caller who
// supplies their own client through [WithHTTPClient] owns this decision, and
// gets the standard library default unless they set their own.
func refuseCrossOriginRedirect(req *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		return nil
	}

	origin := via[0].URL
	if req.URL.Scheme != origin.Scheme || req.URL.Host != origin.Host {
		return fmt.Errorf("%w: %s://%s to %s://%s",
			ErrRedirectCrossOrigin, origin.Scheme, origin.Host, req.URL.Scheme, req.URL.Host)
	}
	return nil
}

// do performs one API call and returns the raw response body. op names the
// operation for the error message, so a failure identifies the call that
// produced it without the caller having to add that context at every site.
func (c *Client) do(ctx context.Context, op, method string, path endpoint, form url.Values) ([]byte, error) {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+string(path), body)
	if err != nil {
		return nil, fmt.Errorf("hrobot: %s: build request: %w", op, err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("User-Agent", c.userAgent)

	c.mu.RLock()
	req.SetBasicAuth(c.username, c.password)
	c.mu.RUnlock()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hrobot: %s: %w", op, err)
	}
	defer resp.Body.Close()

	raw, err := readCapped(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("hrobot: %s: read response: %w", op, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("hrobot: %s: %w", op, errorFromResponse(resp.StatusCode, raw))
	}
	return raw, nil
}

// fetch performs one API call and decodes its body into T. It exists so the
// request, the status handling and the decode error message are written once
// rather than repeated in every resource method.
func fetch[T any](ctx context.Context, c *Client, op, method string, path endpoint, form url.Values) (T, error) {
	var out T

	raw, err := c.do(ctx, op, method, path, form)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("hrobot: %s: decode response: %w", op, err)
	}
	return out, nil
}

// fetchList performs one call against a collection endpoint and decodes it into
// a slice of T, unwrapping each element from its envelope with pick.
//
// It also owns what an empty collection means, because Robot does not answer
// that consistently. The server and reverse-DNS collections return 200 with an
// empty array, while the IP, key and failover collections return 404 with
// NOT_FOUND. Both are normalised to an empty slice and a nil error, so a caller
// iterating results does not need to know which kind of endpoint it asked.
//
// The normalisation is deliberately narrow. Only NOT_FOUND is treated as empty.
// A 404 carrying a specific code such as SERVER_NOT_FOUND, and a 404 with no
// error document at all, both still fail, so a request aimed at a path that
// does not exist stays loud instead of reading as an empty result.
func fetchList[E, T any](ctx context.Context, c *Client, op string, path endpoint, pick func(E) T) ([]T, error) {
	envelopes, err := fetch[[]E](ctx, c, op, http.MethodGet, path, nil)
	if err != nil {
		if IsError(err, ErrorCodeNotFound) {
			return []T{}, nil
		}
		return nil, err
	}

	out := make([]T, 0, len(envelopes))
	for i := range envelopes {
		out = append(out, pick(envelopes[i]))
	}
	return out, nil
}

// readCapped reads at most maxResponseBytes, and reports a body that exceeds
// it rather than silently returning a truncated document that would then fail
// to parse for a reason the caller cannot diagnose.
func readCapped(r io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseBytes {
		return nil, ErrResponseTooLarge
	}
	return body, nil
}

// endpoint is a request path that has been built safely, and is what [fetch]
// and [fetchList] accept.
//
// The type is the enforcement. Go converts an untyped string constant to a
// named string type implicitly, so a literal path such as "/server" still reads
// naturally at the call site, and a literal carries no caller input to escape.
// An expression involving a string variable is typed string and does not
// convert, so a hand-concatenated path will not compile. Escaping stops being
// something each new endpoint has to remember.
type endpoint string

// scopedEndpoint builds a path for a server-scoped resource, such as
// /boot/321/rescue, after checking the server number.
//
// Rejecting a non-positive number here turns a guaranteed 404 into an immediate
// local error. prefix and suffix are fixed by the caller in this package and
// never carry caller input, which is why only the number is validated.
func scopedEndpoint(prefix string, id int, suffix string) (endpoint, error) {
	if id <= 0 {
		return "", fmt.Errorf("%w, got %d", ErrInvalidServerID, id)
	}
	return endpoint(prefix + "/" + strconv.Itoa(id) + suffix), nil
}

// addressEndpoint builds a path for an address-scoped resource, such as
// /rdns/1.2.3.4, after checking the address and escaping it.
//
// Escaping matters because the address is caller-supplied. Without it a value
// containing a slash addresses a different endpoint than the method name
// promises. url.PathEscape leaves a colon alone, so an IPv6 failover address
// still routes.
func addressEndpoint(prefix, ip string) (endpoint, error) {
	if ip == "" {
		return "", ErrEmptyIP
	}
	return endpoint(prefix + "/" + url.PathEscape(ip)), nil
}
