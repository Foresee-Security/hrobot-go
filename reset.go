package hrobot

import (
	"context"
	"net/http"
	"net/url"
)

// ResetType is a way of restarting a server. Which types a given server
// supports is reported by [Client.ResetGet] in [Reset.Type].
type ResetType string

const (
	// ResetTypePower presses the power button. A running server receives an
	// ACPI signal and shuts down the way a desktop does when its power button
	// is pressed, and a powered-down server turns back on. Supported by some
	// servers.
	ResetTypePower ResetType = "power"

	// ResetTypePowerLong holds the power button down, cutting power
	// immediately without giving the operating system a chance to shut down.
	// The server stays off afterwards, so it takes a following
	// [ResetTypePower] to turn it back on. Supported by some servers.
	ResetTypePowerLong ResetType = "power_long"

	// ResetTypeSoftware sends CTRL+ALT+DEL. Supported by almost all servers.
	ResetTypeSoftware ResetType = "sw"

	// ResetTypeHardware triggers a hardware reset, the equivalent of a desktop
	// reset button. Supported by all servers.
	ResetTypeHardware ResetType = "hw"

	// ResetTypeManual emails Hetzner's data centre staff to physically
	// disconnect and reconnect the server's power. Supported by all servers,
	// but it involves a human and is not suited to automation.
	ResetTypeManual ResetType = "man"
)

// Reset reports which reset options a server supports.
type Reset struct {
	// ServerIP is the server's main IPv4 address.
	ServerIP string `json:"server_ip"`
	// ServerIPv6Net is the server's main IPv6 network.
	ServerIPv6Net string `json:"server_ipv6_net"`
	// ServerNumber is the Robot server ID.
	ServerNumber int `json:"server_number"`
	// Type lists every reset option this server supports.
	Type []ResetType `json:"type"`
	// OperatingStatus is the server's power state, such as "running" or
	// "not supported" where the hardware cannot report it.
	OperatingStatus string `json:"operating_status"`
}

// ResetPost is the outcome of a triggered reset.
type ResetPost struct {
	// ServerIP is the server's main IPv4 address.
	ServerIP string `json:"server_ip"`
	// ServerIPv6Net is the server's main IPv6 network.
	ServerIPv6Net string `json:"server_ipv6_net"`
	// ServerNumber is the Robot server ID.
	ServerNumber int `json:"server_number"`
	// Type is the reset that was performed.
	Type ResetType `json:"type"`
}

// ResetSetInput selects the reset [Client.ResetSet] performs.
type ResetSetInput struct {
	// Type is the reset to perform. It must be one the server supports, as
	// reported by [Client.ResetGet].
	Type ResetType
}

// resetResponse is the envelope reset options are wrapped in.
type resetResponse struct {
	Reset Reset `json:"reset"`
}

// resetPostResponse is the envelope a performed reset is wrapped in.
type resetPostResponse struct {
	Reset ResetPost `json:"reset"`
}

// ResetGet returns the reset options a server supports.
//
// It returns [ErrInvalidServerID] without contacting the API if id is not
// positive.
func (c *Client) ResetGet(ctx context.Context, id int) (*Reset, error) {
	const op = "reset get"

	path, err := scopedEndpoint("/reset", id, "")
	if err != nil {
		return nil, err
	}

	resp, err := fetch[resetResponse](ctx, c, op, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return &resp.Reset, nil
}

// ResetSet restarts a server.
//
// This is disruptive. Every type other than [ResetTypePower] interrupts the
// running system, and [ResetTypeHardware] and [ResetTypePowerLong] do so
// without letting it shut down cleanly.
//
// It returns [ErrNilInput] if input is nil, and [ErrInvalidServerID] if id is
// not positive, in both cases without contacting the API. Asking for a type
// the server does not support surfaces as an [Error] with
// [ErrorCodeResetNotAvailable].
func (c *Client) ResetSet(ctx context.Context, id int, input *ResetSetInput) (*ResetPost, error) {
	const op = "reset set"

	if input == nil {
		return nil, ErrNilInput
	}
	path, err := scopedEndpoint("/reset", id, "")
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("type", string(input.Type))

	resp, err := fetch[resetPostResponse](ctx, c, op, http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}
	return &resp.Reset, nil
}
