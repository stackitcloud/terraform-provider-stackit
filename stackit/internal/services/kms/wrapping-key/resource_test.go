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
	projectId     = uuid.NewString()
	keyRingId     = uuid.NewString()
	wrappingKeyId = uuid.NewString()
)

func TestMapFields(t *testing.T) {
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
					Id:          utils.Ptr("wid"),
					AccessScope: utils.Ptr(kms.ACCESSSCOPE_PUBLIC),
					Algorithm:   utils.Ptr(kms.WRAPPINGALGORITHM__2048_OAEP_SHA256),
					Purpose:     utils.Ptr(kms.WRAPPINGPURPOSE_ASYMMETRIC_KEY),
					Protection:  utils.Ptr(kms.PROTECTION_SOFTWARE),
				},
				region: "eu01",
			},
			expected: &Model{
				Description:   types.StringNull(),
				DisplayName:   types.StringNull(),
				KeyRingId:     types.StringValue(keyRingId),
				Id:            types.StringValue(fmt.Sprintf("%s,eu01,%s,%s", projectId, keyRingId, wrappingKeyId)),
				ProjectId:     types.StringValue(projectId),
				Region:        types.StringValue("eu01"),
				WrappingKeyId: types.StringValue(wrappingKeyId),
				AccessScope:   types.StringValue(string(kms.ACCESSSCOPE_PUBLIC)),
				Algorithm:     types.StringValue(string(kms.WRAPPINGALGORITHM__2048_OAEP_SHA256)),
				Purpose:       types.StringValue(string(kms.WRAPPINGPURPOSE_ASYMMETRIC_KEY)),
				Protection:    types.StringValue(string(kms.PROTECTION_SOFTWARE)),
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
					Description: utils.Ptr("descr"),
					DisplayName: utils.Ptr("name"),
					Id:          utils.Ptr(wrappingKeyId),
					AccessScope: utils.Ptr(kms.ACCESSSCOPE_PUBLIC),
					Algorithm:   utils.Ptr(kms.WRAPPINGALGORITHM__2048_OAEP_SHA256),
					Purpose:     utils.Ptr(kms.WRAPPINGPURPOSE_ASYMMETRIC_KEY),
					Protection:  utils.Ptr(kms.PROTECTION_SOFTWARE),
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
				Algorithm:     types.StringValue(string(kms.WRAPPINGALGORITHM__2048_OAEP_SHA256)),
				Purpose:       types.StringValue(string(kms.WRAPPINGPURPOSE_ASYMMETRIC_KEY)),
				Protection:    types.StringValue(string(kms.PROTECTION_SOFTWARE)),
			},
			isValid: true,
		},
		{
			description: "nil_response_field",
			args: args{
				state: &Model{},
				input: &kms.WrappingKey{
					Id: nil,
				},
			},
			expected: &Model{},
			isValid:  false,
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
		{
			description: "no_resource_id",
			args: args{
				state: &Model{
					Region:    types.StringValue("eu01"),
					ProjectId: types.StringValue("pid"),
				},
				input: &kms.WrappingKey{},
			},
			expected: &Model{},
			isValid:  false,
		},
		{
			// TODO: test for workaround - remove once STACKITKMS-377 is resolved
			description: "empty description string",
			args: args{
				state: &Model{
					KeyRingId:     types.StringValue(keyRingId),
					ProjectId:     types.StringValue(projectId),
					WrappingKeyId: types.StringValue(wrappingKeyId),
				},
				input: &kms.WrappingKey{
					Description: utils.Ptr(""),
					Id:          utils.Ptr(wrappingKeyId),
					AccessScope: utils.Ptr(kms.ACCESSSCOPE_PUBLIC),
					Algorithm:   utils.Ptr(kms.WRAPPINGALGORITHM__2048_OAEP_SHA256),
					Purpose:     utils.Ptr(kms.WRAPPINGPURPOSE_ASYMMETRIC_KEY),
					Protection:  utils.Ptr(kms.PROTECTION_SOFTWARE),
				},
				region: "eu02",
			},
			expected: &Model{
				Description:   types.StringNull(),
				KeyRingId:     types.StringValue(keyRingId),
				Id:            types.StringValue(fmt.Sprintf("%s,eu02,%s,%s", projectId, keyRingId, wrappingKeyId)),
				ProjectId:     types.StringValue(projectId),
				Region:        types.StringValue("eu02"),
				WrappingKeyId: types.StringValue(wrappingKeyId),
				AccessScope:   types.StringValue(string(kms.ACCESSSCOPE_PUBLIC)),
				Algorithm:     types.StringValue(string(kms.WRAPPINGALGORITHM__2048_OAEP_SHA256)),
				Purpose:       types.StringValue(string(kms.WRAPPINGPURPOSE_ASYMMETRIC_KEY)),
				Protection:    types.StringValue(string(kms.PROTECTION_SOFTWARE)),
			},
			isValid: true,
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
