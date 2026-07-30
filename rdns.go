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
func (c *Client) RDNSGetList(ctx context.Context) ([]RDNS, error) {
	const op = "rdns list"

	list, err := fetch[[]rdnsResponse](ctx, c, op, http.MethodGet, "/rdns", nil)
	if err != nil {
		return nil, err
	}

	entries := make([]RDNS, 0, len(list))
	for i := range list {
		entries = append(entries, list[i].RDNS)
	}
	return entries, nil
}

// RDNSGet returns the reverse DNS entry for one address.
//
// It returns [ErrEmptyIP] without contacting the API if ip is empty, because an
// empty address would otherwise address the collection endpoint and decode a
// list where the caller expects one entry.
func (c *Client) RDNSGet(ctx context.Context, ip string) (*RDNS, error) {
	const op = "rdns get"

	segment, err := ipSegment(ip)
	if err != nil {
		return nil, err
	}

	resp, err := fetch[rdnsResponse](ctx, c, op, http.MethodGet, "/rdns/"+segment, nil)
	if err != nil {
		return nil, err
	}
	return &resp.RDNS, nil
}
