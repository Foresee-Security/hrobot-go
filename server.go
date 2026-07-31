package hrobot

import (
	"context"
	"net/http"
	"net/url"
)

// Server is a dedicated server on the account.
type Server struct {
	// ServerIP is the main IPv4 address.
	ServerIP string `json:"server_ip"`
	// ServerIPv6Net is the main IPv6 network, such as "2a01:4f8:111:4221::".
	ServerIPv6Net string `json:"server_ipv6_net"`
	// ServerNumber is the Robot server ID, the value every server-scoped
	// endpoint is addressed by.
	ServerNumber int `json:"server_number"`
	// Name is the operator-assigned label, set with [Client.ServerSetName].
	Name string `json:"server_name"`
	// Product is the marketing name of the hardware, such as "EQ 8".
	Product string `json:"product"`
	// DC identifies the data centre and rack, such as "NBG1-DC1".
	DC string `json:"dc"`
	// Traffic is the monthly allowance as a human-readable string, such as
	// "5 TB" or "unlimited". It is not a number.
	Traffic string `json:"traffic"`
	// Status is the provisioning state, "ready" or "in process".
	Status string `json:"status"`
	// Cancelled reports whether the server is under notice.
	Cancelled bool `json:"cancelled"`
	// PaidUntil is the end of the paid period, formatted "2006-01-02".
	PaidUntil string `json:"paid_until"`
	// IP lists the single IPv4 addresses assigned to the server.
	IP []string `json:"ip"`
	// Subnet lists the assigned subnets. The API sends null rather than an
	// empty array when there are none, which decodes to a nil slice.
	Subnet []Subnet `json:"subnet"`
	// Reset reports whether a reset can be triggered through the API.
	Reset bool `json:"reset"`
	// Rescue reports whether the rescue system is available.
	Rescue bool `json:"rescue"`
	// VNC reports whether VNC installation is available.
	VNC bool `json:"vnc"`
	// Windows reports whether a Windows installation is available.
	Windows bool `json:"windows"`
	// Plesk reports whether a Plesk installation is available.
	Plesk bool `json:"plesk"`
	// CPanel reports whether a cPanel installation is available.
	CPanel bool `json:"cpanel"`
	// WOL reports whether Wake on LAN is available.
	WOL bool `json:"wol"`
	// HotSwap reports whether the server has a hot-swap drive tray.
	HotSwap bool `json:"hot_swap"`
	// LinkedStoragebox is the ID of an attached Storage Box, or zero.
	LinkedStoragebox int `json:"linked_storagebox"`
}

// Subnet is a subnet as it appears nested in a [Server].
//
// It carries only the two fields the server endpoints return. The standalone
// subnet endpoints, which report gateway, traffic and locking state as well,
// are not implemented by this client.
type Subnet struct {
	// IP is the network address.
	IP string `json:"ip"`
	// Mask is the prefix length, sent by the API as a string such as "64".
	Mask string `json:"mask"`
}

// ServerSetNameInput names the field [Client.ServerSetName] updates.
type ServerSetNameInput struct {
	// Name is the new label. An empty value clears the current one.
	Name string
}

// Cancellation is the cancellation state of a server.
type Cancellation struct {
	// ServerIP is the server's main IPv4 address.
	ServerIP string `json:"server_ip"`
	// ServerIPv6Net is the server's main IPv6 network.
	ServerIPv6Net string `json:"server_ipv6_net"`
	// ServerNumber is the Robot server ID.
	ServerNumber int `json:"server_number"`
	// Name is the server's label.
	Name string `json:"server_name"`
	// EarliestCancellationDate is the soonest date the contract can end,
	// formatted "2006-01-02".
	EarliestCancellationDate string `json:"earliest_cancellation_date"`
	// Cancelled reports whether a cancellation is currently registered.
	Cancelled bool `json:"cancelled"`
	// ReservationPossible reports whether the server's location may be
	// reserved when cancelling.
	ReservationPossible bool `json:"reservation_possible"`
	// Reserved reports whether the location has been reserved. The API sends
	// this as "reserved", not "reservation".
	Reserved bool `json:"reserved"`
	// CancellationDate is the registered end date, formatted "2006-01-02".
	CancellationDate string `json:"cancellation_date"`
	// CancellationReason holds the selectable reasons while the server is not
	// cancelled, and the chosen reason once it is. It is empty when the API
	// sends null. See [StringList].
	CancellationReason StringList `json:"cancellation_reason"`
}

// serverResponse is the envelope a single server is wrapped in.
type serverResponse struct {
	Server Server `json:"server"`
}

// cancellationResponse is the envelope a cancellation is wrapped in.
type cancellationResponse struct {
	Cancellation Cancellation `json:"cancellation"`
}

// ServerGetList returns every dedicated server on the account.
//
// An account with no dedicated servers answers 200 with an empty array, so this
// returns an empty slice and a nil error. That is measured against the live API
// rather than inferred. Robot is not consistent about this across collections,
// and [Client.IPGetList], [Client.KeyGetList] and [Client.FailoverGetList]
// answer the same situation with a 404.
func (c *Client) ServerGetList(ctx context.Context) ([]Server, error) {
	const op = "server list"

	list, err := fetch[[]serverResponse](ctx, c, op, http.MethodGet, "/server", nil)
	if err != nil {
		return nil, err
	}

	servers := make([]Server, 0, len(list))
	for i := range list {
		servers = append(servers, list[i].Server)
	}
	return servers, nil
}

// ServerGet returns the server with the given Robot server number.
//
// It returns [ErrInvalidServerID] without contacting the API if id is not
// positive.
func (c *Client) ServerGet(ctx context.Context, id int) (*Server, error) {
	const op = "server get"

	segment, err := serverSegment(id)
	if err != nil {
		return nil, err
	}

	resp, err := fetch[serverResponse](ctx, c, op, http.MethodGet, "/server/"+segment, nil)
	if err != nil {
		return nil, err
	}
	return &resp.Server, nil
}

// ServerSetName sets the server's label and returns the updated server.
//
// It returns [ErrNilInput] if input is nil, and [ErrInvalidServerID] if id is
// not positive, in both cases without contacting the API.
func (c *Client) ServerSetName(ctx context.Context, id int, input *ServerSetNameInput) (*Server, error) {
	const op = "server set name"

	if input == nil {
		return nil, ErrNilInput
	}
	segment, err := serverSegment(id)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("server_name", input.Name)

	resp, err := fetch[serverResponse](ctx, c, op, http.MethodPost, "/server/"+segment, form)
	if err != nil {
		return nil, err
	}
	return &resp.Server, nil
}

// ServerCancellationWithdraw revokes a registered cancellation and returns the
// resulting cancellation state.
//
// It returns [ErrInvalidServerID] without contacting the API if id is not
// positive.
func (c *Client) ServerCancellationWithdraw(ctx context.Context, id int) (*Cancellation, error) {
	const op = "server cancellation withdraw"

	segment, err := serverSegment(id)
	if err != nil {
		return nil, err
	}

	resp, err := fetch[cancellationResponse](ctx, c, op, http.MethodDelete, "/server/"+segment+"/cancellation", nil)
	if err != nil {
		return nil, err
	}
	return &resp.Cancellation, nil
}
