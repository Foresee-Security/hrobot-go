# Robot Webservice behaviour, inconsistencies and gotchas

Everything here is about how the **Hetzner Robot Webservice actually behaves**,
not about how this client is written. It is the document to read before you
trust an assumption about the API.

Each claim is labelled with where it comes from:

| Label | Meaning |
|---|---|
| **measured** | Observed against the live API on a real account. Trustworthy. |
| **documented** | Stated in Hetzner's published Robot documentation. Trustworthy but not always complete. |
| **unverified** | Inherited from upstream or inferred. Treat as a guess until someone confirms it. |

Measurements were taken on 2026-07-30 and 2026-07-31 against an account that
owns **zero dedicated servers**. That matters: everything server-scoped in this
document rests on documentation and fixtures, not observation. See
[Unproven ground](#unproven-ground).

---

## 1. Robot is not Hetzner Cloud

**documented.** Two unrelated APIs.

| | Robot | Cloud |
|---|---|---|
| Manages | physical servers | virtual machines |
| Billing | monthly, with a setup fee | hourly |
| Endpoint | `robot-ws.your-server.de` | `api.hetzner.cloud` |
| Auth | HTTP basic, Webservice user | bearer token |
| Go client | this package | `hcloud-go` |

Credentials do not cross over. A Cloud token will not authenticate to Robot.

## 2. Authentication, and the lockout that bites automation

**documented.** The Webservice user is created under Settings in the Robot web
interface. It is not the Hetzner account login.

**Three failed logins block the calling IP for ten minutes.** This is the single
most dangerous behaviour for a service. A worker that retries a rejected
credential in a loop locks itself, and every other process on that IP, out of
the API entirely. Treat `UNAUTHORIZED` as terminal and never retry it.

**measured.** A 401 is returned *before* the API looks for the resource. Any
404 you receive therefore proves the credentials were accepted. That is what
makes a 404-tolerant credential check sound, and it is why
`Client.ValidateCredentials` treats a not-found listing as success.

**measured.** The username on our account is `#`-prefixed. Do not assume a
username is a plain identifier when writing config validation.

## 3. Rate limiting arrives as 403

**documented.** `RATE_LIMIT_EXCEEDED` is returned with HTTP status **403**, not
the 429 nearly every other API uses. Code that branches on 429 will not see it.

The same response carries two fields describing the budget:

```json
{"error": {"status": 403, "code": "RATE_LIMIT_EXCEEDED",
           "max_request": 200, "interval": 3600,
           "message": "rate limit exceeded"}}
```

**Gap in this client:** `max_request` and `interval` are not decoded, so the
retry budget is not available to callers. `Error` also drops the `status` field.
There is no built-in retry or backoff.

## 4. Empty collections are answered two different ways

**measured.** This is the API's sharpest inconsistency.

| Endpoint | Empty response |
|---|---|
| `/server` | 200 with `[]` |
| `/rdns` | 200 with `[]` |
| `/ip` | 404 `NOT_FOUND` |
| `/key` | 404 `NOT_FOUND`, message "No keys found" |
| `/failover` | 404 `NOT_FOUND` |
| `/vswitch` | 200 with `[]` |
| `/subnet` | 404 `NOT_FOUND` |
| `/storagebox` | 404 `STORAGEBOX_NOT_FOUND` |

Anything that treats "no rows" as one uniform shape is wrong roughly half the
time.

Note `/key` returns the **generic** `NOT_FOUND`, not a key-specific code. A
caller cannot distinguish "this account has no keys" from "you asked for a path
that does not exist" by error code alone.

**What this client does:** every list method normalises both shapes to an empty
slice and a nil error. The rule is narrow. Only `NOT_FOUND` counts as empty. A
404 carrying a specific code, and a 404 with no error document, both still fail,
so a wrong path stays loud.

## 5. Fields that change JSON type with state

**documented and measured.** Several fields are `Array|Scalar` depending on
whether the resource is active.

| Field | Active | Inactive |
|---|---|---|
| `os` (rescue) | `"linux"` | `["linux","freebsd","vkvm"]` |
| `dist` (linux) | `"CentOS 5.5 minimal"` | `["CentOS 5.5 minimal", ...]` |
| `lang` (linux) | `"en"` | `["en"]` |
| `arch` (both) | `64` | `[64,32]` |
| `cancellation_reason` | `null` once cancelled | array of selectable reasons |

The semantics flip with the shape. A scalar is **the value in force**. An array
is **the menu of what you could choose**. Same field, two different meanings.

**What this client does:** these are `StringList` and `IntList`, which decode
both into a slice. A one-element list is the active value. Check `Active` to
know which meaning applies rather than inferring it from the length, because a
menu can legitimately contain one item.

## 6. Deprecated fields still in the responses

**documented.** `arch` on the boot endpoints is marked `@deprecated` by Hetzner
on both input and output. It is still returned, so this client still decodes it
and marks it `Deprecated:` in godoc. Do not build new logic on it.

## 7. Shapes Hetzner does not document

**unverified.** `host_key` on the boot endpoints is typed only as "Array" in the
documentation, and every published example plus every fixture we have is empty.
The element shape is unknown.

This client surfaces it as `[]json.RawMessage` rather than guessing a struct.
If you ever observe a non-empty `host_key`, that is worth recording and the type
can be tightened.

Note "Array" tells you nothing about elements in this API generally.
`authorized_key` is also documented as "Array" and its elements are
`{"key": {...}}` objects.

## 8. Error codes

**documented — 18 confirmed** against the published error tables:

```
RATE_LIMIT_EXCEEDED   CONFLICT              NOT_FOUND            INVALID_INPUT
SERVER_NOT_FOUND      IP_NOT_FOUND          SUBNET_NOT_FOUND     RDNS_NOT_FOUND
RESET_NOT_AVAILABLE   RESET_MANUAL_ACTIVE   RESET_FAILED
BOOT_NOT_AVAILABLE    BOOT_ACTIVATION_FAILED    BOOT_DEACTIVATION_FAILED
KEY_ALREADY_EXISTS    KEY_CREATE_FAILED     KEY_UPDATE_FAILED    KEY_DELETE_FAILED
```

**unverified — 4 inherited from upstream** and present in no published error
table:

```
UNAUTHORIZED    INTERNAL_ERROR    BOOT_ALREADY_ENABLED    BOOT_BLOCKED
```

They are kept, because deleting one would break a caller that does receive it,
but a match on them is unproven. They live in their own `const` block in
`error.go` saying so. Matching on a code the API never sends fails silently and
forever, which is exactly the bug class this fork was rewritten to remove.

If you observe one of the four in a real response, move it to the verified block
and say where you saw it.

**Unknown codes round-trip.** A code this package has no constant for is
preserved verbatim, so a code Hetzner adds stays matchable without a release
here.

## 9. Endpoints that do not exist

**documented by absence.** There is no `POST /server/{id}/reversal`. Upstream
shipped a `ServerReverse` method calling it for years. Withdrawing a
cancellation is `DELETE /server/{id}/cancellation`, which is what
`ServerCancellationWithdraw` calls.

**measured.** The entire `/order/*` tree returns **401 UNAUTHORIZED** on our
account even though the same credentials work everywhere else. Cause not
determined. Candidates: ordering is a separate entitlement, or an account must
place its first order through the web frontend before API ordering unlocks. Do
not retry these in a loop, see the three-strikes lockout in section 2.

## 10. Booting is two steps, and one of them is destructive

**documented.**

Arming a boot configuration does **not** restart the server.
`BootRescueSet` and `BootLinuxSet` only decide what the machine boots *next*.
It keeps running the current system until something restarts it.

Consequences worth stating plainly:

- To actually enter the rescue system you must follow the arm with `ResetSet`.
- `BootLinuxSet` arms a **destructive reinstall**. If you arm it and the server
  later restarts for any unrelated reason, a power event, a hardware reset, an
  operator, the machine is wiped. An armed installer is a loaded gun with a
  hair trigger, and the trigger is any reboot.
- `password` is only populated while the configuration is active. Capture it
  from the response to the arming call, because a later GET on an inactive
  configuration will not have it.

## 11. Reset types are per-server and one of them involves a human

**documented.** `ResetGet` reports which types a given server supports. Do not
assume.

| Type | Effect | Support |
|---|---|---|
| `power` | ACPI power button. Graceful shutdown, or power on if off. | some servers |
| `power_long` | Held power button. Immediate cut, **server stays off**. | some servers |
| `sw` | CTRL+ALT+DEL. | almost all |
| `hw` | Hardware reset button. Ungraceful. | all |
| `man` | **Emails a data centre technician** to physically recycle power. | all |

Two traps. `power_long` leaves the machine **off** and needs a following `power`
to come back. And `man` is not automation: it creates a human ticket, is slow,
and `RESET_MANUAL_ACTIVE` is returned if one is already running.

## 12. Field-level surprises

**documented and measured.**

- **`Server.Traffic` is a string**, such as `"5 TB"` or `"unlimited"`. It is not
  a number and will not parse as one.
- **`Server.Subnet` can be `null`**, not an empty array. It decodes to a nil
  slice.
- **`Subnet` nested in a server carries only `ip` and `mask`.** The gateway,
  traffic and locking fields belong to the standalone subnet endpoints, which
  this client does not implement.
- **`Subnet.Mask` is a string** (`"64"`) while **`IP.Mask` is a number** (`27`).
  Same concept, two types, two endpoints.
- **`Key.Data` is empty when a key is nested** as an `AuthorizedKey` in a boot
  response. Only the metadata comes through. Fetch the key itself for material.
- **`Key.CreatedAt` is `"2006-01-02 15:04:05"`**, not RFC 3339.
- **Date fields are `"2006-01-02"` strings**, not timestamps.
- **`IP.SeparateMAC` is `null`** when unset, decoding to an empty string.
- **`Cancellation` uses `reserved`**, not `reservation`. Upstream had the wrong
  tag, so the field never populated.
- **`Failover.ServerIP` and `Failover.ActiveServerIP` differ** once an address
  has been switched. The first is the owning server, the second is where traffic
  actually goes. Conflating them defeats the point of a failover address.

## 13. What this client changes about the raw API

Behaviours below are this package's, not Hetzner's. They exist because the raw
behaviour is a footgun.

| Raw API | This client |
|---|---|
| Empty collection is 200 `[]` or 404 depending on endpoint | always an empty slice |
| A redirect can replay basic auth to another port or over http | cross-origin redirects refused |
| Caller input concatenated into a URL path | escaped, and enforced by the type system |
| No request bound | 30s default timeout, context honoured |
| Unbounded response read | 8 MiB cap |
| Error is untyped JSON | `Error` with a matchable code, or `StatusError` |
| A bad server number costs a round trip | rejected locally before any request |

## 14. Unproven ground

Be honest about what has and has not been exercised.

**Never run against a real dedicated server.** The account used for measurement
owns zero of them. Every server-scoped path in this client, `ServerGet`,
`ServerSetName`, `ServerCancellationWithdraw`, all six boot methods, both reset
methods, has only ever been exercised against fixtures and the published
documentation. The 404 paths were confirmed live, the success paths were not.

**Not implemented at all:** firewall (including the `rules[output]` egress
direction), vSwitch, Storage Box, subnet, traffic, Wake on LAN, ordering.

**Known gaps in what is implemented:** the rate-limit budget fields, the
`status` field on `Error`, the `host_key` element shape, and the four unverified
error codes.

## 15. Evidence

- Hetzner's documentation: <https://robot.your-server.de/doc/webservice/en.html>
- Live probes, 2026-07-30 and 2026-07-31, against an account with zero
  dedicated servers, covering `/server`, `/key`, `/ip`, `/failover`, `/rdns`,
  `/vswitch`, `/subnet`, `/storagebox`, `/firewall/template` and `/order/*`.
- Fixtures in `testdata/response/` are trimmed samples of real responses. Two
  of them were previously trimmed of the very fields that were silently being
  dropped, which is why the bugs survived a suite at 90% coverage.
