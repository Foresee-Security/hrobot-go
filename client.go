package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Foresee-Security/hrobot-go/models"
)

const (
	baseURL   string = "https://robot-ws.your-server.de"
	version          = "0.2.6"
	userAgent        = "hrobot-client/" + version

	// defaultTimeout bounds a request whose caller supplied no deadline of its
	// own. A zero-value http.Client has no timeout whatsoever, so without this
	// a single unresponsive request blocks its caller until the process dies.
	// Callers that know better should pass a context with a deadline, which
	// takes precedence over this.
	defaultTimeout = 30 * time.Second

	// maxResponseBytes caps how much of a response body is read into memory.
	// The Robot API returns small JSON documents. Anything larger is a broken
	// or hostile endpoint rather than a response worth parsing.
	maxResponseBytes = 8 << 20 // 8 MiB
)

// ErrResponseTooLarge is returned when a response body exceeds maxResponseBytes.
var ErrResponseTooLarge = errors.New("hrobot: response body exceeds maximum size")

// Client talks to the Hetzner Robot Webservice.
//
// Not safe for concurrent modification: SetBaseURL, SetUserAgent and
// SetCredentials mutate the client and must not race with in-flight requests.
// Concurrent requests on an otherwise unmodified client are safe, since the
// underlying http.Client is.
type Client struct {
	Username   string
	Password   string
	baseURL    string
	userAgent  string
	httpClient *http.Client
}

// String redacts the credentials so a client cannot leak its password through
// a stray %v or %+v. Redaction lives on the type rather than at call sites,
// because a call site that forgets is exactly the case that leaks.
func (c *Client) String() string {
	return fmt.Sprintf("hrobot.Client{Username: %q, Password: REDACTED, baseURL: %q}", c.Username, c.baseURL)
}

// LogValue is the structured-logging counterpart to String, so slog output
// redacts the password too.
func (c *Client) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("username", c.Username),
		slog.String("password", "REDACTED"),
		slog.String("base_url", c.baseURL),
	)
}

func NewBasicAuthClient(username, password string) RobotClient {
	return &Client{
		Username:   username,
		Password:   password,
		baseURL:    baseURL,
		userAgent:  userAgent,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// NewBasicAuthClientWithCustomHTTPClient builds a client on a caller-supplied
// http.Client. The caller owns its timeout: nothing is imposed here, because
// overriding a deliberately configured transport would be worse than the
// missing bound.
func NewBasicAuthClientWithCustomHTTPClient(username, password string, httpClient *http.Client) RobotClient {
	return &Client{
		Username:   username,
		Password:   password,
		baseURL:    baseURL,
		userAgent:  userAgent,
		httpClient: httpClient,
	}
}

func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

func (c *Client) SetUserAgent(userAgent string) {
	c.userAgent = userAgent
}

func (c *Client) GetVersion() string {
	return version
}

func (c *Client) ValidateCredentials(ctx context.Context) error {
	if _, err := c.doGetRequest(ctx, c.baseURL); err != nil {
		return err
	}
	return nil
}

func (c *Client) SetCredentials(username, password string) error {
	if username == "" {
		return errors.New("username cannot be empty")
	}
	if password == "" {
		return errors.New("password cannot be empty")
	}
	c.Username = username
	c.Password = password
	return nil
}

func (c *Client) doGetRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (c *Client) doDeleteRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (c *Client) doPostFormRequest(ctx context.Context, url string, formData url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", c.userAgent)
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := readCapped(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		return nil, errorFromResponse(resp, body)
	}
	return body, nil
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

func errorFromResponse(resp *http.Response, body []byte) (reterr error) {
	var errorResponse models.ErrorResponse
	reterr = fmt.Errorf("server responded with status code %v", resp.StatusCode)
	if err := json.Unmarshal(body, &errorResponse); err != nil {
		return
	}
	if errorResponse.Error.Code == "" && errorResponse.Error.Message == "" {
		return
	}
	return errorResponse.Error
}
