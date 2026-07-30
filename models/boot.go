package models

type RescueResponse struct {
	Rescue Rescue `json:"rescue"`
}
type AuthorizedKey struct {
	Key Key `json:"key"`
}

type Rescue struct {
	ServerIP      string          `json:"server_ip"`
	ServerIPv6Net string          `json:"server_ipv6_net"`
	ServerNumber  int             `json:"server_number"`
	Os            any             `json:"os"`   // unfortunately the API returns an array vs. a single value based on state (active/inactive)
	Arch          any             `json:"arch"` // unfortunately the API returns an array vs. a single value based on state (active/inactive)
	Active        bool            `json:"active"`
	Password      string          `json:"password"`
	AuthorizedKey []AuthorizedKey `json:"authorized_key"`
	HostKey       []any           `json:"host_key"`
}

type RescueSetInput struct {
	OS            string
	Arch          int
	AuthorizedKey string
}

type LinuxResponse struct {
	Linux Linux `json:"linux"`
}

type Linux struct {
	ServerIP      string          `json:"server_ip"`
	ServerIPv6Net string          `json:"server_ipv6_net"`
	ServerNumber  int             `json:"server_number"`
	Dist          any             `json:"dist"` // unfortunately the API returns an array vs. a single value based on state (active/inactive)
	Arch          any             `json:"arch"` // unfortunately the API returns an array vs. a single value based on state (active/inactive)
	Lang          any             `json:"lang"` // unfortunately the API returns an array vs. a single value based on state (active/inactive)
	Active        bool            `json:"active"`
	Password      string          `json:"password"`
	AuthorizedKey []AuthorizedKey `json:"authorized_key"`
	HostKey       []any           `json:"host_key"`
}

type LinuxSetInput struct {
	Dist          string
	Arch          int
	Lang          string
	AuthorizedKey string
}
