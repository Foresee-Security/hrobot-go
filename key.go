package hrobot

import (
	"context"
	"net/http"
	"net/url"
)

// Key is an SSH public key stored on the account, available to authorize
// rescue-system and installation boots.
type Key struct {
	// Name is the operator-assigned label.
	Name string `json:"name"`
	// Fingerprint identifies the key and is the value the boot endpoints take
	// to authorize it.
	Fingerprint string `json:"fingerprint"`
	// Type is the key algorithm, such as "RSA", "ECDSA" or "ED25519".
	Type string `json:"type"`
	// Size is the key length in bits.
	Size int `json:"size"`
	// Data is the public key in OpenSSH format. It is not populated where a
	// key appears nested as an [AuthorizedKey].
	Data string `json:"data"`
	// CreatedAt is when the key was stored, formatted "2006-01-02 15:04:05".
	CreatedAt string `json:"created_at"`
}

// KeySetInput is the key [Client.KeySet] uploads.
type KeySetInput struct {
	// Name is the label to store the key under. Required.
	Name string
	// Data is the public key in OpenSSH format. Required.
	Data string
}

// keyResponse is the envelope a single key is wrapped in.
type keyResponse struct {
	Key Key `json:"key"`
}

// KeyGetList returns every SSH key stored on the account.
//
// An account with none gets an empty slice and a nil error. See [fetchList] for
// why that takes normalising.
func (c *Client) KeyGetList(ctx context.Context) ([]Key, error) {
	return fetchList(ctx, c, "key list", "/key", func(e keyResponse) Key { return e.Key })
}

// KeySet uploads an SSH public key and returns it as stored.
//
// It returns [ErrNilInput] without contacting the API if input is nil. An
// already-present key surfaces as an [Error] with [ErrorCodeKeyAlreadyExists].
func (c *Client) KeySet(ctx context.Context, input *KeySetInput) (*Key, error) {
	const op = "key set"

	if input == nil {
		return nil, ErrNilInput
	}

	form := url.Values{}
	form.Set("name", input.Name)
	form.Set("data", input.Data)

	resp, err := fetch[keyResponse](ctx, c, op, http.MethodPost, "/key", form)
	if err != nil {
		return nil, err
	}
	return &resp.Key, nil
}
