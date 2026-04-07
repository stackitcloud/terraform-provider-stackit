package kms

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kms "github.com/stackitcloud/stackit-sdk-go/services/kms/v1api"
)

var (
	projectId     = uuid.NewString()
	keyRingId     = uuid.NewString()
	wrappingKeyId = uuid.NewString()
)

func TestMapFields(t *testing.T) {
	createdAt := time.Now()
	expiresAt := time.Now().Add(time.Hour)

	type args struct {
		state  *Model
		input  *kms.WrappingKey
		region string
	}
	tests := []struct {
		description string
		args        args
		expected    *Model
		isValid     bool
	}{
		{
			description: "default values",
			args: args{
				state: &Model{
					KeyRingId:     types.StringValue(keyRingId),
					ProjectId:     types.StringValue(projectId),
					WrappingKeyId: types.StringValue(wrappingKeyId),
				},
				input: &kms.WrappingKey{
					Id:          "wid",
					DisplayName: "display-name",
					AccessScope: kms.ACCESSSCOPE_PUBLIC,
					Algorithm:   kms.WRAPPINGALGORITHM_RSA_2048_OAEP_SHA256,
					Purpose:     kms.WRAPPINGPURPOSE_WRAP_ASYMMETRIC_KEY,
					Protection:  kms.PROTECTION_SOFTWARE,
					PublicKey:   new("public-key"),
					ExpiresAt:   expiresAt,
					CreatedAt:   createdAt,
				},
				region: "eu01",
			},
			expected: &Model{
				Description:   types.StringNull(),
				DisplayName:   types.StringValue("display-name"),
				KeyRingId:     types.StringValue(keyRingId),
				Id:            types.StringValue(fmt.Sprintf("%s,eu01,%s,%s", projectId, keyRingId, wrappingKeyId)),
				ProjectId:     types.StringValue(projectId),
				Region:        types.StringValue("eu01"),
				WrappingKeyId: types.StringValue(wrappingKeyId),
				AccessScope:   types.StringValue(string(kms.ACCESSSCOPE_PUBLIC)),
				Algorithm:     types.StringValue(string(kms.WRAPPINGALGORITHM_RSA_2048_OAEP_SHA256)),
				Purpose:       types.StringValue(string(kms.WRAPPINGPURPOSE_WRAP_ASYMMETRIC_KEY)),
				Protection:    types.StringValue(string(kms.PROTECTION_SOFTWARE)),
				PublicKey:     types.StringValue("public-key"),
				ExpiresAt:     types.StringValue(expiresAt.Format(time.RFC3339)),
				CreatedAt:     types.StringValue(createdAt.Format(time.RFC3339)),
			},
			isValid: true,
		},
		{
			description: "values_ok",
			args: args{
				state: &Model{
					KeyRingId:     types.StringValue(keyRingId),
					ProjectId:     types.StringValue(projectId),
					WrappingKeyId: types.StringValue(wrappingKeyId),
				},
				input: &kms.WrappingKey{
					Description: new("descr"),
					DisplayName: "name",
					Id:          wrappingKeyId,
					AccessScope: kms.ACCESSSCOPE_PUBLIC,
					Algorithm:   kms.WRAPPINGALGORITHM_RSA_2048_OAEP_SHA256,
					Purpose:     kms.WRAPPINGPURPOSE_WRAP_ASYMMETRIC_KEY,
					Protection:  kms.PROTECTION_SOFTWARE,
					PublicKey:   new("public-key"),
					ExpiresAt:   expiresAt,
					CreatedAt:   createdAt,
				},
				region: "eu02",
			},
			expected: &Model{
				Description:   types.StringValue("descr"),
				DisplayName:   types.StringValue("name"),
				KeyRingId:     types.StringValue(keyRingId),
				Id:            types.StringValue(fmt.Sprintf("%s,eu02,%s,%s", projectId, keyRingId, wrappingKeyId)),
				ProjectId:     types.StringValue(projectId),
				Region:        types.StringValue("eu02"),
				WrappingKeyId: types.StringValue(wrappingKeyId),
				AccessScope:   types.StringValue(string(kms.ACCESSSCOPE_PUBLIC)),
				Algorithm:     types.StringValue(string(kms.WRAPPINGALGORITHM_RSA_2048_OAEP_SHA256)),
				Purpose:       types.StringValue(string(kms.WRAPPINGPURPOSE_WRAP_ASYMMETRIC_KEY)),
				Protection:    types.StringValue(string(kms.PROTECTION_SOFTWARE)),
				PublicKey:     types.StringValue("public-key"),
				ExpiresAt:     types.StringValue(expiresAt.Format(time.RFC3339)),
				CreatedAt:     types.StringValue(createdAt.Format(time.RFC3339)),
			},
			isValid: true,
		},
		{
			description: "nil_response",
			args: args{
				state: &Model{},
				input: nil,
			},
			expected: &Model{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(tt.args.input, tt.args.state, tt.args.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}

			if tt.isValid {
				diff := cmp.Diff(tt.args.state, tt.expected)
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
			&kms.CreateWrappingKeyPayload{
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
