package kms

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kms "github.com/stackitcloud/stackit-sdk-go/services/kms/v1api"
)

const testRegion = "eu01"

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *kms.KeyRing
		expected    Model
		isValid     bool
	}{
		{
			"default values",
			Model{
				KeyRingId: types.StringValue("krid"),
				ProjectId: types.StringValue("pid"),
			},
			&kms.KeyRing{
				Id:          "krid",
				DisplayName: "display-name",
			},
			Model{
				Description: types.StringNull(),
				DisplayName: types.StringValue("display-name"),
				KeyRingId:   types.StringValue("krid"),
				Id:          types.StringValue("pid,eu01,krid"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue(testRegion),
			},
			true,
		},
		{
			"values_ok",
			Model{
				KeyRingId: types.StringValue("krid"),
				ProjectId: types.StringValue("pid"),
			},
			&kms.KeyRing{
				Description: new("descr"),
				DisplayName: "name",
				Id:          "krid",
			},
			Model{
				Description: types.StringValue("descr"),
				DisplayName: types.StringValue("name"),
				KeyRingId:   types.StringValue("krid"),
				Id:          types.StringValue("pid,eu01,krid"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue(testRegion),
			},
			true,
		},
		{
			"nil_response",
			Model{},
			nil,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId: tt.expected.ProjectId,
				KeyRingId: tt.expected.KeyRingId,
			}
			err := mapFields(tt.input, state, testRegion)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected)
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
		expected    *kms.CreateKeyRingPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&kms.CreateKeyRingPayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				DisplayName: types.StringValue("name"),
			},
			&kms.CreateKeyRingPayload{
				DisplayName: "name",
			},
			true,
		},
		{
			"null_fields",
			&Model{
				DisplayName: types.StringValue(""),
				Description: types.StringValue(""),
			},
			&kms.CreateKeyRingPayload{
				DisplayName: "",
				Description: new(""),
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
