# hrobot-go: a Go client for the Hetzner Robot Webservice

The Robot Webservice manages Hetzner's dedicated (bare-metal) servers. It is a
different API from Hetzner Cloud, with different credentials and a different
endpoint, so `hcloud-go` is not a substitute for it.

Hetzner's own API documentation lives at
[robot.your-server.de](https://robot.your-server.de/doc/webservice/en.html).

```go
import "github.com/Foresee-Security/hrobot-go"

c := hrobot.NewBasicAuthClient("user", "pass")

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

servers, err := c.ServerGetList(ctx)
if err != nil {
    return fmt.Errorf("list servers: %w", err)
}
```

Credentials are a **Webservice user**, created under Settings in the Robot web
interface. They are not the Hetzner account login and not a Cloud API token.

## About this fork

This is Foresee Security's actively developed fork. We use it in production to
drive bare-metal analysis hosts, so we treat it as code we own rather than as a
dependency we track. The lineage is nl2go, then
[syself](https://github.com/syself/hrobot-go), then here. Upstream is kept as a
git remote and we will merge from it where that is useful, but we are not
constrained by it, and the exported surface has already diverged.

Import `github.com/Foresee-Security/hrobot-go`, not the upstream path. The
package is named `hrobot`, so no import alias is needed.

## Quality gate

`make check` runs the same gate Voltz runs against its own Go components, with
the same pinned tools: `go vet`, `go build`, `golangci-lint` 2.12.2 across 38
linters, `go test -race`, `govulncheck`, and `nilaway`, on Go 1.26.5.

It passes with zero lint issues, zero vulnerabilities, and 97% statement
coverage. nilaway reports one finding, a false positive on `http.Client.Do`
tracked as nilaway issue #126, which the `nilaway` target filters by the same
narrow rule Voltz uses. The filter suppresses only that pattern, prints the
count so the exemption cannot go quiet, and fails on anything else.

Coverage is not the measure we trust here. At 96% the suite still could not
tell whether the default timeout, the transport seam or the credential guard
existed at all: deleting any of the three left the whole gate green. Those are
pinned now, and the checks that matter are mutation checks rather than the
percentage.

The module has **no dependencies**. There is no `go.sum`.

The client has also been run against the live Robot Webservice on a real
account, which is the only way to settle behaviour fixtures can only assume.
That run confirmed the error codes this client keys on, that a listing with
nothing in it comes back as 200 with an empty array rather than a 404, and that
a rejected argument never reaches the network.

## What changed from upstream

### Bugs

Four defects were found by checking the code against Hetzner's published API
documentation. The previous test suite could not catch any of them, because its
fixture server answered every request with the same document regardless of
method or path, so nothing verified where a request actually went.

- **`ServerReverse` called an endpoint that does not exist.** It issued
  `POST /server/{id}/reversal`. The Robot Webservice defines no `reversal`
  path. Withdrawing a cancellation is `DELETE /server/{id}/cancellation`. The
  method is now `ServerCancellationWithdraw` and calls that.
- **`Cancellation.Reservation` never populated.** It was tagged `reservation`.
  The API sends `reserved`. The field is now `Reserved`, tagged to match.
- **`Failover` dropped `server_ipv6_net`.** The field was absent from the
  struct, so the value was discarded for every failover address.
- **`Key` dropped `created_at`.** Same cause, same effect.

### Security

- **Path segments built from caller input are now escaped, and the compiler
  enforces it.** `RDNSGet` and `FailoverGet` interpolated their argument
  straight into the URL, so a value containing a slash addressed a different
  endpoint than the method named. Request paths are now an `endpoint` type
  built by one of two constructors. A literal still converts implicitly, so
  `"/server"` reads normally, but a hand-concatenated path no longer compiles.
- **Redirects to another origin are refused.** Go decides whether to resend
  `Authorization` by comparing hostnames, which drops the port and ignores the
  scheme, so credentials were replayed to a redirect target on a different port
  and would survive an https to http downgrade on the same host. A redirect
  that stays on the configured origin is still followed.
- **Credentials are unexported and redacted.** `Username` and `Password` were
  exported fields, so any `%+v` wrote the password wherever that landed.
  `String` and `LogValue` mask it, and redaction lives on the type rather than
  asking every call site to remember.
- **Response bodies are capped at 8 MiB** and an oversize body is reported as
  `ErrResponseTooLarge` rather than truncated into a parse failure nobody can
  diagnose.

### API

- **One package.** `models` is gone and its types live in the root package, so
  a caller has one import instead of two. The wire envelope types
  (`ServerResponse`, `KeyResponse`, and the rest) are unexported, since they
  were never anything a caller needed.
- **Context on every call.** Every method that performs I/O takes a
  `context.Context` first and builds its request with
  `http.NewRequestWithContext`. Nothing here could previously be cancelled.
- **Bounded by default.** `NewBasicAuthClient` built a bare `&http.Client{}`,
  which in Go means no timeout at all. It now carries 30 seconds.
- **Arguments validated before the request.** A non-positive server ID, an
  empty IP, or a nil input struct returns a sentinel error immediately instead
  of a guaranteed 404, or, for the nil input, a panic across the API boundary.
- **Errors carry the operation and wrap.** Every returned error names the call
  that produced it. `IsError` uses `errors.As`, so adding context with `%w`
  does not break code matching. A failure whose body is not a Robot error
  document, which is what a proxy produces, is now a typed `StatusError`
  carrying the status rather than an unmatchable string.
- **No `any` in the public surface.** Six fields were `any` because the API
  returns `os`, `arch`, `dist`, `lang` and `cancellation_reason` as a bare
  scalar while a resource is active and as an array of the available choices
  while it is not. They are now `StringList` and `IntList`, which decode both
  shapes into a slice. A one-element list means that value is the active one.
- **Options instead of setters, and they commute.** `SetBaseURL` and
  `SetUserAgent` mutated a live client. They are now `WithBaseURL` and
  `WithUserAgent` construction options, alongside `WithHTTPClient` and
  `WithTimeout`. The order you write them in does not change the result, and a
  client you supply is never retimed underneath you. `SetCredentials` stays,
  guarded, so credentials can be rotated on a running client.
- **Empty collections look the same everywhere.** Robot answers an empty
  collection with 200 and an empty array on some endpoints and 404 on others,
  so three of the five list methods used to return an error to describe an
  account that simply owns nothing. They all return an empty slice now. The
  normalisation is narrow: only `NOT_FOUND` counts as empty, so a request
  aimed at a path that does not exist still fails.
- **Constructors return `*Client`,** not the interface, following "accept
  interfaces, return structs". `RobotClient` still exists for test doubles,
  covering what a substitute can meaningfully stand in for. A test asserts
  every exported method on `*Client` is either declared there or listed as
  excluded with a reason, so the interface cannot quietly drift out of date.
- **Acronyms are cased correctly:** `Rdns` is `RDNS`, `Os` is `OS`, `Dc` is
  `DC`, `Vnc` is `VNC`, `Wol` is `WOL`, `SeparateMac` is `SeparateMAC`. Reset
  types are a typed `ResetType` rather than bare strings.
- **`Subnet` carries only `ip` and `mask`,** the two fields the server
  endpoints actually return. The other nine were never populated.

### Tests

The suite was rewritten from `gopkg.in/check.v1` onto the standard library,
which removed the last three dependencies. It is table-driven, asserts the
method and path of every request, and asserts that a rejected argument produces
**no request at all**. Coverage went from 90.8% to 97.2%.

The request assertions are the part that matters. The previous suite pointed
every test at a fixture server that answered any request with the same
document, so it could only ever verify decoding. That is how a method calling
an endpoint which does not exist passed its own test for years.

Where a guarantee is not observable from outside the package, such as how the
constructor wires the `http.Client`, a small in-package test covers it and
everything else stays external.

## Not implemented

The client covers server, boot, reset, key, IP, reverse DNS and failover. The
firewall, vSwitch, Storage Box, traffic, subnet, Wake on LAN and ordering
endpoints are absent. Firewall is the one we expect to need first.

Contributions and issues are welcome. Where a change is generally useful and
not specific to how we operate, we would rather send it upstream than keep it
here.

## Releasing

Update `Version` in `client.go`, then:

```sh
make check

export RELEASE_TAG=vX.Y.Z
git tag -a ${RELEASE_TAG} -m ${RELEASE_TAG}
git push origin ${RELEASE_TAG}
```
