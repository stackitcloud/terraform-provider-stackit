package kms

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
)

var (
	keyId     = uuid.NewString()
	keyRingId = uuid.NewString()
	projectId = uuid.NewString()
)

func TestMapFields(t *testing.T) {
	type args struct {
		state  Model
		input  *kms.Key
		region string
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
					KeyId:     types.StringValue(keyId),
					KeyRingId: types.StringValue(keyRingId),
					ProjectId: types.StringValue(projectId),
				},
				input: &kms.Key{
					Id:          utils.Ptr(keyId),
					Protection:  utils.Ptr(kms.PROTECTION_SOFTWARE),
					Algorithm:   utils.Ptr(kms.ALGORITHM_ECDSA_P256_SHA256),
					Purpose:     utils.Ptr(kms.PURPOSE_ASYMMETRIC_SIGN_VERIFY),
					AccessScope: utils.Ptr(kms.ACCESSSCOPE_PUBLIC),
				},
				region: "eu01",
			},
			expected: Model{
				Description: types.StringNull(),
				DisplayName: types.StringNull(),
				KeyRingId:   types.StringValue(keyRingId),
				KeyId:       types.StringValue(keyId),
				Id:          types.StringValue(fmt.Sprintf("%s,eu01,%s,%s", projectId, keyRingId, keyId)),
				ProjectId:   types.StringValue(projectId),
				Region:      types.StringValue("eu01"),
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
					KeyId:     types.StringValue(keyId),
					KeyRingId: types.StringValue(keyRingId),
					ProjectId: types.StringValue(projectId),
				},
				input: &kms.Key{
					Id:          utils.Ptr(keyId),
					Description: utils.Ptr("descr"),
					DisplayName: utils.Ptr("name"),
					ImportOnly:  utils.Ptr(true),
					Protection:  utils.Ptr(kms.PROTECTION_SOFTWARE),
					Algorithm:   utils.Ptr(kms.ALGORITHM_AES_256_GCM),
					Purpose:     utils.Ptr(kms.PURPOSE_MESSAGE_AUTHENTICATION_CODE),
					AccessScope: utils.Ptr(kms.ACCESSSCOPE_SNA),
				},
				region: "eu01",
			},
			expected: Model{
				Description: types.StringValue("descr"),
				DisplayName: types.StringValue("name"),
				KeyId:       types.StringValue(keyId),
				KeyRingId:   types.StringValue(keyRingId),
				Id:          types.StringValue(fmt.Sprintf("%s,eu01,%s,%s", projectId, keyRingId, keyId)),
				ProjectId:   types.StringValue(projectId),
				Region:      types.StringValue("eu01"),
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
					Region:    types.StringValue("eu01"),
					ProjectId: types.StringValue(projectId),
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
			err := mapFields(tt.args.input, state, tt.args.region)
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
