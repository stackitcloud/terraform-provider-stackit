package keypair

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *iaas.Keypair
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				Name: types.StringValue("name"),
			},
			&iaas.Keypair{
				Name: utils.Ptr("name"),
			},
			Model{
				Id:          types.StringValue("name"),
				Name:        types.StringValue("name"),
				PublicKey:   types.StringNull(),
				Fingerprint: types.StringNull(),
				Labels:      types.MapNull(types.StringType),
			},
			true,
		},
		{
			"simple_values",
			Model{
				Name: types.StringValue("name"),
			},
			&iaas.Keypair{
				Name:        utils.Ptr("name"),
				PublicKey:   utils.Ptr("public_key"),
				Fingerprint: utils.Ptr("fingerprint"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			Model{
				Id:          types.StringValue("name"),
				Name:        types.StringValue("name"),
				PublicKey:   types.StringValue("public_key"),
				Fingerprint: types.StringValue("fingerprint"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			true,
		},
		{
			"empty_labels",
			Model{
				Name: types.StringValue("name"),
			},
			&iaas.Keypair{
				Name:        utils.Ptr("name"),
				PublicKey:   utils.Ptr("public_key"),
				Fingerprint: utils.Ptr("fingerprint"),
				Labels:      &map[string]interface{}{},
			},
			Model{
				Id:          types.StringValue("name"),
				Name:        types.StringValue("name"),
				PublicKey:   types.StringValue("public_key"),
				Fingerprint: types.StringValue("fingerprint"),
				Labels:      types.MapNull(types.StringType),
			},
			true,
		},
		{
			"response_nil_fail",
			Model{},
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{},
			&iaas.Keypair{
				PublicKey:   utils.Ptr("public_key"),
				Fingerprint: utils.Ptr("fingerprint"),
				Labels:      &map[string]interface{}{},
			},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.state, tt.expected)
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
		expected    *iaas.CreateKeyPairPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name:      types.StringValue("name"),
				PublicKey: types.StringValue("public_key"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key1": types.StringValue("value1"),
					"key2": types.StringValue("value2"),
				}),
			},
			&iaas.CreateKeyPairPayload{
				Name:      utils.Ptr("name"),
				PublicKey: utils.Ptr("public_key"),
				Labels: &map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaas.NullableString{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *iaas.UpdateKeyPairPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name:      types.StringValue("name"),
				PublicKey: types.StringValue("public_key"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key1": types.StringValue("value1"),
					"key2": types.StringValue("value2"),
				}),
			},
			&iaas.UpdateKeyPairPayload{
				Labels: &map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input, types.MapNull(types.StringType))
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaas.NullableString{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
