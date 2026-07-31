// Package hrobot is a client for the Hetzner Robot Webservice, the API that
// manages dedicated (bare-metal) servers.
//
// # Relationship to Hetzner Cloud
//
// Robot and Hetzner Cloud are separate APIs. Robot manages physical machines,
// billed monthly, addressed by server number, authenticated with HTTP basic
// auth against robot-ws.your-server.de. Cloud manages virtual machines, billed
// hourly, authenticated with a bearer token against api.hetzner.cloud. The
// credentials and endpoints are not interchangeable, and hcloud-go does not
// talk to Robot.
//
// # Credentials
//
// Robot uses a Webservice user, created under Settings in the Robot web
// interface. This is a separate credential from the Hetzner account login and
// from any Cloud API token.
//
// Hetzner blocks the calling IP for ten minutes after three failed logins. A
// service that retries a rejected credential will lock itself out, along with
// anything else running on that address. Treat [ErrorCodeUnauthorized] as
// terminal.
//
//	c := hrobot.NewBasicAuthClient("user", "pass")
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	servers, err := c.ServerGetList(ctx)
//	if err != nil {
//		return fmt.Errorf("list servers: %w", err)
//	}
//
// # Deadlines
//
// Every method that reaches the network takes a [context.Context] first and
// honours its deadline and cancellation. A client from [NewBasicAuthClient]
// also carries a 30-second per-request timeout, so a context without a
// deadline is still bounded.
//
// [WithHTTPClient] moves that responsibility to the caller. No timeout is
// imposed on a client passed that way. Use [WithTimeout], which retimes a copy
// rather than the client you supplied, or set a context deadline. Options can
// be given in any order.
//
// # Errors
//
// An [Error] is the API rejecting the request. It carries Robot's own
// [ErrorCode]. Match it with [IsError], which unwraps, so wrapping with %w
// upstream does not break the match.
//
//	_, err := c.ServerGet(ctx, id)
//	if hrobot.IsError(err, hrobot.ErrorCodeServerNotFound) {
//		// the server number is not on this account
//	}
//
// A [StatusError] is an HTTP failure whose body was not a Robot error
// document, which is what a proxy or load balancer returns. It carries the
// status code, so a 5xx stays distinguishable from a 4xx.
//
// A sentinel such as [ErrInvalidServerID], [ErrEmptyIP] or [ErrNilInput] means
// the arguments were rejected before a request was built. These never reach
// the network.
//
// Errors name the operation that produced them, so a decode failure several
// calls deep identifies the call it came from.
//
// # Rate limiting
//
// Robot reports a rate limit as [ErrorCodeRateLimitExceeded] with HTTP status
// 403, where most APIs use 429. The response also carries max_request and
// interval fields describing the budget, which this client does not decode.
// There is no built-in retry or backoff.
//
// # Empty collections
//
// Robot answers an empty collection differently depending on the endpoint.
// Measured against the live API, the server and reverse-DNS collections return
// 200 with an empty array. The IP, key and failover collections return 404
// with [ErrorCodeNotFound].
//
// This package normalises both, so every list method returns an empty slice
// and a nil error for an account that owns nothing. Only NOT_FOUND is treated
// as empty. A 404 carrying a more specific code, and a 404 with no error
// document, both still return an error, so a request to a path that does not
// exist does not read as an empty result.
//
// # Fields that change JSON type
//
// Several boot fields change shape with the state of the resource. While the
// rescue system is active, "os" is the string naming the running system. While
// it is inactive, "os" is an array of the systems that could be booted. The
// same applies to "dist", "lang" and "arch", and to a cancellation's
// "cancellation_reason".
//
// These fields are typed [StringList] and [IntList], which decode both shapes
// into a slice. A one-element slice means that value is the active one.
//
//	rescue, err := c.BootRescueGet(ctx, id)
//	if rescue.Active {
//		running := rescue.OS[0]      // the active system
//	} else {
//		available := rescue.OS       // everything bootable
//	}
//
// # Boot configuration does not restart the server
//
// [Client.BootRescueSet] and [Client.BootLinuxSet] set what the server will
// boot next. The machine keeps running its current system until
// [Client.ResetSet] restarts it.
//
// [Client.BootLinuxSet] arms a reinstall that erases the disk when it runs. It
// stays armed until deleted, so a restart triggered for any other reason will
// run it.
//
// # Redirects
//
// A client from [NewBasicAuthClient] does not follow a redirect that changes
// scheme, host or port. It returns [ErrRedirectCrossOrigin] instead.
//
// The standard library decides whether to resend an Authorization header by
// comparing hostnames, which excludes the port and ignores the scheme. Under
// that rule a redirect from https to http on the same host still carries the
// credentials. A client supplied through [WithHTTPClient] keeps the standard
// library behaviour.
//
// # Concurrency
//
// [Client] is safe for concurrent use. The endpoint, user agent and http.Client
// are fixed at construction, and the credentials are mutex-guarded, so
// [Client.SetCredentials] can rotate them while requests are in flight.
//
// # Response size
//
// Response bodies are read up to 8 MiB. A larger body returns
// [ErrResponseTooLarge] rather than a truncated document.
//
// # API coverage
//
// Implemented: server, boot (rescue and linux), reset, SSH key, IP, reverse DNS
// and failover. Not implemented: firewall, vSwitch, Storage Box, subnet,
// traffic, Wake on LAN and the ordering tree.
//
// # This fork
//
// Foresee Security's fork of github.com/syself/hrobot-go, which is itself a
// fork of nl2go/hrobot-go. It is developed against the published Robot
// Webservice documentation rather than tracking upstream, and its exported
// surface has diverged. See README.md and docs/BEHAVIOUR.md.
package hrobot
