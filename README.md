# hrobot-go

A Go client for the **Hetzner Robot Webservice**, the API that manages Hetzner's
dedicated (bare-metal) servers.

```go
import "github.com/Foresee-Security/hrobot-go"
```

No dependencies. Go 1.26 or later.

---

> ### Robot is not Hetzner Cloud
>
> These are two unrelated APIs, and confusing them is the first mistake to
> avoid. **Robot** manages physical servers billed monthly at
> `robot-ws.your-server.de` with HTTP basic auth. **Cloud** manages virtual
> machines billed hourly at `api.hetzner.cloud` with a bearer token. They share
> no credentials and no endpoint. If you want Cloud, you want
> [`hcloud-go`](https://github.com/hetznercloud/hcloud-go) instead.

---

## Contents

- [Getting started](#getting-started)
- [Credentials](#credentials)
- [Deadlines and timeouts](#deadlines-and-timeouts)
- [Handling errors](#handling-errors)
- [Common tasks](#common-tasks)
- [Gotchas at a glance](#gotchas-at-a-glance)
- [What is implemented](#what-is-implemented)
- [About this fork](#about-this-fork)
- [Project status](#project-status)
- [Development](#development)

## Getting started

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Foresee-Security/hrobot-go"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	c := hrobot.NewBasicAuthClient("user", "pass")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	servers, err := c.ServerGetList(ctx)
	if err != nil {
		return fmt.Errorf("list servers: %w", err)
	}

	for i := range servers {
		s := &servers[i]
		fmt.Printf("%d  %-20s %-10s %s\n", s.ServerNumber, s.Name, s.DC, s.Status)
	}
	return nil
}
```

The `run` split is not ceremony. `log.Fatal` calls `os.Exit`, which skips
deferred functions, so a `defer cancel()` written beside it never runs. Ranging
by index avoids copying a 200-byte `Server` on every iteration.

The package is named `hrobot`, so the import needs no alias despite the
`hrobot-go` path.

## Credentials

Robot needs a **Webservice user**, created under Settings in the Robot web
interface. It is not your Hetzner account login and not a Cloud API token.

> **Never retry a rejected credential.** Hetzner blocks the calling IP for ten
> minutes after three failed logins, which takes out every other process on that
> address too. Treat `ErrorCodeUnauthorized` as terminal.

To check credentials without assuming the account owns anything:

```go
if err := c.ValidateCredentials(ctx); err != nil {
	return fmt.Errorf("robot credentials rejected: %w", err)
}
```

A nil return means the credentials authenticated. It does not mean the account
owns any servers.

Credentials can be rotated on a live client, safely, while requests are in
flight:

```go
if err := c.SetCredentials(newUser, newPass); err != nil {
	return err   // empty values are rejected and nothing is changed
}
```

## Deadlines and timeouts

Every method that reaches the network takes a `context.Context` first and
honours its deadline and cancellation.

A client from `NewBasicAuthClient` also carries a **30-second per-request
timeout**, so a caller who passes a context without a deadline is still bounded.

```go
c := hrobot.NewBasicAuthClient("user", "pass",
	hrobot.WithTimeout(10*time.Second),
)
```

Options may be given in any order. Supplying your own transport hands the bound
back to you:

```go
c := hrobot.NewBasicAuthClient("user", "pass",
	hrobot.WithHTTPClient(myInstrumentedClient),   // your timeout, your policy
	hrobot.WithTimeout(10*time.Second),            // retimes a copy, not yours
)
```

## Handling errors

Three kinds of failure are distinguishable, and which one you get tells you
where the problem is.

**1. The API rejected the request.** An `Error` carrying Robot's own code. Match
it with `IsError`, which unwraps, so wrapping with `%w` upstream does not break
the match.

```go
_, err := c.ServerGet(ctx, id)
switch {
case hrobot.IsError(err, hrobot.ErrorCodeServerNotFound):
	// not on this account
case hrobot.IsError(err, hrobot.ErrorCodeRateLimitExceeded):
	// back off, see the note below
case err != nil:
	return err
}
```

**2. Something between you and the API failed.** A `StatusError`, produced when
the body was not a Robot error document, which is what a proxy or load balancer
returns. It carries the status so you can tell a retryable 5xx from a terminal
4xx.

```go
var se hrobot.StatusError
if errors.As(err, &se) && se.StatusCode >= 500 {
	// worth retrying
}
```

**3. The call was wrong before it was sent.** A sentinel such as
`ErrInvalidServerID`, `ErrEmptyIP` or `ErrNilInput`. These never reach the
network, so they cost nothing and cannot be rate limited.

> **Rate limits arrive as HTTP 403, not 429.** Code that branches on 429 will
> not see them. There is no built-in retry or backoff.

## Common tasks

### Boot a server into the rescue system

Arming a boot configuration **does not restart anything**. It decides what the
machine boots next. Restarting it is a separate, deliberate step.

```go
rescue, err := c.BootRescueSet(ctx, id, &hrobot.RescueSetInput{
	OS:            "linux",
	AuthorizedKey: fingerprint,   // omit to get a generated password instead
})
if err != nil {
	return err
}

// Capture this now. A later GET on an inactive configuration will not have it.
password := rescue.Password

// Nothing has rebooted yet. This is what enters the rescue system.
if _, err := c.ResetSet(ctx, id, &hrobot.ResetSetInput{
	Type: hrobot.ResetTypeHardware,
}); err != nil {
	return err
}
```

### Choose a reset type the server actually supports

```go
reset, err := c.ResetGet(ctx, id)
if err != nil {
	return err
}
if !slices.Contains(reset.Type, hrobot.ResetTypePower) {
	// this server has no graceful option, decide deliberately
}
```

`ResetTypeManual` emails a data centre technician. It is not automation.
`ResetTypePowerLong` leaves the machine **off** and needs a following
`ResetTypePower` to come back.

### Read a field that changes shape

Boot fields hold the value in force while active, and the menu of choices while
inactive.

```go
rescue, err := c.BootRescueGet(ctx, id)
if err != nil {
	return err
}

if rescue.Active {
	fmt.Println("running:", rescue.OS[0])
} else {
	fmt.Println("available:", strings.Join(rescue.OS, ", "))
}
```

Check `Active` rather than inferring from the length. A menu can legitimately
contain one item.

### Upload an SSH key

```go
key, err := c.KeySet(ctx, &hrobot.KeySetInput{
	Name: "deploy",
	Data: string(pubkey),
})
if hrobot.IsError(err, hrobot.ErrorCodeKeyAlreadyExists) {
	// already there, carry on
}
```

### List things that might not exist

Every list method returns an **empty slice and a nil error** when the account
owns nothing, on every collection. You do not need to special-case it.

```go
keys, err := c.KeyGetList(ctx)   // no keys -> len(keys) == 0, err == nil
if err != nil {
	return err
}
```

This is normalisation on our side. The raw API answers an empty collection two
different ways depending on the endpoint, which is
[documented in detail](docs/BEHAVIOUR.md#4-empty-collections-are-answered-two-different-ways).

## Gotchas at a glance

The full catalogue, with evidence for every claim, is in
**[docs/BEHAVIOUR.md](docs/BEHAVIOUR.md)**. The ones most likely to bite:

| Gotcha | Detail |
|---|---|
| Three failed logins block your IP for ten minutes | [Authentication](docs/BEHAVIOUR.md#2-authentication-and-the-lockout-that-bites-automation) |
| Rate limits are HTTP **403**, not 429 | [Rate limiting](docs/BEHAVIOUR.md#3-rate-limiting-arrives-as-403) |
| Empty collections answer 200 `[]` or 404, per endpoint | [Empty collections](docs/BEHAVIOUR.md#4-empty-collections-are-answered-two-different-ways) |
| `os`, `dist`, `lang`, `arch` change JSON type with state | [Changing types](docs/BEHAVIOUR.md#5-fields-that-change-json-type-with-state) |
| `BootLinuxSet` arms a destructive reinstall on the next boot | [Booting](docs/BEHAVIOUR.md#10-booting-is-two-steps-and-one-of-them-is-destructive) |
| `power_long` leaves the server **off** | [Reset types](docs/BEHAVIOUR.md#11-reset-types-are-per-server-and-one-of-them-involves-a-human) |
| `ResetTypeManual` creates a human ticket | [Reset types](docs/BEHAVIOUR.md#11-reset-types-are-per-server-and-one-of-them-involves-a-human) |
| `Server.Traffic` is a string like `"5 TB"` | [Field surprises](docs/BEHAVIOUR.md#12-field-level-surprises) |
| `Subnet.Mask` is a string, `IP.Mask` is a number | [Field surprises](docs/BEHAVIOUR.md#12-field-level-surprises) |
| Four error-code constants are unverified | [Error codes](docs/BEHAVIOUR.md#8-error-codes) |

## What is implemented

| Area | Methods |
|---|---|
| Server | `ServerGetList` `ServerGet` `ServerSetName` `ServerCancellationWithdraw` |
| Boot | `BootRescueGet/Set/Delete` `BootLinuxGet/Set/Delete` |
| Reset | `ResetGet` `ResetSet` |
| SSH keys | `KeyGetList` `KeySet` |
| IP | `IPGetList` |
| Reverse DNS | `RDNSGetList` `RDNSGet` |
| Failover | `FailoverGetList` `FailoverGet` |
| Client | `ValidateCredentials` `SetCredentials` `GetVersion` |

**Not implemented:** firewall (including the `rules[output]` egress direction),
vSwitch, Storage Box, subnet, traffic, Wake on LAN, and the ordering tree.
Firewall is the one we expect to need first.

`RobotClient` is exported for test doubles and covers everything that reaches
the network. Prefer declaring your own narrower interface naming only the calls
you make. Constructors return the concrete `*Client`, which satisfies both.

## About this fork

Foresee Security's actively developed fork. We use it in production to drive
bare-metal analysis hosts, so we treat it as code we own rather than a
dependency we track. The lineage is nl2go, then
[syself](https://github.com/syself/hrobot-go), then here. Upstream is kept as a
git remote and we merge from it where useful, but we are not constrained by it.

Import `github.com/Foresee-Security/hrobot-go`, not the upstream path.

### Substantive divergence from upstream

**Bugs fixed.** `ServerReverse` called `POST /server/{id}/reversal`, a path the
Robot API does not define. `Cancellation.Reservation` was tagged `reservation`
where the API sends `reserved`, so it never populated. `Failover` dropped
`server_ipv6_net` and `Key` dropped `created_at`. None were catchable by the
old suite, whose fixture server answered every request identically regardless
of method or path.

**Security.** Caller-supplied path segments are escaped, enforced by the type
system rather than convention. Cross-origin redirects are refused, because Go
compares hostnames when deciding whether to resend `Authorization`, which drops
the port and permits an https to http downgrade on the same host. Credentials
are unexported and redacted in `String` and `LogValue`. Response bodies are
capped at 8 MiB.

**API.** One package instead of `client` plus `models`. Context on every call.
A 30-second default timeout where a bare `http.Client` previously meant none.
Arguments validated before any request. Errors wrapped with the operation and
matched with `errors.As`. No `any` in the exported surface. Construction
options instead of setters that mutated a live client.

## Project status

Honest summary rather than a badge.

**The code is production-grade.** Zero findings across 38 linters, no
`//nolint`, no TODOs, no dependencies, 97% statement coverage where the
guarantees are verified by mutation rather than by line count.

**It is not production-proven.** Three things a reader should weigh:

1. **It has never run against a real dedicated server.** Measurements were taken
   on an account owning zero of them, so every server-scoped success path rests
   on documentation and fixtures. The failure paths were confirmed live.
2. **Firewall endpoints are absent**, and they are the ones we need first.
3. **There is no CI.** The gate is reproducible but runs by hand.

## Development

`make check` runs the same gate Voltz runs against its own Go components, with
the same pinned tools: `go vet`, `go build`, `golangci-lint` across 38 linters,
`go test -race`, `govulncheck` and `nilaway`.

```sh
make check      # the full gate
make test       # go test -race
make cover      # coverage report
```

nilaway reports one finding here, a false positive on `http.Client.Do` tracked
as nilaway issue #126, which the `nilaway` target filters by the same narrow
rule Voltz uses. It suppresses only that pattern, prints the count so the
exemption cannot go quiet, and fails on anything else.

Coverage is not the measure we trust. At 96% the suite still could not tell
whether the default timeout, the transport seam or the credential guard existed
at all, since deleting any of the three left the whole gate green. They are
pinned now. When changing behaviour, check that a test fails when you break it,
rather than checking that the percentage held.

### Contributing

Issues and pull requests welcome. Where a change is generally useful and not
specific to how we operate, we would rather send it upstream than keep it here.

If you observe a behaviour that contradicts [docs/BEHAVIOUR.md](docs/BEHAVIOUR.md),
that is a valuable report. Several entries there are marked unverified
specifically so they can be confirmed or removed.

### Releasing

Update `Version` in `client.go`, then:

```sh
make check

export RELEASE_TAG=vX.Y.Z
git tag -a ${RELEASE_TAG} -m ${RELEASE_TAG}
git push origin ${RELEASE_TAG}
```

## Licence

MIT, inherited from upstream. See [LICENSE](LICENSE).
