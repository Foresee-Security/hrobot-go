# Robot Webservice behaviour

How the Hetzner Robot Webservice behaves, as distinct from how this client is
written. Read it before relying on an assumption about the API.

Claims are labelled by source:

| Label | Meaning |
|---|---|
| measured | Observed against the live API on a real account. |
| documented | Stated in Hetzner's published Robot documentation. |
| unverified | Inherited from upstream or inferred. Unconfirmed. |

Measurements were taken on 2026-07-30 and 2026-07-31 against an account that
owns no dedicated servers. Everything server-scoped here therefore comes from
documentation and fixtures rather than observation. See
[section 14](#14-what-has-not-been-exercised).

---

## 1. Robot and Cloud are separate APIs

documented.

| | Robot | Cloud |
|---|---|---|
| Manages | physical servers | virtual machines |
| Billing | monthly, with a setup fee | hourly |
| Endpoint | `robot-ws.your-server.de` | `api.hetzner.cloud` |
| Auth | HTTP basic, Webservice user | bearer token |
| Go client | this package | `hcloud-go` |

A Cloud token will not authenticate to Robot.

## 2. Failed logins block your IP

documented. The Webservice user is created under Settings in the Robot web
interface. It is not the Hetzner account login.

Three failed logins block the calling IP for ten minutes. Any process retrying
a rejected credential will lock out every other process on that address, so
`UNAUTHORIZED` should be handled as terminal rather than retried.

measured. A 401 is returned before the API looks for the resource, so any 404
received proves the credentials were accepted. `Client.ValidateCredentials`
relies on this when it treats a not-found listing as success.

measured. The username on our account is prefixed with `#`. Config validation
should not assume a plain identifier.

## 3. Rate limits use status 403

documented. `RATE_LIMIT_EXCEEDED` comes back with HTTP 403 rather than the 429
most APIs use, so code branching on 429 will miss it.

The response includes the budget:

```json
{"error": {"status": 403, "code": "RATE_LIMIT_EXCEEDED",
           "max_request": 200, "interval": 3600,
           "message": "rate limit exceeded"}}
```

This client does not decode `max_request` or `interval`, so the budget is not
available to callers. `Error` also drops `status`. There is no built-in retry
or backoff.

## 4. Empty collections have two different shapes

measured.

| Endpoint | Empty response |
|---|---|
| `/server` | 200 with `[]` |
| `/rdns` | 200 with `[]` |
| `/vswitch` | 200 with `[]` |
| `/ip` | 404 `NOT_FOUND` |
| `/key` | 404 `NOT_FOUND`, message "No keys found" |
| `/failover` | 404 `NOT_FOUND` |
| `/subnet` | 404 `NOT_FOUND` |
| `/storagebox` | 404 `STORAGEBOX_NOT_FOUND` |

Code that assumes one shape for "no rows" will be wrong on about half these
endpoints.

`/key` returns the generic `NOT_FOUND` rather than a key-specific code, so the
error alone does not distinguish an account with no keys from a request to a
path that does not exist.

This client normalises both shapes to an empty slice with a nil error. Only
`NOT_FOUND` is treated as empty. A 404 with a more specific code, or with no
error document, still returns an error.

## 5. Some fields change JSON type with state

documented and measured. Several fields are `Array|Scalar` depending on whether
the resource is active.

| Field | Active | Inactive |
|---|---|---|
| `os` (rescue) | `"linux"` | `["linux","freebsd","vkvm"]` |
| `dist` (linux) | `"CentOS 5.5 minimal"` | `["CentOS 5.5 minimal", ...]` |
| `lang` (linux) | `"en"` | `["en"]` |
| `arch` (both) | `64` | `[64,32]` |
| `cancellation_reason` | `null` once cancelled | array of selectable reasons |

The meaning changes with the shape. A scalar is the value currently in force.
An array lists the available options.

This client decodes both into `StringList` and `IntList`. Read `Active` to know
which meaning applies, because a list of options can contain one item and is
then indistinguishable from an active value by length alone.

## 6. Deprecated fields still appear in responses

documented. `arch` on the boot endpoints is marked `@deprecated` on both input
and output, and is still returned. This client decodes it and marks it
`Deprecated:` in godoc. New code should not depend on it.

## 7. Undocumented shapes

unverified. `host_key` on the boot endpoints is typed only as "Array". Every
published example and every fixture we hold is empty, so the element shape is
unknown. This client surfaces it as `[]json.RawMessage` rather than guessing at
a struct. A non-empty `host_key` observed in the wild is worth recording.

"Array" carries no information about elements in this API. `authorized_key` is
also documented as "Array" and its elements are `{"key": {...}}` objects.

## 8. Error codes

documented. Eighteen codes appear in the published error tables:

```
RATE_LIMIT_EXCEEDED   CONFLICT              NOT_FOUND            INVALID_INPUT
SERVER_NOT_FOUND      IP_NOT_FOUND          SUBNET_NOT_FOUND     RDNS_NOT_FOUND
RESET_NOT_AVAILABLE   RESET_MANUAL_ACTIVE   RESET_FAILED
BOOT_NOT_AVAILABLE    BOOT_ACTIVATION_FAILED    BOOT_DEACTIVATION_FAILED
KEY_ALREADY_EXISTS    KEY_CREATE_FAILED     KEY_UPDATE_FAILED    KEY_DELETE_FAILED
```

unverified. Four more are inherited from upstream and appear in no published
error table:

```
UNAUTHORIZED    INTERNAL_ERROR    BOOT_ALREADY_ENABLED    BOOT_BLOCKED
```

They are kept, since deleting one would break a caller that does receive it,
but a match on them is unconfirmed. They sit in a separate `const` block in
`error.go` that says so. A constant matching a code the API never sends fails
silently and permanently.

If you observe one of the four in a real response, move it to the verified
block and record where you saw it.

Codes without a constant here are preserved verbatim, so a code Hetzner adds
later stays matchable without a release.

## 9. Endpoints that do not exist

documented by absence. There is no `POST /server/{id}/reversal`. Upstream
shipped a `ServerReverse` method calling it. Withdrawing a cancellation is
`DELETE /server/{id}/cancellation`, which `ServerCancellationWithdraw` calls.

measured. The `/order/*` tree returns 401 UNAUTHORIZED on our account although
the same credentials work on every other endpoint. The cause is undetermined.
Ordering may be a separate entitlement, or an account may need to place its
first order through the web frontend before API ordering unlocks. Do not retry
these calls in a loop, given the lockout in section 2.

## 10. Arming a boot configuration does not restart the server

documented.

`BootRescueSet` and `BootLinuxSet` decide what the machine boots next. The
running system is unaffected until something restarts it.

Three consequences:

Entering the rescue system takes a following `ResetSet`.

`BootLinuxSet` arms an installation that erases the disk. It stays armed until
deleted, so a restart for any unrelated reason, a power event, a hardware
reset, an operator, runs the installer.

`password` is only populated while the configuration is active. Capture it from
the response to the arming call, since a later read of an inactive
configuration will not carry it.

## 11. Reset types vary per server

documented. `ResetGet` reports which types a given server supports.

| Type | Effect | Support |
|---|---|---|
| `power` | ACPI power button. Graceful shutdown, or power on if off. | some servers |
| `power_long` | Held power button. Immediate cut, server stays off. | some servers |
| `sw` | CTRL+ALT+DEL. | almost all |
| `hw` | Hardware reset button. Ungraceful. | all |
| `man` | Emails a data centre technician to recycle power by hand. | all |

`power_long` leaves the machine off, and a following `power` is needed to bring
it back. `man` creates a human ticket rather than an API action, takes time,
and returns `RESET_MANUAL_ACTIVE` if one is already running.

## 12. Field-level surprises

documented and measured.

`Server.Traffic` is a string such as `"5 TB"` or `"unlimited"`, not a number.

`Server.Subnet` can be `null` rather than an empty array, decoding to a nil
slice.

`Subnet` nested in a server response carries only `ip` and `mask`. The gateway,
traffic and locking fields belong to the standalone subnet endpoints, which
this client does not implement.

`Subnet.Mask` is a string (`"64"`) while `IP.Mask` is a number (`27`).

`Key.Data` is empty when a key appears nested as an `AuthorizedKey` in a boot
response. Only metadata is included there.

`Key.CreatedAt` uses `"2006-01-02 15:04:05"` rather than RFC 3339, and date
fields elsewhere are `"2006-01-02"` strings rather than timestamps.

`IP.SeparateMAC` is `null` when unset and decodes to an empty string.

`Cancellation` uses the JSON name `reserved`. Upstream tagged the field
`reservation`, so it never populated.

`Failover.ServerIP` and `Failover.ActiveServerIP` differ once an address has
been switched. The first names the owning server, the second names where
traffic currently goes.

## 13. Behaviour this client changes

The following are decisions in this package rather than API behaviour.

| API | This client |
|---|---|
| Empty collection is 200 `[]` or 404 by endpoint | always an empty slice |
| A redirect can replay basic auth to another port or over http | cross-origin redirects refused |
| Caller input concatenated into a URL path | escaped, enforced by the type system |
| No request bound | 30s default timeout, context honoured |
| Unbounded response read | 8 MiB cap |
| Error is untyped JSON | `Error` with a matchable code, or `StatusError` |
| A bad server number costs a round trip | rejected before the request is built |

## 14. What has not been exercised

The account used for measurement owns no dedicated servers. Every server-scoped
path in this client, `ServerGet`, `ServerSetName`,
`ServerCancellationWithdraw`, the six boot methods and both reset methods, has
run against fixtures and documentation only. The 404 paths were confirmed live.
The success paths were not.

Not implemented: firewall (including the `rules[output]` egress direction),
vSwitch, Storage Box, subnet, traffic, Wake on LAN, ordering.

Gaps in what is implemented: the rate-limit budget fields, the `status` field
on `Error`, the `host_key` element shape, and the four unverified error codes.

## 15. Sources

Hetzner's documentation is at
<https://robot.your-server.de/doc/webservice/en.html>.

Live probes on 2026-07-30 and 2026-07-31 covered `/server`, `/key`, `/ip`,
`/failover`, `/rdns`, `/vswitch`, `/subnet`, `/storagebox`,
`/firewall/template` and `/order/*`, against an account with no dedicated
servers.

Fixtures in `testdata/response/` are trimmed samples of real responses. Two of
them had been trimmed of the fields that were silently being dropped, which is
why a suite at 90% statement coverage did not catch those bugs.
