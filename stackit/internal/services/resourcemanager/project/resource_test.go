package project

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description    string
		input          *resourcemanager.ProjectResponseWithParents
		expected       Model
		expectedLabels *map[string]string
		isValid        bool
	}{
		{
			"default_ok",
			&resourcemanager.ProjectResponseWithParents{
				ContainerId: utils.Ptr("cid"),
			},
			Model{
				Id:                types.StringValue("cid"),
				ContainerId:       types.StringValue("cid"),
				ContainerParentId: types.StringNull(),
				Name:              types.StringNull(),
			},
			nil,
			true,
		},
		{
			"values_ok",
			&resourcemanager.ProjectResponseWithParents{
				ContainerId: utils.Ptr("cid"),
				Labels: &map[string]string{
					"label1": "ref1",
					"label2": "ref2",
				},
				Parent: &resourcemanager.Parent{
					ContainerId: utils.Ptr("pid"),
				},
				Name: utils.Ptr("name"),
			},
			Model{
				Id:                types.StringValue("cid"),
				ContainerId:       types.StringValue("cid"),
				ContainerParentId: types.StringValue("pid"),
				Name:              types.StringValue("name"),
			},
			&map[string]string{
				"label1": "ref1",
				"label2": "ref2",
			},
			true,
		},
		{
			"response_nil_fail",
			nil,
			Model{},
			nil,
			false,
		},
		{
			"no_resource_id",
			&resourcemanager.ProjectResponseWithParents{},
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
			state := &Model{
				ContainerId: tt.expected.ContainerId,
			}

			err := mapFields(context.Background(), tt.input, state)
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
			"default_ok",
			&Model{},
			nil,
			&resourcemanager.CreateProjectPayload{
				ContainerParentId: nil,
				Labels:            nil,
				Members: &[]resourcemanager.ProjectMember{
					{
						Role:    utils.Ptr(projectOwner),
						Subject: utils.Ptr("service_account_email"),
					},
				},
				Name: nil,
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
			&resourcemanager.CreateProjectPayload{
				ContainerParentId: utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "1",
					"label2": "2",
				},
				Members: &[]resourcemanager.ProjectMember{
					{
						Role:    utils.Ptr(projectOwner),
						Subject: utils.Ptr("service_account_email"),
					},
					{
						Role:    utils.Ptr(projectOwner),
						Subject: utils.Ptr("owner_email"),
					},
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
			output, err := toCreatePayload(tt.input, "service_account_email")
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
		expected    *resourcemanager.UpdateProjectPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{},
			nil,
			&resourcemanager.UpdateProjectPayload{
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
			&resourcemanager.UpdateProjectPayload{
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
