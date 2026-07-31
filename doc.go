// Package hrobot is a client for the Hetzner Robot Webservice, the API that
// manages dedicated (bare-metal) servers.
//
// # Robot is not Hetzner Cloud
//
// These are two unrelated APIs and mixing them up is the first mistake to
// avoid. Robot manages physical machines, billed monthly, addressed by server
// number, authenticated with HTTP basic auth against robot-ws.your-server.de.
// Cloud manages virtual machines, billed hourly, addressed by an integer id,
// authenticated with a bearer token against api.hetzner.cloud. They share no
// credentials and no endpoint, and hcloud-go is not a substitute for this
// package.
//
// # Credentials
//
// Robot needs a dedicated Webservice user, created under Settings in the Robot
// web interface. It is neither the Hetzner account login nor a Cloud API token.
//
// Authentication failures are rate limited by Hetzner at the network level.
// Three failed logins block the calling IP for ten minutes, so a service that
// retries a rejected credential in a loop will lock itself out. Treat
// [ErrorCodeUnauthorized] as terminal.
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
// also carries a 30-second per-request timeout, so a caller who passes a
// context without a deadline is still bounded.
//
// Supplying a transport with [WithHTTPClient] hands that bound back to you.
// Nothing is imposed on a client passed that way. Pair it with [WithTimeout],
// which retimes a copy rather than the client you supplied, or with a context
// deadline.
//
// Options may be given in any order.
//
// # Errors
//
// Three kinds of failure are distinguishable, and which one you get says where
// the problem is.
//
// An [Error] is the Robot API rejecting the request, carrying its own
// machine-readable [ErrorCode]. Match it with [IsError], which unwraps, so
// wrapping with %w upstream does not break the match.
//
//	_, err := c.ServerGet(ctx, id)
//	if hrobot.IsError(err, hrobot.ErrorCodeServerNotFound) {
//		// the server number is not on this account
//	}
//
// A [StatusError] is an HTTP failure whose body was not a Robot error document,
// which is what an intermediary such as a proxy or load balancer produces. It
// carries the status so a caller can still tell a retryable 5xx from a terminal
// 4xx.
//
// A sentinel such as [ErrInvalidServerID], [ErrEmptyIP] or [ErrNilInput] is
// this package rejecting the call before any request is built. Those never
// reach the network, so they cost nothing and cannot be rate limited.
//
// Every error names the operation that produced it, so a decode failure three
// calls deep says which call it was.
//
// # Rate limiting
//
// Robot reports a rate limit as [ErrorCodeRateLimitExceeded] with HTTP status
// 403, not the 429 most APIs use. The response also carries max_request and
// interval fields describing the budget, which this client does not currently
// decode. There is no built-in retry or backoff. A caller that polls should
// treat this code as a signal to back off rather than retry immediately.
//
// # Empty collections
//
// Robot is not consistent about what an empty collection looks like. Measured
// against the live API, the server and reverse-DNS collections answer 200 with
// an empty array, while the IP, key and failover collections answer 404 with
// [ErrorCodeNotFound].
//
// This package normalises both, so every list method returns an empty slice and
// a nil error for an account that owns nothing. The normalisation is narrow on
// purpose: only NOT_FOUND counts as empty. A 404 carrying a specific code, and
// a 404 with no error document at all, both still fail, so a request aimed at a
// path that does not exist stays loud rather than reading as an empty result.
//
// # Fields that change JSON type
//
// Several boot fields change shape with the resource's state. While the rescue
// system is active, "os" is the string naming the running system. While it is
// inactive, "os" is the array of systems that could be booted. The same applies
// to "dist", "lang" and "arch", and to a cancellation's "cancellation_reason".
//
// Those fields are typed [StringList] and [IntList], which decode both shapes
// into a slice, so callers never type-assert. A one-element list means that
// value is the active one.
//
//	rescue, err := c.BootRescueGet(ctx, id)
//	if rescue.Active {
//		running := rescue.OS[0]      // the active system
//	} else {
//		available := rescue.OS       // everything bootable
//	}
//
// # Booting is two steps
//
// Arming the rescue system or an installation does not restart anything.
// [Client.BootRescueSet] and [Client.BootLinuxSet] only set what the server
// will boot next. The machine keeps running until [Client.ResetSet] restarts
// it.
//
// That separation is deliberate and it is also the sharp edge.
// [Client.BootLinuxSet] arms a destructive reinstall that runs on the next
// boot, so arming it and later resetting the server for an unrelated reason
// wipes the machine.
//
// # Redirects
//
// A client from [NewBasicAuthClient] refuses to follow a redirect that changes
// scheme, host or port, returning [ErrRedirectCrossOrigin].
//
// This is not paranoia about a redirect that Robot does not publish. The
// standard library decides whether to resend an Authorization header by
// comparing hostnames, which drops the port and ignores the scheme, so a
// redirect from https to http on the same host still carries the credentials,
// in cleartext. A client supplied through [WithHTTPClient] keeps the standard
// library behaviour, because its policy belongs to whoever built it.
//
// # Concurrency
//
// [Client] is safe for concurrent use. The endpoint, user agent and http.Client
// are fixed at construction, and the credentials are guarded, so
// [Client.SetCredentials] may rotate them while requests are in flight.
//
// # Limits
//
// Response bodies are capped at 8 MiB, and a body over the cap reports
// [ErrResponseTooLarge] rather than being truncated into a parse failure nobody
// can diagnose.
//
// # Coverage of the API
//
// Implemented: server, boot (rescue and linux), reset, SSH key, IP, reverse DNS
// and failover. Absent: firewall, vSwitch, Storage Box, subnet, traffic, Wake
// on LAN, and the ordering tree.
//
// # This fork
//
// Foresee Security's fork of github.com/syself/hrobot-go, itself a fork of
// nl2go/hrobot-go. It is developed against the published Robot Webservice
// documentation rather than tracking upstream, and its exported surface has
// already diverged. See README.md and docs/BEHAVIOUR.md.
package hrobot
