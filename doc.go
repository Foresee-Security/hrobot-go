// Package hrobot is a client for the Hetzner Robot Webservice, the API that
// manages dedicated (bare-metal) servers.
//
// It is distinct from the Hetzner Cloud API. Robot manages physical machines
// billed monthly, addressed by server number, and authenticated with a
// dedicated Webservice user over HTTP basic auth. Cloud manages virtual
// machines billed hourly and authenticated with a bearer token. The two share
// neither credentials nor an endpoint.
//
// # Usage
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
// # Errors
//
// A response with a 4xx or 5xx status decodes into an [Error] carrying the
// API's own machine-readable code. Match it with [IsError], which unwraps, so
// adding context with %w upstream does not break the match.
//
//	_, err := c.ServerGet(ctx, id)
//	if IsError(err, ErrorCodeServerNotFound) {
//		// the server number is not on this account
//	}
//
// Arguments are validated before any request is made, so a misuse fails
// immediately and locally rather than after a network round trip. Those
// failures match the sentinels declared in this package, such as
// [ErrInvalidServerID], [ErrEmptyIP] and [ErrNilInput].
//
// # Deadlines
//
// Every method that performs I/O takes a [context.Context] as its first
// argument and honours its deadline and cancellation. A client built by
// [NewBasicAuthClient] additionally carries a 30-second per-request timeout, so
// a caller that passes a context without a deadline is still bounded.
//
// Supplying your own transport with [WithHTTPClient] hands that bound back to
// you. Nothing is imposed on a client passed that way, so pair it with
// [WithTimeout] or a context deadline.
//
// # Array-or-scalar fields
//
// Several Robot responses change a field's JSON type with the resource's state.
// Boot configuration returns "os" as a string while the rescue system is
// active, and as an array of the available choices while it is not. Those
// fields are typed [StringList] and [IntList], which decode both shapes into a
// slice, so callers never type-assert. A single-element slice means that value
// is the active one.
//
// # Divergence from upstream
//
// This is Foresee Security's fork of github.com/syself/hrobot-go, itself a fork
// of nl2go/hrobot-go. It is developed against the published Robot Webservice
// documentation rather than tracking upstream, and its exported surface has
// already diverged. See README.md.
package hrobot
