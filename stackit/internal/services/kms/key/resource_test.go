package kms

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
)

func TestMapFields(t *testing.T) {
	const testRegion = "eu01"
	tests := []struct {
		description string
		state       Model
		input       *kms.Key
		expected    Model
		isValid     bool
	}{
		{
			"default values",
			Model{
				KeyId:     types.StringValue("kid"),
				KeyRingId: types.StringValue("krid"),
				ProjectId: types.StringValue("pid"),
			},
			&kms.Key{
				Id: utils.Ptr("kid"),
			},
			Model{
				Description: types.StringNull(),
				DisplayName: types.StringNull(),
				KeyRingId:   types.StringValue("krid"),
				KeyId:       types.StringValue("kid"),
				Id:          types.StringValue("pid,kid"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue(testRegion),
			},
			true,
		},
		{
			"values_ok",
			Model{
				KeyId:     types.StringValue("kid"),
				KeyRingId: types.StringValue("krid"),
				ProjectId: types.StringValue("pid"),
			},
			&kms.Key{
				Description: utils.Ptr("descr"),
				DisplayName: utils.Ptr("name"),
				Id:          utils.Ptr("kid"),
			},
			Model{
				Description: types.StringValue("descr"),
				DisplayName: types.StringValue("name"),
				KeyId:       types.StringValue("kid"),
				KeyRingId:   types.StringValue("krid"),
				Id:          types.StringValue("pid,kid"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue(testRegion),
			},
			true,
		},
		{
			"nil_response_field",
			Model{},
			&kms.Key{
				Id: nil,
			},
			Model{},
			false,
		},
		{
			"nil_response",
			Model{},
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{
				Region:    types.StringValue(testRegion),
				ProjectId: types.StringValue("pid"),
			},
			&kms.Key{},
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
					fmt.Println("state: ", state, " expected: ", tt.expected)
					t.Fatalf("Data does not match")
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *kms.CreateKeyPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&kms.CreateKeyPayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				DisplayName: types.StringValue("name"),
			},
			&kms.CreateKeyPayload{
				DisplayName: utils.Ptr("name"),
			},
			true,
		},
		{
			"null_fields",
			&Model{
				DisplayName: types.StringValue(""),
				Description: types.StringValue(""),
			},
			&kms.CreateKeyPayload{
				DisplayName: utils.Ptr(""),
				Description: utils.Ptr(""),
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
