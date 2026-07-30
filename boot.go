package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// AuthorizedKey wraps an SSH key in the envelope the boot endpoints return it in.
type AuthorizedKey struct {
	// Key is the authorized key. Only its metadata is populated here, not the
	// key material.
	Key Key `json:"key"`
}

// Rescue is the rescue-system boot configuration of a server.
//
// The OS and Arch fields change shape with Active. While the rescue system is
// active they hold the single running choice, and while it is inactive they
// hold every available choice. See [StringList].
type Rescue struct {
	// ServerIP is the server's main IPv4 address.
	ServerIP string `json:"server_ip"`
	// ServerIPv6Net is the server's main IPv6 network.
	ServerIPv6Net string `json:"server_ipv6_net"`
	// ServerNumber is the Robot server ID.
	ServerNumber int `json:"server_number"`
	// OS holds the active rescue system, or every available one.
	OS StringList `json:"os"`
	// Arch holds the active architecture, or every available one.
	//
	// Deprecated: Hetzner has marked the boot architecture field deprecated.
	// It is retained because the API still returns it.
	Arch IntList `json:"arch"`
	// Active reports whether the rescue system is armed for the next boot.
	Active bool `json:"active"`
	// Password is the generated root password, set only while Active.
	Password string `json:"password"`
	// AuthorizedKeys lists the SSH keys authorized for the rescue system.
	AuthorizedKeys []AuthorizedKey `json:"authorized_key"`
	// HostKeys holds the rescue system's host keys.
	//
	// The Robot documentation types this only as an array and every published
	// example is empty, so the elements are surfaced as raw JSON rather than
	// decoded into a guessed shape.
	HostKeys []json.RawMessage `json:"host_key"`
}

// RescueSetInput selects the rescue system [Client.BootRescueSet] activates.
type RescueSetInput struct {
	// OS is the rescue system to boot, such as "linux" or "vkvm". Required.
	OS string
	// Arch is the architecture to boot. Omitted from the request when zero.
	//
	// Deprecated: Hetzner has marked the boot architecture field deprecated.
	Arch int
	// AuthorizedKey is the fingerprint of an SSH key to authorize. Omitted
	// from the request when empty, which makes the API generate a password
	// instead.
	AuthorizedKey string
}

// Linux is the Linux-installation boot configuration of a server.
//
// The Dist, Arch and Lang fields change shape with Active, exactly as
// [Rescue] does.
type Linux struct {
	// ServerIP is the server's main IPv4 address.
	ServerIP string `json:"server_ip"`
	// ServerIPv6Net is the server's main IPv6 network.
	ServerIPv6Net string `json:"server_ipv6_net"`
	// ServerNumber is the Robot server ID.
	ServerNumber int `json:"server_number"`
	// Dist holds the active distribution, or every available one.
	Dist StringList `json:"dist"`
	// Arch holds the active architecture, or every available one.
	//
	// Deprecated: Hetzner has marked the boot architecture field deprecated.
	// It is retained because the API still returns it.
	Arch IntList `json:"arch"`
	// Lang holds the active installer language, or every available one.
	Lang StringList `json:"lang"`
	// Active reports whether the installer is armed for the next boot.
	Active bool `json:"active"`
	// Password is the generated root password, set only while Active.
	Password string `json:"password"`
	// AuthorizedKeys lists the SSH keys authorized for the installation.
	AuthorizedKeys []AuthorizedKey `json:"authorized_key"`
	// HostKeys holds the installed system's host keys. See [Rescue.HostKeys]
	// for why the elements stay raw.
	HostKeys []json.RawMessage `json:"host_key"`
}

// LinuxSetInput selects the installation [Client.BootLinuxSet] arms.
type LinuxSetInput struct {
	// Dist is the distribution to install. Required.
	Dist string
	// Arch is the architecture to install. Omitted from the request when zero.
	//
	// Deprecated: Hetzner has marked the boot architecture field deprecated.
	Arch int
	// Lang is the installer language. Omitted from the request when empty.
	Lang string
	// AuthorizedKey is the fingerprint of an SSH key to authorize. Omitted
	// from the request when empty.
	AuthorizedKey string
}

// rescueResponse is the envelope a rescue configuration is wrapped in.
type rescueResponse struct {
	Rescue Rescue `json:"rescue"`
}

// linuxResponse is the envelope a Linux configuration is wrapped in.
type linuxResponse struct {
	Linux Linux `json:"linux"`
}

