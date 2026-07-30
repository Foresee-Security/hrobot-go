package hrobot_test

import (
	"errors"
	"net/http"
	"testing"

	hrobot "github.com/Foresee-Security/hrobot-go"
)

func TestKeyGetList(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "key_list.json"))

	keys, err := c.KeyGetList(t.Context())
	if err != nil {
		t.Fatalf("KeyGetList: %v", err)
	}

	wantRequest(t, rec.only(t), http.MethodGet, "/key")

	if len(keys) != 2 {
		t.Fatalf("got %d keys, want 2", len(keys))
	}
	if keys[0].Name != "key1" || keys[0].Type != "ECDSA" || keys[0].Size != 521 {
		t.Errorf("keys[0] = %+v, want key1/ECDSA/521", keys[0])
	}
	// created_at was absent from the struct, so the API's value was dropped.
	if keys[0].CreatedAt != "2021-12-31 23:59:59" {
		t.Errorf("keys[0].CreatedAt = %q, want the value the API sent", keys[0].CreatedAt)
	}
	if keys[1].Name != "key2" || keys[1].Type != "ED25519" {
		t.Errorf("keys[1] = %+v, want key2/ED25519", keys[1])
	}
}

func TestKeySet(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, serveFixture(t, "key_set.json"))

	input := &hrobot.KeySetInput{Name: "NewKey", Data: "ssh-rsa AAAAB3NzaC1yc+..."}
	key, err := c.KeySet(t.Context(), input)
	if err != nil {
		t.Fatalf("KeySet: %v", err)
	}

	got := rec.only(t)
	wantRequest(t, got, http.MethodPost, "/key")
	wantBody := "data=ssh-rsa+AAAAB3NzaC1yc%2B...&name=NewKey"
	if got.Body != wantBody {
		t.Errorf("body = %q, want %q", got.Body, wantBody)
	}

	if key.Name != "NewKey" || key.Size != 8192 {
		t.Errorf("key = %+v, want NewKey/8192", key)
	}
}

func TestKeySetAlreadyExists(t *testing.T) {
	t.Parallel()

	body := `{"error":{"code":"KEY_ALREADY_EXISTS","message":"key already exists","status":409}}`
	c, _ := newServer(t, serveBody(http.StatusConflict, body))

	_, err := c.KeySet(t.Context(), &hrobot.KeySetInput{Name: "dup", Data: "ssh-rsa x"})
	if !hrobot.IsError(err, hrobot.ErrorCodeKeyAlreadyExists) {
		t.Fatalf("IsError(err, ErrorCodeKeyAlreadyExists) = false, err = %v", err)
	}
}

func TestKeySetRejectsNilInput(t *testing.T) {
	t.Parallel()

	c, rec := newServer(t, unreachable(t))

	_, err := c.KeySet(t.Context(), nil)
	if !errors.Is(err, hrobot.ErrNilInput) {
		t.Fatalf("error = %v, want ErrNilInput", err)
	}
	if n := rec.count(); n != 0 {
		t.Errorf("sent %d requests, want 0", n)
	}
}
