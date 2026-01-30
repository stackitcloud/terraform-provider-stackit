package machineimages

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
	timestamp := time.Date(2026, 1, 14, 8, 0, 0, 0, time.UTC)
	expDate := timestamp.Format(time.RFC3339)

	tests := []struct {
		name     string
		input    *ske.ProviderOptions
		expected *Model
		isValid  bool
	}{
		{
			name:     "nil input provider options",
			input:    nil,
			expected: &Model{},
			isValid:  false,
		},
		{
			name: "single machine image single version full fields",
			input: &ske.ProviderOptions{
				MachineImages: &[]ske.MachineImage{
					{
						Name: skeutils.Ptr("flatcar"),
						Versions: &[]ske.MachineImageVersion{
							{
								Version:        skeutils.Ptr("4230.2.1"),
								State:          skeutils.Ptr("supported"),
								ExpirationDate: &timestamp,
								Cri: &[]ske.CRI{
									{
										Name: skeutils.Ptr(ske.CRINAME_CONTAINERD),
									},
								},
							},
						},
					},
				},
			},
			expected: &Model{
				MachineImages: types.ListValueMust(
					types.ObjectType{AttrTypes: machineImageType},
					[]attr.Value{
						types.ObjectValueMust(machineImageType, map[string]attr.Value{
							"name": types.StringValue("flatcar"),
							"versions": types.ListValueMust(
								types.ObjectType{AttrTypes: machineImageVersionType},
								[]attr.Value{
									types.ObjectValueMust(machineImageVersionType, map[string]attr.Value{
										"version":         types.StringValue("4230.2.1"),
										"state":           types.StringValue("supported"),
										"expiration_date": types.StringValue(expDate),
										"cri": types.ListValueMust(
											types.StringType,
											[]attr.Value{
												types.StringValue(string(ske.CRINAME_CONTAINERD)),
											},
										),
									}),
								},
							),
						}),
					},
				),
			},
			isValid: true,
		},
		{
			name: "single machine image multiple versions mixed fields",
			input: &ske.ProviderOptions{
				MachineImages: &[]ske.MachineImage{
					{
						Name: skeutils.Ptr("flatcar"),
						Versions: &[]ske.MachineImageVersion{
							{
								Version:        skeutils.Ptr("4230.2.1"),
								State:          skeutils.Ptr("supported"),
								ExpirationDate: &timestamp,
								Cri: &[]ske.CRI{
									{
										Name: skeutils.Ptr(ske.CRINAME_CONTAINERD),
									},
								},
							},
							{
								// nil version, nil state, no expiration date, no CRI
								Version:        nil,
								State:          nil,
								ExpirationDate: nil,
								Cri:            nil,
							},
						},
					},
				},
			},
			expected: &Model{
				MachineImages: types.ListValueMust(
					types.ObjectType{AttrTypes: machineImageType},
					[]attr.Value{
						types.ObjectValueMust(machineImageType, map[string]attr.Value{
							"name": types.StringValue("flatcar"),
							"versions": types.ListValueMust(
								types.ObjectType{AttrTypes: machineImageVersionType},
								[]attr.Value{
									types.ObjectValueMust(machineImageVersionType, map[string]attr.Value{
										"version":         types.StringValue("4230.2.1"),
										"state":           types.StringValue("supported"),
										"expiration_date": types.StringValue(expDate),
										"cri": types.ListValueMust(
											types.StringType,
											[]attr.Value{
												types.StringValue(string(ske.CRINAME_CONTAINERD)),
											},
										),
									}),
									types.ObjectValueMust(machineImageVersionType, map[string]attr.Value{
										"version":         types.StringNull(),
										"state":           types.StringNull(),
										"expiration_date": types.StringNull(),
										// nil CRI => empty list
										"cri": types.ListValueMust(
											types.StringType,
											[]attr.Value{},
										),
									}),
								},
							),
						}),
					},
				),
			},
			isValid: true,
		},
		{
			name: "multiple machine images mixed versions",
			input: &ske.ProviderOptions{
				MachineImages: &[]ske.MachineImage{
					{
						Name: skeutils.Ptr("flatcar"),
						Versions: &[]ske.MachineImageVersion{
							{
								Version:        skeutils.Ptr("4230.2.1"),
								State:          skeutils.Ptr("deprecated"),
								ExpirationDate: &timestamp,
								Cri: &[]ske.CRI{
									{
										Name: skeutils.Ptr(ske.CRINAME_CONTAINERD),
									},
								},
							},
							{
								Version:        skeutils.Ptr("4230.2.3"),
								State:          skeutils.Ptr("supported"),
								ExpirationDate: nil, // no expiration
								Cri: &[]ske.CRI{
									{
										Name: skeutils.Ptr(ske.CRINAME_CONTAINERD),
									},
								},
							},
							{
								Version:        skeutils.Ptr("4459.2.1"),
								State:          skeutils.Ptr("preview"),
								ExpirationDate: nil,
								Cri: &[]ske.CRI{
									{
										Name: skeutils.Ptr(ske.CRINAME_CONTAINERD),
									},
								},
							},
						},
					},
					{
						Name: skeutils.Ptr("ubuntu"),
						Versions: &[]ske.MachineImageVersion{
							{
								Version:        skeutils.Ptr("2204.20250728.0"),
								State:          skeutils.Ptr("supported"),
								ExpirationDate: nil,
								// empty CRI slice
								Cri: &[]ske.CRI{},
							},
						},
					},
				},
			},
			expected: &Model{
				MachineImages: types.ListValueMust(
					types.ObjectType{AttrTypes: machineImageType},
					[]attr.Value{
						types.ObjectValueMust(machineImageType, map[string]attr.Value{
							"name": types.StringValue("flatcar"),
							"versions": types.ListValueMust(
								types.ObjectType{AttrTypes: machineImageVersionType},
								[]attr.Value{
									types.ObjectValueMust(machineImageVersionType, map[string]attr.Value{
										"version":         types.StringValue("4230.2.1"),
										"state":           types.StringValue("deprecated"),
										"expiration_date": types.StringValue(expDate),
										"cri": types.ListValueMust(
											types.StringType,
											[]attr.Value{
												types.StringValue(string(ske.CRINAME_CONTAINERD)),
											},
										),
									}),
									types.ObjectValueMust(machineImageVersionType, map[string]attr.Value{
										"version":         types.StringValue("4230.2.3"),
										"state":           types.StringValue("supported"),
										"expiration_date": types.StringNull(),
										"cri": types.ListValueMust(
											types.StringType,
											[]attr.Value{
												types.StringValue(string(ske.CRINAME_CONTAINERD)),
											},
										),
									}),
									types.ObjectValueMust(machineImageVersionType, map[string]attr.Value{
										"version":         types.StringValue("4459.2.1"),
										"state":           types.StringValue("preview"),
										"expiration_date": types.StringNull(),
										"cri": types.ListValueMust(
											types.StringType,
											[]attr.Value{
												types.StringValue(string(ske.CRINAME_CONTAINERD)),
											},
										),
									}),
								},
							),
						}),
						types.ObjectValueMust(machineImageType, map[string]attr.Value{
							"name": types.StringValue("ubuntu"),
							"versions": types.ListValueMust(
								types.ObjectType{AttrTypes: machineImageVersionType},
								[]attr.Value{
									types.ObjectValueMust(machineImageVersionType, map[string]attr.Value{
										"version":         types.StringValue("2204.20250728.0"),
										"state":           types.StringValue("supported"),
										"expiration_date": types.StringNull(),
										// empty CRI slice => empty list
										"cri": types.ListValueMust(
											types.StringType,
											[]attr.Value{},
										),
									}),
								},
							),
						}),
					},
				),
			},
			isValid: true,
		},
		{
			name: "nil machine images slice",
			input: &ske.ProviderOptions{
				MachineImages: nil,
			},
			expected: &Model{
				// Expect an empty list, not null
				MachineImages: types.ListValueMust(
					types.ObjectType{AttrTypes: machineImageType},
					[]attr.Value{},
				),
			},
			isValid: true,
		},
		{
			name: "empty machine images slice",
			input: &ske.ProviderOptions{
				MachineImages: &[]ske.MachineImage{},
			},
			expected: &Model{
				MachineImages: types.ListValueMust(
					types.ObjectType{AttrTypes: machineImageType},
					[]attr.Value{},
				),
			},
			isValid: true,
		},
		{
			name: "version without cri and without expiration",
			input: &ske.ProviderOptions{
				MachineImages: &[]ske.MachineImage{
					{
						Name: skeutils.Ptr("ubuntu"),
						Versions: &[]ske.MachineImageVersion{
							{
								Version:        skeutils.Ptr("2204.20250728.0"),
								State:          skeutils.Ptr("supported"),
								ExpirationDate: nil,
								Cri:            nil, // explicit nil => empty list
							},
						},
					},
				},
			},
			expected: &Model{
				MachineImages: types.ListValueMust(
					types.ObjectType{AttrTypes: machineImageType},
					[]attr.Value{
						types.ObjectValueMust(machineImageType, map[string]attr.Value{
							"name": types.StringValue("ubuntu"),
							"versions": types.ListValueMust(
								types.ObjectType{AttrTypes: machineImageVersionType},
								[]attr.Value{
									types.ObjectValueMust(machineImageVersionType, map[string]attr.Value{
										"version":         types.StringValue("2204.20250728.0"),
										"state":           types.StringValue("supported"),
										"expiration_date": types.StringNull(),
										"cri": types.ListValueMust(
											types.StringType,
											[]attr.Value{},
										),
									}),
								},
							),
						}),
					},
				),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &Model{}
			err := mapFields(context.Background(), tt.input, model)

			if tt.isValid && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.isValid && err == nil {
				t.Fatal("expected error but got none")
			}

			if tt.isValid {
				if diff := cmp.Diff(tt.expected, model); diff != "" {
					t.Fatalf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
