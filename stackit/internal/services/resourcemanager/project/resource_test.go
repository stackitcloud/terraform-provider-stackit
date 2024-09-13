package project

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
)

func TestMapProjectFields(t *testing.T) {
	testUUID := uuid.New().String()
	tests := []struct {
		description           string
		uuidContainerParentId bool
		projectResp           *resourcemanager.GetProjectResponse
		expected              Model
		expectedLabels        *map[string]string
		isValid               bool
	}{
		{
			"default_ok",
			false,
			&resourcemanager.GetProjectResponse{
				ContainerId: utils.Ptr("cid"),
				ProjectId:   utils.Ptr("pid"),
			},
			Model{
				Id:                types.StringValue("cid"),
				ContainerId:       types.StringValue("cid"),
				ProjectId:         types.StringValue("pid"),
				ContainerParentId: types.StringNull(),
				Name:              types.StringNull(),
				Members:           types.ListNull(types.ObjectType{AttrTypes: memberTypes}),
			},
			nil,
			true,
		},
		{
			"container_parent_id_ok",
			false,
			&resourcemanager.GetProjectResponse{
				ContainerId: utils.Ptr("cid"),
				ProjectId:   utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "ref1",
					"label2": "ref2",
				},
				Parent: &resourcemanager.Parent{
					ContainerId: utils.Ptr("parent_cid"),
					Id:          utils.Ptr("parent_pid"),
				},
				Name: utils.Ptr("name"),
			},
			Model{
				Id:                types.StringValue("cid"),
				ContainerId:       types.StringValue("cid"),
				ProjectId:         types.StringValue("pid"),
				ContainerParentId: types.StringValue("parent_cid"),
				Name:              types.StringValue("name"),
				Members:           types.ListNull(types.ObjectType{AttrTypes: memberTypes}),
			},
			&map[string]string{
				"label1": "ref1",
				"label2": "ref2",
			},
			true,
		},
		{
			"uuid_parent_id_ok",
			true,
			&resourcemanager.GetProjectResponse{
				ContainerId: utils.Ptr("cid"),
				ProjectId:   utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "ref1",
					"label2": "ref2",
				},
				Parent: &resourcemanager.Parent{
					ContainerId: utils.Ptr("parent_cid"),
					Id:          utils.Ptr(testUUID),
				},
				Name: utils.Ptr("name"),
			},
			Model{
				Id:                types.StringValue("cid"),
				ContainerId:       types.StringValue("cid"),
				ProjectId:         types.StringValue("pid"),
				ContainerParentId: types.StringValue(testUUID),
				Name:              types.StringValue("name"),
				Members:           types.ListNull(types.ObjectType{AttrTypes: memberTypes}),
			},
			&map[string]string{
				"label1": "ref1",
				"label2": "ref2",
			},
			true,
		},
		{
			"response_nil_fail",
			false,
			nil,
			Model{},
			nil,
			false,
		},
		{
			"no_resource_id",
			false,
			&resourcemanager.GetProjectResponse{},
			Model{},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if tt.expectedLabels == nil {
				tt.expected.Labels = types.MapNull(types.StringType)
			} else {
				convertedLabels, err := conversion.ToTerraformStringMap(context.Background(), *tt.expectedLabels)
				if err != nil {
					t.Fatalf("Error converting to terraform string map: %v", err)
				}
				tt.expected.Labels = convertedLabels
			}
			var containerParentId = types.StringNull()
			if tt.uuidContainerParentId {
				containerParentId = types.StringValue(testUUID)
			}
			model := &Model{
				ContainerId:       tt.expected.ContainerId,
				ContainerParentId: containerParentId,
				Members:           types.ListNull(types.ObjectType{AttrTypes: memberTypes}),
			}

			err := mapProjectFields(context.Background(), tt.projectResp, model, nil)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(model, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapMembersFields(t *testing.T) {
	tests := []struct {
		description    string
		configMembers  basetypes.ListValue
		membersResp    *[]authorization.Member
		expected       Model
		expectedLabels *map[string]string
		isValid        bool
	}{
		{
			"default_ok",
			types.ListNull(types.ObjectType{AttrTypes: memberTypes}),
			&[]authorization.Member{
				{
					Subject: utils.Ptr("owner_email"),
					Role:    utils.Ptr("owner"),
				},
				{
					Subject: utils.Ptr("reader_email"),
					Role:    utils.Ptr("reader"),
				},
			},
			Model{
				Id:                types.StringNull(),
				ProjectId:         types.StringNull(),
				ContainerId:       types.StringNull(),
				ContainerParentId: types.StringNull(),
				Name:              types.StringNull(),
				Labels:            types.MapNull(types.StringType),
				Members: types.ListValueMust(types.ObjectType{AttrTypes: memberTypes}, []attr.Value{
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("owner_email"),
							"role":    types.StringValue("owner"),
						},
					),
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("reader_email"),
							"role":    types.StringValue("reader"),
						},
					),
				}),
			},
			nil,
			true,
		},
		{
			"default_ok (preserve model order)",
			types.ListValueMust(types.ObjectType{AttrTypes: memberTypes}, []attr.Value{
				types.ObjectValueMust(
					memberTypes,
					map[string]attr.Value{
						"subject": types.StringValue("reader_email"),
						"role":    types.StringValue("reader"),
					},
				),
				types.ObjectValueMust(
					memberTypes,
					map[string]attr.Value{
						"subject": types.StringValue("owner_email"),
						"role":    types.StringValue("owner"),
					},
				),
			}),
			&[]authorization.Member{
				{
					Subject: utils.Ptr("owner_email"),
					Role:    utils.Ptr("owner"),
				},
				{
					Subject: utils.Ptr("reader_email"),
					Role:    utils.Ptr("reader"),
				},
			},
			Model{
				Id:                types.StringNull(),
				ProjectId:         types.StringNull(),
				ContainerId:       types.StringNull(),
				ContainerParentId: types.StringNull(),
				Name:              types.StringNull(),
				Labels:            types.MapNull(types.StringType),
				Members: types.ListValueMust(types.ObjectType{AttrTypes: memberTypes}, []attr.Value{
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("reader_email"),
							"role":    types.StringValue("reader"),
						},
					),
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("owner_email"),
							"role":    types.StringValue("owner"),
						},
					),
				}),
			},
			nil,
			true,
		},
		{
			"empty members",
			types.ListNull(types.ObjectType{AttrTypes: memberTypes}),
			&[]authorization.Member{},
			Model{
				Id:                types.StringNull(),
				ProjectId:         types.StringNull(),
				ContainerId:       types.StringNull(),
				ContainerParentId: types.StringNull(),
				Name:              types.StringNull(),
				Labels:            types.MapNull(types.StringType),
				Members:           types.ListValueMust(types.ObjectType{AttrTypes: memberTypes}, []attr.Value{}),
			},
			nil,
			true,
		},
		{
			"nil members",
			types.ListNull(types.ObjectType{AttrTypes: memberTypes}),
			nil,
			Model{
				Id:                types.StringNull(),
				ProjectId:         types.StringNull(),
				ContainerId:       types.StringNull(),
				ContainerParentId: types.StringNull(),
				Name:              types.StringNull(),
				Members:           types.ListNull(types.ObjectType{AttrTypes: memberTypes}),
				Labels:            types.MapNull(types.StringType),
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				Id:                types.StringNull(),
				ProjectId:         types.StringNull(),
				ContainerId:       types.StringNull(),
				ContainerParentId: types.StringNull(),
				Name:              types.StringNull(),
				Labels:            types.MapNull(types.StringType),
			}
			if !tt.configMembers.IsNull() {
				state.Members = tt.configMembers
			}
			err := mapMembersFields(context.Background(), tt.membersResp, state)
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
		inputLabels *map[string]string
		expected    *resourcemanager.CreateProjectPayload
		isValid     bool
	}{
		{
			"mapping_with_conversions_single_member",
			&Model{
				ContainerParentId: types.StringValue("pid"),
				Name:              types.StringValue("name"),
				Members: types.ListValueMust(types.ObjectType{AttrTypes: memberTypes}, []attr.Value{
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("owner_email"),
							"role":    types.StringValue("owner"),
						},
					),
				}),
			},
			&map[string]string{
				"label1": "1",
				"label2": "2",
			},
			&resourcemanager.CreateProjectPayload{
				ContainerParentId: utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "1",
					"label2": "2",
				},
				Members: &[]resourcemanager.Member{
					{
						Subject: utils.Ptr("owner_email"),
						Role:    utils.Ptr("owner"),
					},
				},
				Name: utils.Ptr("name"),
			},
			true,
		},
		{
			"mapping_with_conversions_ok_multiple_members",
			&Model{
				ContainerParentId: types.StringValue("pid"),
				Name:              types.StringValue("name"),
				Members: types.ListValueMust(types.ObjectType{AttrTypes: memberTypes}, []attr.Value{
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("owner_email"),
							"role":    types.StringValue("owner"),
						},
					),
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("reader_email"),
							"role":    types.StringValue("reader"),
						},
					),
				}),
			},
			&map[string]string{
				"label1": "1",
				"label2": "2",
			},
			&resourcemanager.CreateProjectPayload{
				ContainerParentId: utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "1",
					"label2": "2",
				},
				Members: &[]resourcemanager.Member{
					{
						Subject: utils.Ptr("owner_email"),
						Role:    utils.Ptr("owner"),
					},
					{
						Subject: utils.Ptr("reader_email"),
						Role:    utils.Ptr("reader"),
					},
				},
				Name: utils.Ptr("name"),
			},
			true,
		},
		{
			"new members field takes precedence over deprecated owner_email field",
			&Model{
				ContainerParentId: types.StringValue("pid"),
				Name:              types.StringValue("name"),
				OwnerEmail:        types.StringValue("some_email_deprecated"),
				Members: types.ListValueMust(types.ObjectType{AttrTypes: memberTypes}, []attr.Value{
					types.ObjectValueMust(
						memberTypes,
						map[string]attr.Value{
							"subject": types.StringValue("owner_email"),
							"role":    types.StringValue("owner"),
						},
					),
				}),
			},
			&map[string]string{
				"label1": "1",
				"label2": "2",
			},
			&resourcemanager.CreateProjectPayload{
				ContainerParentId: utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "1",
					"label2": "2",
				},
				Members: &[]resourcemanager.Member{
					{
						Subject: utils.Ptr("owner_email"),
						Role:    utils.Ptr("owner"),
					},
				},
				Name: utils.Ptr("name"),
			},
			true,
		},
		{
			"deprecated owner_email field still works",
			&Model{
				ContainerParentId: types.StringValue("pid"),
				Name:              types.StringValue("name"),
				OwnerEmail:        types.StringValue("some_email_deprecated"),
			},
			&map[string]string{
				"label1": "1",
				"label2": "2",
			},
			&resourcemanager.CreateProjectPayload{
				ContainerParentId: utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "1",
					"label2": "2",
				},
				Members: &[]resourcemanager.Member{
					{
						Subject: utils.Ptr("some_email_deprecated"),
						Role:    utils.Ptr("owner"),
					},
				},
				Name: utils.Ptr("name"),
			},
			true,
		},
		{
			"no members or owner_email fails",
			&Model{
				ContainerParentId: types.StringValue("pid"),
				Name:              types.StringValue("name"),
			},
			&map[string]string{},
			nil,
			false,
		},
		{
			"nil_model",
			nil,
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if tt.input != nil {
				if tt.inputLabels == nil {
					tt.input.Labels = types.MapNull(types.StringType)
				} else {
					convertedLabels, err := conversion.ToTerraformStringMap(context.Background(), *tt.inputLabels)
					if err != nil {
						t.Fatalf("Error converting to terraform string map: %v", err)
					}
					tt.input.Labels = convertedLabels
				}
			}
			output, err := toCreatePayload(context.Background(), tt.input)
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

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		inputLabels *map[string]string
		expected    *resourcemanager.PartialUpdateProjectPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{},
			nil,
			&resourcemanager.PartialUpdateProjectPayload{
				ContainerParentId: nil,
				Labels:            nil,
				Name:              nil,
			},
			true,
		},
		{
			"mapping_with_conversions_ok",
			&Model{
				ContainerParentId: types.StringValue("pid"),
				Name:              types.StringValue("name"),
				OwnerEmail:        types.StringValue("owner_email"),
			},
			&map[string]string{
				"label1": "1",
				"label2": "2",
			},
			&resourcemanager.PartialUpdateProjectPayload{
				ContainerParentId: utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "1",
					"label2": "2",
				},
				Name: utils.Ptr("name"),
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if tt.input != nil {
				if tt.inputLabels == nil {
					tt.input.Labels = types.MapNull(types.StringType)
				} else {
					convertedLabels, err := conversion.ToTerraformStringMap(context.Background(), *tt.inputLabels)
					if err != nil {
						t.Fatalf("Error converting to terraform string map: %v", err)
					}
					tt.input.Labels = convertedLabels
				}
			}
			output, err := toUpdatePayload(tt.input)
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
