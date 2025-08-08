package kms

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
	"testing"
)

func TestMapFields(t *testing.T) {
	const testRegion = "eu01"
	tests := []struct {
		description string
		state       Model
		input       *kms.WrappingKey
		expected    Model
		isValid     bool
	}{
		{
			"default values",
			Model{
				KeyRingId:     types.StringValue("krid"),
				ProjectId:     types.StringValue("pid"),
				WrappingKeyId: types.StringValue("wid"),
			},
			&kms.WrappingKey{
				Id: utils.Ptr("wid"),
			},
			Model{
				Description:   types.StringNull(),
				DisplayName:   types.StringNull(),
				KeyRingId:     types.StringValue("krid"),
				Id:            types.StringValue("pid,wid"),
				ProjectId:     types.StringValue("pid"),
				Region:        types.StringValue(testRegion),
				WrappingKeyId: types.StringValue("wid"),
			},
			true,
		},
		{
			"values_ok",
			Model{
				KeyRingId:     types.StringValue("krid"),
				ProjectId:     types.StringValue("pid"),
				WrappingKeyId: types.StringValue("wid"),
			},
			&kms.WrappingKey{
				Description: utils.Ptr("descr"),
				DisplayName: utils.Ptr("name"),
				Id:          utils.Ptr("wid"),
			},
			Model{
				Description:   types.StringValue("descr"),
				DisplayName:   types.StringValue("name"),
				KeyRingId:     types.StringValue("krid"),
				Id:            types.StringValue("pid,wid"),
				ProjectId:     types.StringValue("pid"),
				Region:        types.StringValue(testRegion),
				WrappingKeyId: types.StringValue("wid"),
			},
			true,
		},
		{
			"nil_response_field",
			Model{},
			&kms.WrappingKey{
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
			&kms.WrappingKey{},
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
		expected    *kms.CreateWrappingKeyPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&kms.CreateWrappingKeyPayload{},
			true,
		},
		{
			"simple_values",
			&Model{
				DisplayName: types.StringValue("name"),
			},
			&kms.CreateWrappingKeyPayload{
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
			&kms.CreateWrappingKeyPayload{
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
