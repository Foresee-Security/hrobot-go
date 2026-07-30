# hrobot-go: A Go library for the Hetzner Robot Webservice

Package hrobot-go is a library for the Hetzner Robot Webservice.

The public API documentation is available at
[robot.your-server.de](https://robot.your-server.de/doc/webservice/en.html).

## About this fork

This is **Foresee Security's actively developed fork**, and it is expected to
diverge from upstream rather than track it closely. We use it in production to
drive bare-metal analysis hosts, so we are treating it as code we own: we will
extend the API surface where the Robot Webservice offers something this library
does not yet expose, and we will change existing behaviour where it does not
meet our reliability or security bar.

The lineage is nl2go, then [syself](https://github.com/syself/hrobot-go), then
here. Upstream is tracked as a git remote and we will merge from it where that
is useful, but we are not constrained by it.

Note the module path is `github.com/Foresee-Security/hrobot-go`. Import that,
not the upstream path.

### Quality gate

This fork is checked with the same setup as the code that consumes it: Go
1.26.5, `golangci-lint` 2.12.2 across 38 linters against the `.golangci.yml`
here, `go test -race`, `govulncheck`, and `nilaway`. That first run reported 28
lint issues. Three remain, all of them the same one, described below.

### Closed

- **Context and deadlines.** Every method that performs I/O takes a
  `context.Context` first and builds requests with
  `http.NewRequestWithContext`. Previously nothing could be cancelled and no
  call carried a deadline.
- **Request timeout.** `NewBasicAuthClient` built a bare `&http.Client{}`,
  which in Go means no timeout at all. It now defaults to 30 seconds. A
  caller-supplied client keeps its own.
- **Credential redaction.** `String` and `LogValue` mask the password, so a
  stray `%v` cannot leak it.
- **Bounded reads.** Response bodies cap at 8 MiB and report oversize rather
  than truncating into an undiagnosable parse failure.
- **Wrapped-error matching.** `models.IsError` uses `errors.As` instead of a
  direct type assertion, so adding context with `%w` no longer breaks code
  matching.

### Still open

1. **Missing endpoints.** Firewall (including the `rules[output]` egress
   direction), vSwitch, Storage Box, traffic, subnet, and the whole ordering
   tree are absent. Firewall is the one we need first.
2. **Test framework.** All eight test files dot-import `gopkg.in/check.v1`,
   which is that framework's idiom and the source of the three remaining lint
   findings. Moving to the standard library would also drop the last three
   dependencies and leave this module with none, but it means rewriting 225
   assertions. Deliberately deferred rather than overlooked.
3. **No CI.** The gate above runs by hand.

The `go` directive has been moved from 1.17 to 1.26 to match the toolchain we
build against.

Contributions and issues are welcome. Where a change is generally useful and
not specific to how we operate, we would rather send it upstream than keep it
here.

## Example

```go
package main

import (
    "fmt"
    "log"

    client "github.com/Foresee-Security/hrobot-go"
)

func main() {
    robotClient := client.NewBasicAuthClient("user", "pass")

    servers, err := robotClient.ServerGetList()
    if err != nil {
        log.Fatalf("error while retrieving server list: %s\n", err)
    }

    fmt.Println(servers)
}
```

To add instrumentation, for example to debug Hetzner API rate limits, use
`NewBasicAuthClientWithCustomHttpClient()` to supply your own `http.Client`.
Until the timeout defaults above are fixed, supplying your own client with an
explicit timeout is also the way to avoid point 2.

## Releasing

Update the version number in `client.go`.

```sh
make test

export RELEASE_TAG=vX.Y.Z

git tag -a ${RELEASE_TAG} -m ${RELEASE_TAG}

git push origin $RELEASE_TAG
```
