package kms

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
)

const testRegion = "eu01"

func TestMapFields(t *testing.T) {
	type args struct {
		state Model
		input *kms.Key
	}
	tests := []struct {
		description string
		args        args
		expected    Model
		isValid     bool
	}{
		{
			description: "default values",
			args: args{
				state: Model{
					KeyId:     types.StringValue("kid"),
					KeyRingId: types.StringValue("krid"),
					ProjectId: types.StringValue("pid"),
				},
				input: &kms.Key{
					Id:          utils.Ptr("kid"),
					Protection:  utils.Ptr(kms.PROTECTION_SOFTWARE),
					Algorithm:   utils.Ptr(kms.ALGORITHM_ECDSA_P256_SHA256),
					Purpose:     utils.Ptr(kms.PURPOSE_ASYMMETRIC_SIGN_VERIFY),
					AccessScope: utils.Ptr(kms.ACCESSSCOPE_PUBLIC),
				},
			},
			expected: Model{
				Description: types.StringNull(),
				DisplayName: types.StringNull(),
				KeyRingId:   types.StringValue("krid"),
				KeyId:       types.StringValue("kid"),
				Id:          types.StringValue("pid,eu01,krid,kid"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue(testRegion),
				Protection:  types.StringValue(string(kms.PROTECTION_SOFTWARE)),
				Algorithm:   types.StringValue(string(kms.ALGORITHM_ECDSA_P256_SHA256)),
				Purpose:     types.StringValue(string(kms.PURPOSE_ASYMMETRIC_SIGN_VERIFY)),
				AccessScope: types.StringValue(string(kms.ACCESSSCOPE_PUBLIC)),
			},
			isValid: true,
		},
		{
			description: "values_ok",
			args: args{
				state: Model{
					KeyId:     types.StringValue("kid"),
					KeyRingId: types.StringValue("krid"),
					ProjectId: types.StringValue("pid"),
				},
				input: &kms.Key{
					Id:          utils.Ptr("kid"),
					Description: utils.Ptr("descr"),
					DisplayName: utils.Ptr("name"),
					ImportOnly:  utils.Ptr(true),
					Protection:  utils.Ptr(kms.PROTECTION_SOFTWARE),
					Algorithm:   utils.Ptr(kms.ALGORITHM_AES_256_GCM),
					Purpose:     utils.Ptr(kms.PURPOSE_MESSAGE_AUTHENTICATION_CODE),
					AccessScope: utils.Ptr(kms.ACCESSSCOPE_SNA),
				},
			},
			expected: Model{
				Description: types.StringValue("descr"),
				DisplayName: types.StringValue("name"),
				KeyId:       types.StringValue("kid"),
				KeyRingId:   types.StringValue("krid"),
				Id:          types.StringValue("pid,eu01,krid,kid"),
				ProjectId:   types.StringValue("pid"),
				Region:      types.StringValue(testRegion),
				ImportOnly:  types.BoolValue(true),
				Protection:  types.StringValue(string(kms.PROTECTION_SOFTWARE)),
				Algorithm:   types.StringValue(string(kms.ALGORITHM_AES_256_GCM)),
				Purpose:     types.StringValue(string(kms.PURPOSE_MESSAGE_AUTHENTICATION_CODE)),
				AccessScope: types.StringValue(string(kms.ACCESSSCOPE_SNA)),
			},
			isValid: true,
		},
		{
			description: "nil_response_field",
			args: args{
				state: Model{},
				input: &kms.Key{
					Id: nil,
				},
			},
			expected: Model{},
			isValid:  false,
		},
		{
			description: "nil_response",
			args: args{
				state: Model{},
				input: nil,
			},
			expected: Model{},
			isValid:  false,
		},
		{
			description: "no_resource_id",
			args: args{
				state: Model{
					Region:    types.StringValue(testRegion),
					ProjectId: types.StringValue("pid"),
				},
				input: &kms.Key{},
			},
			expected: Model{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId: tt.expected.ProjectId,
				KeyRingId: tt.expected.KeyRingId,
			}
			err := mapFields(tt.args.input, state, testRegion)
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
		expected    *kms.CreateKeyPayload
		isValid     bool
	}{
		{
			description: "default_values",
			input:       &Model{},
			expected:    &kms.CreateKeyPayload{},
			isValid:     true,
		},
		{
			description: "simple_values",
			input: &Model{
				DisplayName: types.StringValue("name"),
			},
			expected: &kms.CreateKeyPayload{
				DisplayName: utils.Ptr("name"),
			},
			isValid: true,
		},
		{
			description: "null_fields",
			input: &Model{
				DisplayName: types.StringValue(""),
				Description: types.StringValue(""),
			},
			expected: &kms.CreateKeyPayload{
				DisplayName: utils.Ptr(""),
				Description: utils.Ptr(""),
			},
			isValid: true,
		},
		{
			description: "nil_model",
			input:       nil,
			expected:    nil,
			isValid:     false,
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
