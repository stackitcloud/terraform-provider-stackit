package kms

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kms "github.com/stackitcloud/stackit-sdk-go/services/kms/v1api"
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
					Id:          keyId,
					DisplayName: "display-name",
					Protection:  kms.PROTECTION_SOFTWARE,
					Algorithm:   kms.ALGORITHM_ECDSA_P256_SHA256,
					Purpose:     kms.PURPOSE_ASYMMETRIC_SIGN_VERIFY,
					AccessScope: kms.ACCESSSCOPE_PUBLIC,
					ImportOnly:  true,
				},
				region: "eu01",
			},
			expected: Model{
				Description: types.StringNull(),
				DisplayName: types.StringValue("display-name"),
				KeyRingId:   types.StringValue(keyRingId),
				KeyId:       types.StringValue(keyId),
				Id:          types.StringValue(fmt.Sprintf("%s,eu01,%s,%s", projectId, keyRingId, keyId)),
				ProjectId:   types.StringValue(projectId),
				Region:      types.StringValue("eu01"),
				Protection:  types.StringValue(string(kms.PROTECTION_SOFTWARE)),
				Algorithm:   types.StringValue(string(kms.ALGORITHM_ECDSA_P256_SHA256)),
				Purpose:     types.StringValue(string(kms.PURPOSE_ASYMMETRIC_SIGN_VERIFY)),
				AccessScope: types.StringValue(string(kms.ACCESSSCOPE_PUBLIC)),
				ImportOnly:  types.BoolValue(true),
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
					Id:          keyId,
					Description: new("descr"),
					DisplayName: "name",
					ImportOnly:  true,
					Protection:  kms.PROTECTION_SOFTWARE,
					Algorithm:   kms.ALGORITHM_AES_256_GCM,
					Purpose:     kms.PURPOSE_MESSAGE_AUTHENTICATION_CODE,
					AccessScope: kms.ACCESSSCOPE_SNA,
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
			description: "nil_response",
			args: args{
				state: Model{},
				input: nil,
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
				DisplayName: "name",
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
				DisplayName: "",
				Description: new(""),
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
