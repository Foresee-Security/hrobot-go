package hrobot_test

import (
	"encoding/json"
	"slices"
	"testing"

	hrobot "github.com/Foresee-Security/hrobot-go"
)

func TestStringListUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{name: "array", input: `["linux","freebsd","vkvm"]`, want: []string{"linux", "freebsd", "vkvm"}},
		{name: "bare scalar", input: `"linux"`, want: []string{"linux"}},
		{name: "empty array", input: `[]`, want: []string{}},
		{name: "empty string", input: `""`, want: []string{""}},
		{name: "null", input: `null`, want: nil},
		{name: "leading whitespace", input: "  [\"linux\"]", want: []string{"linux"}},
		{name: "rejects a number", input: `5`, wantErr: true},
		{name: "rejects an object", input: `{"os":"linux"}`, wantErr: true},
		{name: "rejects a malformed array", input: `["linux",]`, wantErr: true},
		{name: "rejects an array of numbers", input: `[1,2]`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got hrobot.StringList
			err := json.Unmarshal([]byte(tc.input), &got)

			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if !slices.Equal([]string(got), tc.want) {
				t.Errorf("got %#v, want %#v", []string(got), tc.want)
			}
		})
	}
}

func TestIntListUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
	}{
		{name: "array", input: `[64,32]`, want: []int{64, 32}},
		{name: "bare scalar", input: `64`, want: []int{64}},
		{name: "zero", input: `0`, want: []int{0}},
		{name: "empty array", input: `[]`, want: []int{}},
		{name: "null", input: `null`, want: nil},
		{name: "rejects a string", input: `"64"`, wantErr: true},
		{name: "rejects a float", input: `64.5`, wantErr: true},
		{name: "rejects an array of strings", input: `["64"]`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got hrobot.IntList
			err := json.Unmarshal([]byte(tc.input), &got)

			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if !slices.Equal([]int(got), tc.want) {
				t.Errorf("got %#v, want %#v", []int(got), tc.want)
			}
		})
	}
}

// TestFlexibleListsInsideAStruct covers the shape these types actually appear
// in, where the surrounding object decodes normally around them.
func TestFlexibleListsInsideAStruct(t *testing.T) {
	t.Parallel()

	var target struct {
		OS   hrobot.StringList `json:"os"`
		Arch hrobot.IntList    `json:"arch"`
	}

	const doc = `{"os":"linux","arch":[64,32]}`
	if err := json.Unmarshal([]byte(doc), &target); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !slices.Equal([]string(target.OS), []string{"linux"}) {
		t.Errorf("OS = %v, want [linux]", target.OS)
	}
	if !slices.Equal([]int(target.Arch), []int{64, 32}) {
		t.Errorf("Arch = %v, want [64 32]", target.Arch)
	}
}
