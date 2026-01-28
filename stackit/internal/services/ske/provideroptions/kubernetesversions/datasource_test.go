package kubernetesversions

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	skeutils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
)

func TestMapFields(t *testing.T) {
	expDeprecated1 := time.Date(2026, 1, 28, 8, 0, 0, 0, time.UTC)
	expDeprecated2 := time.Date(2026, 1, 14, 8, 0, 0, 0, time.UTC)

	expDeprecated1Str := expDeprecated1.Format(time.RFC3339)
	expDeprecated2Str := expDeprecated2.Format(time.RFC3339)

	tests := []struct {
		name     string
		input    *ske.ProviderOptions
		model    *Model
		expected *Model
		isValid  bool
	}{
		{
			name:     "nil input provider options",
			input:    nil,
			model:    &Model{},
			expected: &Model{}, // not used, we expect an error
			isValid:  false,
		},
		{
			name: "multiple versions realistic payload",
			input: &ske.ProviderOptions{
				KubernetesVersions: &[]ske.KubernetesVersion{
					{
						Version:        skeutils.Ptr("1.31.14"),
						State:          skeutils.Ptr("deprecated"),
						ExpirationDate: &expDeprecated1,
						FeatureGates:   &map[string]string{},
					},
					{
						Version:        skeutils.Ptr("1.32.10"),
						State:          skeutils.Ptr("deprecated"),
						ExpirationDate: &expDeprecated2,
						FeatureGates:   &map[string]string{},
					},
					{
						Version:        skeutils.Ptr("1.33.6"),
						State:          skeutils.Ptr("deprecated"),
						ExpirationDate: &expDeprecated2,
						FeatureGates:   &map[string]string{},
					},
					{
						Version:        skeutils.Ptr("1.34.2"),
						State:          skeutils.Ptr("deprecated"),
						ExpirationDate: &expDeprecated2,
						FeatureGates:   &map[string]string{},
					},
					{
						Version:        skeutils.Ptr("1.32.11"),
						State:          skeutils.Ptr("supported"),
						ExpirationDate: nil,
						FeatureGates:   &map[string]string{},
					},
					{
						Version:        skeutils.Ptr("1.33.7"),
						State:          skeutils.Ptr("supported"),
						ExpirationDate: nil,
						FeatureGates:   &map[string]string{},
					},
					{
						Version:        skeutils.Ptr("1.34.3"),
						State:          skeutils.Ptr("supported"),
						ExpirationDate: nil,
						FeatureGates:   &map[string]string{},
					},
				},
			},
			model: &Model{},
			expected: &Model{
				KubernetesVersions: types.ListValueMust(
					types.ObjectType{AttrTypes: kubernetesVersionType},
					[]attr.Value{
						types.ObjectValueMust(kubernetesVersionType, map[string]attr.Value{
							"version":         types.StringValue("1.31.14"),
							"state":           types.StringValue("deprecated"),
							"expiration_date": types.StringValue(expDeprecated1Str),
							"feature_gates": types.MapValueMust(
								types.StringType,
								map[string]attr.Value{},
							),
						}),
						types.ObjectValueMust(kubernetesVersionType, map[string]attr.Value{
							"version":         types.StringValue("1.32.10"),
							"state":           types.StringValue("deprecated"),
							"expiration_date": types.StringValue(expDeprecated2Str),
							"feature_gates": types.MapValueMust(
								types.StringType,
								map[string]attr.Value{},
							),
						}),
						types.ObjectValueMust(kubernetesVersionType, map[string]attr.Value{
							"version":         types.StringValue("1.33.6"),
							"state":           types.StringValue("deprecated"),
							"expiration_date": types.StringValue(expDeprecated2Str),
							"feature_gates": types.MapValueMust(
								types.StringType,
								map[string]attr.Value{},
							),
						}),
						types.ObjectValueMust(kubernetesVersionType, map[string]attr.Value{
							"version":         types.StringValue("1.34.2"),
							"state":           types.StringValue("deprecated"),
							"expiration_date": types.StringValue(expDeprecated2Str),
							"feature_gates": types.MapValueMust(
								types.StringType,
								map[string]attr.Value{},
							),
						}),
						types.ObjectValueMust(kubernetesVersionType, map[string]attr.Value{
							"version":         types.StringValue("1.32.11"),
							"state":           types.StringValue("supported"),
							"expiration_date": types.StringNull(),
							"feature_gates": types.MapValueMust(
								types.StringType,
								map[string]attr.Value{},
							),
						}),
						types.ObjectValueMust(kubernetesVersionType, map[string]attr.Value{
							"version":         types.StringValue("1.33.7"),
							"state":           types.StringValue("supported"),
							"expiration_date": types.StringNull(),
							"feature_gates": types.MapValueMust(
								types.StringType,
								map[string]attr.Value{},
							),
						}),
						types.ObjectValueMust(kubernetesVersionType, map[string]attr.Value{
							"version":         types.StringValue("1.34.3"),
							"state":           types.StringValue("supported"),
							"expiration_date": types.StringNull(),
							"feature_gates": types.MapValueMust(
								types.StringType,
								map[string]attr.Value{},
							),
						}),
					},
				),
			},
			isValid: true,
		},
		{
			name: "mixed fields with nil feature gates and nil state",
			input: &ske.ProviderOptions{
				KubernetesVersions: &[]ske.KubernetesVersion{
					{
						Version:        skeutils.Ptr("1.32.11"),
						State:          skeutils.Ptr("supported"),
						ExpirationDate: nil,
						FeatureGates: &map[string]string{
							"SomeGate": "foo",
						},
					},
					{
						Version:        nil,
						State:          nil,
						ExpirationDate: nil,
						FeatureGates:   nil,
					},
				},
			},
			model: &Model{},
			expected: &Model{
				KubernetesVersions: types.ListValueMust(
					types.ObjectType{AttrTypes: kubernetesVersionType},
					[]attr.Value{
						types.ObjectValueMust(kubernetesVersionType, map[string]attr.Value{
							"version":         types.StringValue("1.32.11"),
							"state":           types.StringValue("supported"),
							"expiration_date": types.StringNull(),
							"feature_gates": types.MapValueMust(
								types.StringType,
								map[string]attr.Value{
									"SomeGate": types.StringValue("foo"),
								},
							),
						}),
						types.ObjectValueMust(kubernetesVersionType, map[string]attr.Value{
							"version":         types.StringNull(),
							"state":           types.StringNull(),
							"expiration_date": types.StringNull(),
							// nil feature gates => empty map
							"feature_gates": types.MapValueMust(
								types.StringType,
								map[string]attr.Value{},
							),
						}),
					},
				),
			},
			isValid: true,
		},
		{
			name: "nil kubernetes versions slice",
			input: &ske.ProviderOptions{
				KubernetesVersions: nil,
			},
			model: &Model{},
			expected: &Model{
				KubernetesVersions: types.ListValueMust(
					types.ObjectType{AttrTypes: kubernetesVersionType},
					[]attr.Value{},
				),
			},
			isValid: true,
		},
		{
			name: "empty kubernetes versions slice",
			input: &ske.ProviderOptions{
				KubernetesVersions: &[]ske.KubernetesVersion{},
			},
			model: &Model{},
			expected: &Model{
				KubernetesVersions: types.ListValueMust(
					types.ObjectType{AttrTypes: kubernetesVersionType},
					[]attr.Value{},
				),
			},
			isValid: true,
		},
		{
			name: "feature gates empty map",
			input: &ske.ProviderOptions{
				KubernetesVersions: &[]ske.KubernetesVersion{
					{
						Version:        skeutils.Ptr("1.33.7"),
						State:          skeutils.Ptr("supported"),
						ExpirationDate: nil,
						FeatureGates:   &map[string]string{},
					},
				},
			},
			model: &Model{},
			expected: &Model{
				KubernetesVersions: types.ListValueMust(
					types.ObjectType{AttrTypes: kubernetesVersionType},
					[]attr.Value{
						types.ObjectValueMust(kubernetesVersionType, map[string]attr.Value{
							"version":         types.StringValue("1.33.7"),
							"state":           types.StringValue("supported"),
							"expiration_date": types.StringNull(),
							// empty map from API => empty map in Terraform
							"feature_gates": types.MapValueMust(
								types.StringType,
								map[string]attr.Value{},
							),
						}),
					},
				),
			},
			isValid: true,
		},
		{
			name: "nil model",
			input: &ske.ProviderOptions{
				KubernetesVersions: &[]ske.KubernetesVersion{
					{
						Version:        skeutils.Ptr("1.32.11"),
						State:          skeutils.Ptr("supported"),
						ExpirationDate: nil,
						FeatureGates:   &map[string]string{},
					},
				},
			},
			model:    nil,
			expected: nil, // not used, we expect an error
			isValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, tt.model)

			if tt.isValid && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.isValid && err == nil {
				t.Fatal("expected error but got none")
			}

			if tt.isValid {
				if diff := cmp.Diff(tt.expected, tt.model); diff != "" {
					t.Fatalf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
