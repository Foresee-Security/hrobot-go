package hrobot

import (
	"context"
	"net/http"
)

// Failover is a failover address, which can be routed to any server on the
// account and rerouted to another to survive a host failure.
type Failover struct {
	// IP is the failover address itself, IPv4 or IPv6.
	IP string `json:"ip"`
	// Netmask is the address's netmask, in dotted-quad form for IPv4 and
	// colon-hex form for IPv6.
	Netmask string `json:"netmask"`
	// ServerIP is the main address of the server the failover address belongs
	// to. This is the owning server, which is not necessarily the one traffic
	// currently reaches.
	ServerIP string `json:"server_ip"`
	// ServerIPv6Net is the owning server's main IPv6 network.
	ServerIPv6Net string `json:"server_ipv6_net"`
	// ServerNumber is the owning server's Robot server ID.
	ServerNumber int `json:"server_number"`
	// ActiveServerIP is the main address of the server traffic is currently
	// routed to. It differs from ServerIP once the address has been switched.
	ActiveServerIP string `json:"active_server_ip"`
}

// failoverResponse is the envelope a single failover address is wrapped in.
type failoverResponse struct {
	Failover Failover `json:"failover"`
}

// FailoverGetList returns every failover address on the account.
//
// An account with none gets an empty slice and a nil error. See [fetchList] for
// why that takes normalising.
func (c *Client) FailoverGetList(ctx context.Context) ([]Failover, error) {
	return fetchList(ctx, c, "failover list", "/failover", func(e failoverResponse) Failover { return e.Failover })
}

// FailoverGet returns one failover address.
//
// It returns [ErrEmptyIP] without contacting the API if ip is empty, because an
// empty address would otherwise address the collection endpoint and decode a
// list where the caller expects one address.
func (c *Client) FailoverGet(ctx context.Context, ip string) (*Failover, error) {
	const op = "failover get"

	path, err := addressEndpoint("/failover", ip)
	if err != nil {
		return nil, err
	}

	resp, err := fetch[failoverResponse](ctx, c, op, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return &resp.Failover, nil
}
