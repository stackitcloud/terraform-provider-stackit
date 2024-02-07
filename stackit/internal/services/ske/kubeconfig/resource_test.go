package ske

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *ske.Kubeconfig
		expected    Model
		isValid     bool
	}{
		{
			"simple_values",
			&ske.Kubeconfig{
				ExpirationTimestamp: utils.Ptr("2024-02-07T16:42:12Z"),
				Kubeconfig:          utils.Ptr("kubeconfig"),
			},
			Model{
				ClusterName: types.StringValue("name"),
				ProjectId:   types.StringValue("pid"),
				Kubeconfig:  types.StringValue("kubeconfig"),
				Expiration:  types.Int64Null(),
				ExpiresAt:   types.StringValue("2024-02-07T16:42:12Z"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
		{
			"empty_kubeconfig",
			&ske.Kubeconfig{},
			Model{},
			false,
		},
		{
			"no_kubeconfig_field",
			&ske.Kubeconfig{
				ExpirationTimestamp: utils.Ptr("2024-02-07T16:42:12Z"),
			},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:   tt.expected.ProjectId,
				ClusterName: tt.expected.ClusterName,
			}
			err := mapFields(tt.input, state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected, cmpopts.IgnoreFields(Model{}, "Id")) // Id includes a random uuid
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *ske.CreateKubeconfigPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&ske.CreateKubeconfigPayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				Expiration: types.Int64Value(3600),
			},
			&ske.CreateKubeconfigPayload{
				ExpirationSeconds: utils.Ptr("3600"),
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
