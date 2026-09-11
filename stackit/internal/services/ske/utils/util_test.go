package utils

import (
	"testing"

	ske "github.com/stackitcloud/stackit-sdk-go/services/ske/v2api"
)

func TestIsEmptyNetwork(t *testing.T) {
	tests := []struct {
		name  string
		input *ske.Network
		want  bool
	}{
		{
			name:  "nil network",
			input: nil,
			want:  true,
		},
		{
			name:  "empty",
			input: &ske.Network{},
			want:  true,
		},
		{
			name: "only AdditionalProperties are set",
			input: &ske.Network{
				AdditionalProperties: map[string]interface{}{
					"foo": "bar",
				},
			},
			want: true,
		},
		{
			name: "id set",
			input: &ske.Network{
				Id: new("network-id"),
			},
			want: false,
		},
		{
			name: "control plane set",
			input: &ske.Network{
				ControlPlane: &ske.V2ControlPlaneNetwork{},
			},
			want: false,
		},
		{
			name: "id and control plane set",
			input: &ske.Network{
				Id: new("network-id"),
				ControlPlane: &ske.V2ControlPlaneNetwork{
					AccessScope: ske.ACCESSSCOPE_SNA.Ptr(),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEmptyNetwork(tt.input); got != tt.want {
				t.Errorf("IsEmptyNetwork() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsEmptyExtension(t *testing.T) {
	tests := []struct {
		name  string
		input *ske.Extension
		want  bool
	}{
		{
			name:  "nil extension",
			input: nil,
			want:  true,
		},
		{
			name:  "empty",
			input: &ske.Extension{},
			want:  true,
		},
		{
			name: "only AdditionalProperties are set",
			input: &ske.Extension{
				AdditionalProperties: map[string]interface{}{
					"foo": "bar",
				},
			},
			want: true,
		},
		{
			name: "acl set",
			input: &ske.Extension{
				Acl: ske.NewACL([]string{"1.1.1.0/24"}, true),
			},
			want: false,
		},
		{
			name: "observability set",
			input: &ske.Extension{
				Observability: ske.NewObservability(true, "instance-id"),
			},
			want: false,
		},
		{
			name: "dns set",
			input: &ske.Extension{
				Dns: ske.NewDNS(true),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEmptyExtension(tt.input); got != tt.want {
				t.Errorf("IsEmptyExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}
