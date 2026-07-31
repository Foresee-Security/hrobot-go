package hrobot_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	hrobot "github.com/Foresee-Security/hrobot-go"
)

// These examples are compiled by the test suite, so the documentation cannot
// drift away from the API without the build noticing. They are not executed,
// because every one of them would need a live Robot account.

func Example() {
	c := hrobot.NewBasicAuthClient("user", "pass")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	servers, err := c.ServerGetList(ctx)
	if err != nil {
		// Returning rather than calling log.Fatal, which would exit before
		// the deferred cancel could run.
		fmt.Println("list servers:", err)
		return
	}

	// Indexed rather than ranged by value, because Server is a little over
	// 200 bytes and copying it per iteration buys nothing.
	for i := range servers {
		s := &servers[i]
		fmt.Printf("%d  %-20s %-10s %s\n", s.ServerNumber, s.Name, s.DC, s.Status)
	}
}

// An account that owns nothing yields an empty slice and a nil error, on every
// collection. The raw API answers that situation two different ways depending
// on the endpoint, and this client normalises both.
func ExampleClient_KeyGetList() {
	c := hrobot.NewBasicAuthClient("user", "pass")

	keys, err := c.KeyGetList(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("account holds %d keys\n", len(keys))
}

// Arming the rescue system does not restart the server. It decides what the
// machine boots next, and restarting it is a separate deliberate step.
func ExampleClient_BootRescueSet() {
	c := hrobot.NewBasicAuthClient("user", "pass")
	ctx := context.Background()

	const serverID = 321

	rescue, err := c.BootRescueSet(ctx, serverID, &hrobot.RescueSetInput{
		OS: "linux",
	})
	if err != nil {
		log.Fatalf("arm rescue system: %v", err)
	}

	// Capture the password now. A later read of an inactive configuration
	// will not carry it.
	log.Printf("rescue password issued for %s", rescue.ServerIP)

	// Nothing has rebooted yet. This is the step that enters the rescue system.
	if _, err := c.ResetSet(ctx, serverID, &hrobot.ResetSetInput{
		Type: hrobot.ResetTypeHardware,
	}); err != nil {
		log.Fatalf("restart into rescue: %v", err)
	}
}

// Which reset types a server supports varies by machine, so ask before
// choosing one.
func ExampleClient_ResetGet() {
	c := hrobot.NewBasicAuthClient("user", "pass")

	reset, err := c.ResetGet(context.Background(), 321)
	if err != nil {
		log.Fatal(err)
	}

	switch {
	case slices.Contains(reset.Type, hrobot.ResetTypePower):
		// Graceful. The operating system is asked to shut down.
	case slices.Contains(reset.Type, hrobot.ResetTypeHardware):
		// Ungraceful, but supported everywhere.
	default:
		// ResetTypeManual emails a technician. It is not automation.
	}
}

// Boot fields hold the value in force while the configuration is active, and
// the menu of available choices while it is not.
func ExampleClient_BootRescueGet() {
	c := hrobot.NewBasicAuthClient("user", "pass")

	rescue, err := c.BootRescueGet(context.Background(), 321)
	if err != nil {
		log.Fatal(err)
	}

	// Check Active rather than inferring from the length, because a menu can
	// legitimately hold a single item.
	if rescue.Active {
		fmt.Println("running:", rescue.OS[0])
	} else {
		fmt.Println("available:", strings.Join(rescue.OS, ", "))
	}
}

// IsError unwraps, so adding context with %w upstream does not stop a code
// matching.
func ExampleIsError() {
	c := hrobot.NewBasicAuthClient("user", "pass")

	_, err := c.ServerGet(context.Background(), 321)
	if err != nil {
		err = fmt.Errorf("load server: %w", err)
	}

	switch {
	case hrobot.IsError(err, hrobot.ErrorCodeServerNotFound):
		fmt.Println("that server number is not on this account")
	case hrobot.IsError(err, hrobot.ErrorCodeRateLimitExceeded):
		// Robot reports a rate limit with HTTP 403, not 429.
		fmt.Println("backing off")
	case err != nil:
		log.Fatal(err)
	}
}

// A failure whose body was not a Robot error document is a StatusError, which
// is what an intermediary such as a proxy produces. The status is preserved so
// a retryable 5xx stays distinguishable from a terminal 4xx.
func ExampleStatusError() {
	c := hrobot.NewBasicAuthClient("user", "pass")

	_, err := c.ServerGetList(context.Background())

	var statusErr hrobot.StatusError
	if errors.As(err, &statusErr) && statusErr.StatusCode >= 500 {
		fmt.Println("upstream failure, worth retrying")
	}
}

// Arguments are checked before a request is built, so a misuse fails locally
// rather than after a network round trip.
func ExampleClient_ServerGet_validation() {
	c := hrobot.NewBasicAuthClient("user", "pass")

	_, err := c.ServerGet(context.Background(), 0)
	fmt.Println(errors.Is(err, hrobot.ErrInvalidServerID))

	// Output: true
}

// Options may be given in any order. A client you supply keeps its own policy,
// and WithTimeout retimes a copy rather than the client you passed in.
func ExampleWithHTTPClient() {
	// Whatever transport your service already uses for outbound calls, with
	// its proxy settings, tracing or connection pooling.
	instrumented := &http.Client{Transport: http.DefaultTransport}

	c := hrobot.NewBasicAuthClient("user", "pass",
		hrobot.WithHTTPClient(instrumented),
		hrobot.WithTimeout(10*time.Second),
	)

	_ = c
}
