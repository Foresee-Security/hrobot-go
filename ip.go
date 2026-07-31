package hrobot

import (
	"context"
	"net/http"
)

// IP is a single IPv4 address assigned to a server.
type IP struct {
	// IP is the address itself.
	IP string `json:"ip"`
	// Gateway is the address's gateway.
	Gateway string `json:"gateway"`
	// Mask is the prefix length as a number, such as 27.
	Mask int `json:"mask"`
	// Broadcast is the broadcast address.
	Broadcast string `json:"broadcast"`
	// ServerIP is the main address of the server holding this one.
	ServerIP string `json:"server_ip"`
	// ServerNumber is the Robot server ID of the holding server.
	ServerNumber int `json:"server_number"`
	// Locked reports whether the address is administratively locked.
	Locked bool `json:"locked"`
	// SeparateMAC is the address's own MAC where one is assigned. The API
	// sends null when there is none, which decodes to an empty string.
	SeparateMAC string `json:"separate_mac"`
	// TrafficWarnings reports whether traffic warnings are enabled.
	TrafficWarnings bool `json:"traffic_warnings"`
	// TrafficHourly is the hourly traffic warning threshold, in MB.
	TrafficHourly int `json:"traffic_hourly"`
	// TrafficDaily is the daily traffic warning threshold, in MB.
	TrafficDaily int `json:"traffic_daily"`
	// TrafficMonthly is the monthly traffic warning threshold, in GB.
	TrafficMonthly int `json:"traffic_monthly"`
}

// ipResponse is the envelope a single address is wrapped in.
type ipResponse struct {
	IP IP `json:"ip"`
}

// IPGetList returns every single IPv4 address on the account.
//
// An account with no single addresses answers 404, so this returns an [Error]
// with [ErrorCodeNotFound] rather than an empty slice. Measured against the
// live API. See [Client.ServerGetList] for why that differs per collection.
func (c *Client) IPGetList(ctx context.Context) ([]IP, error) {
	const op = "ip list"

	list, err := fetch[[]ipResponse](ctx, c, op, http.MethodGet, "/ip", nil)
	if err != nil {
		return nil, err
	}

	ips := make([]IP, 0, len(list))
	for i := range list {
		ips = append(ips, list[i].IP)
	}
	return ips, nil
}