// BootRescueGet returns the rescue-system configuration of a server.
//
// It returns [ErrInvalidServerID] without contacting the API if id is not
// positive.
func (c *Client) BootRescueGet(ctx context.Context, id int) (*Rescue, error) {
	return c.bootRescue(ctx, "boot rescue get", http.MethodGet, id, nil)
}

// BootRescueSet arms the rescue system for the server's next boot and returns
// the resulting configuration, including the generated root password.
//
// Arming the rescue system does not reboot the server. Follow it with
// [Client.ResetSet] to make the server boot into it.
//
// It returns [ErrNilInput] if input is nil, and [ErrInvalidServerID] if id is
// not positive, in both cases without contacting the API.
func (c *Client) BootRescueSet(ctx context.Context, id int, input *RescueSetInput) (*Rescue, error) {
	if input == nil {
		return nil, ErrNilInput
	}

	form := url.Values{}
	form.Set("os", input.OS)
	if input.Arch > 0 {
		form.Set("arch", strconv.Itoa(input.Arch))
	}
	if input.AuthorizedKey != "" {
		form.Set("authorized_key", input.AuthorizedKey)
	}

	return c.bootRescue(ctx, "boot rescue set", http.MethodPost, id, form)
}

// BootRescueDelete disarms the rescue system and returns the resulting
// configuration.
//
// It returns [ErrInvalidServerID] without contacting the API if id is not
// positive.
func (c *Client) BootRescueDelete(ctx context.Context, id int) (*Rescue, error) {
	return c.bootRescue(ctx, "boot rescue delete", http.MethodDelete, id, nil)
}

// BootLinuxGet returns the Linux-installation configuration of a server.
//
// It returns [ErrInvalidServerID] without contacting the API if id is not
// positive.
func (c *Client) BootLinuxGet(ctx context.Context, id int) (*Linux, error) {
	return c.bootLinux(ctx, "boot linux get", http.MethodGet, id, nil)
}

// BootLinuxSet arms an automatic Linux installation for the server's next boot
// and returns the resulting configuration, including the generated password.
//
// Arming the installer does not reboot the server, and the installation is
// destructive once it runs. Follow it with [Client.ResetSet] deliberately.
//
// It returns [ErrNilInput] if input is nil, and [ErrInvalidServerID] if id is
// not positive, in both cases without contacting the API.
func (c *Client) BootLinuxSet(ctx context.Context, id int, input *LinuxSetInput) (*Linux, error) {
	if input == nil {
		return nil, ErrNilInput
	}

	form := url.Values{}
	form.Set("dist", input.Dist)
	if input.Arch > 0 {
		form.Set("arch", strconv.Itoa(input.Arch))
	}
	if input.Lang != "" {
		form.Set("lang", input.Lang)
	}
	if input.AuthorizedKey != "" {
		form.Set("authorized_key", input.AuthorizedKey)
	}

	return c.bootLinux(ctx, "boot linux set", http.MethodPost, id, form)
}

// BootLinuxDelete disarms the Linux installation and returns the resulting
// configuration.
//
// It returns [ErrInvalidServerID] without contacting the API if id is not
// positive.
func (c *Client) BootLinuxDelete(ctx context.Context, id int) (*Linux, error) {
	return c.bootLinux(ctx, "boot linux delete", http.MethodDelete, id, nil)
}

// bootRescue runs one call against a server's rescue endpoint. The three rescue
// methods differ only in HTTP method and request body.
func (c *Client) bootRescue(ctx context.Context, op, method string, id int, form url.Values) (*Rescue, error) {
	segment, err := serverSegment(id)
	if err != nil {
		return nil, err
	}

	resp, err := fetch[rescueResponse](ctx, c, op, method, "/boot/"+segment+"/rescue", form)
	if err != nil {
		return nil, err
	}
	return &resp.Rescue, nil
}

// bootLinux runs one call against a server's linux endpoint, for the same
// reason [Client.bootRescue] exists.
func (c *Client) bootLinux(ctx context.Context, op, method string, id int, form url.Values) (*Linux, error) {
	segment, err := serverSegment(id)
	if err != nil {
		return nil, err
	}

	resp, err := fetch[linuxResponse](ctx, c, op, method, "/boot/"+segment+"/linux", form)
	if err != nil {
		return nil, err
	}
	return &resp.Linux, nil
}
