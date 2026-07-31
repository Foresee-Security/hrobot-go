package hrobot

import (
	"context"
	"net/http"
)

// RDNS is the reverse DNS entry for one address.
type RDNS struct {
	// IP is the address the entry is for.
	IP string `json:"ip"`
	// PTR is the hostname the address resolves back to.
	PTR string `json:"ptr"`
}

// rdnsResponse is the envelope a single entry is wrapped in.
type rdnsResponse struct {
	RDNS RDNS `json:"rdns"`
}

// RDNSGetList returns every reverse DNS entry on the account.
//
// An account with none gets an empty slice and a nil error. See [fetchList] for
// why that takes normalising.
func (c *Client) RDNSGetList(ctx context.Context) ([]RDNS, error) {
	return fetchList(ctx, c, "rdns list", "/rdns", func(e rdnsResponse) RDNS { return e.RDNS })
}

// RDNSGet returns the reverse DNS entry for one address.
//
// It returns [ErrEmptyIP] without contacting the API if ip is empty, because an
// empty address would otherwise address the collection endpoint and decode a
// list where the caller expects one entry.
func (c *Client) RDNSGet(ctx context.Context, ip string) (*RDNS, error) {
	const op = "rdns get"

	path, err := addressEndpoint("/rdns", ip)
	if err != nil {
		return nil, err
	}

	resp, err := fetch[rdnsResponse](ctx, c, op, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return &resp.RDNS, nil
}
