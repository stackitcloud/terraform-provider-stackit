package folder

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
)

func TestMapFolderFields(t *testing.T) {
	testUUID := "73b2d741-bddd-471f-8d47-3d1aa677a19c"

	tests := []struct {
		description           string
		uuidContainerParentId bool
		respContainerId       *string
		respName              *string
		labels                *map[string]string
		parent                *resourcemanager.Parent
		expected              Model
		expectedLabels        *map[string]string
		isValid               bool
	}{
		{
			"valid input with UUID parent ID",
			true,
			utils.Ptr("folder-cid-uuid"),
			utils.Ptr("folder-name"),
			&map[string]string{
				"env": "prod",
			},
			&resourcemanager.Parent{
				Id: utils.Ptr(testUUID),
			},
			Model{
				Id:                types.StringValue("folder-cid-uuid"),
				ContainerId:       types.StringValue("folder-cid-uuid"),
				ContainerParentId: types.StringValue(testUUID),
				Name:              types.StringValue("folder-name"),
			},
			&map[string]string{
				"env": "prod",
			},
			true,
		},
		{
			"valid input with UUID parent ID no labels",
			true,
			utils.Ptr("folder-cid-uuid"),
			utils.Ptr("folder-name"),
			nil,
			&resourcemanager.Parent{
				Id: utils.Ptr(testUUID),
			},
			Model{
				Id:                types.StringValue("folder-cid-uuid"),
				ContainerId:       types.StringValue("folder-cid-uuid"),
				ContainerParentId: types.StringValue(testUUID),
				Name:              types.StringValue("folder-name"),
			},
			nil,
			true,
		},
		{
			"valid input with ContainerId as parent",
			false,
			utils.Ptr("folder-cid"),
			utils.Ptr("folder-name"),
			&map[string]string{
				"env": "dev",
			},
			&resourcemanager.Parent{
				ContainerId: utils.Ptr("parent-container-id"),
			},
			Model{
				Id:                types.StringValue("folder-cid"),
				ContainerId:       types.StringValue("folder-cid"),
				ContainerParentId: types.StringValue("parent-container-id"),
				Name:              types.StringValue("folder-name"),
			},
			&map[string]string{
				"env": "dev",
			},
			true,
		},
		{
			"valid input with ContainerId as parent no labels",
			false,
			utils.Ptr("folder-cid"),
			utils.Ptr("folder-name"),
			nil,
			&resourcemanager.Parent{
				ContainerId: utils.Ptr("parent-container-id"),
			},
			Model{
				Id:                types.StringValue("folder-cid"),
				ContainerId:       types.StringValue("folder-cid"),
				ContainerParentId: types.StringValue("parent-container-id"),
				Name:              types.StringValue("folder-name"),
			},
			nil,
			true,
		},
		{
			"nil labels",
			false,
			utils.Ptr("folder-cid"),
			utils.Ptr("folder-name"),
			nil,
			nil,
			Model{
				Id:                types.StringValue("folder-cid"),
				ContainerId:       types.StringValue("folder-cid"),
				ContainerParentId: types.StringNull(),
				Name:              types.StringValue("folder-name"),
			},
			nil,
			true,
		},
		{
			"nil container ID, should fail",
			false,
			nil,
			utils.Ptr("name"),
			nil,
			nil,
			Model{},
			nil,
			false,
		},
		{
			"empty container ID, should fail",
			false,
			utils.Ptr(""),
			utils.Ptr("name"),
			nil,
			nil,
			Model{},
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			// Handle expected label conversion
			if tt.expectedLabels == nil {
				tt.expected.Labels = types.MapNull(types.StringType)
			} else {
				convertedLabels, err := conversion.ToTerraformStringMap(context.Background(), *tt.expectedLabels)
				if err != nil {
					t.Fatalf("Error converting to terraform string map: %v", err)
				}
				tt.expected.Labels = convertedLabels
			}

			// Simulate ContainerParentId configuration based on UUID detection logic
			var containerParentId basetypes.StringValue
			if tt.uuidContainerParentId {
				containerParentId = types.StringValue(testUUID)
			} else if tt.parent != nil && tt.parent.ContainerId != nil {
				containerParentId = types.StringValue(*tt.parent.ContainerId)
			} else {
				containerParentId = types.StringNull()
			}

			model := &Model{
				ContainerId:       tt.expected.ContainerId,
				ContainerParentId: containerParentId,
			}

			err := mapFolderFields(context.Background(), tt.respContainerId, tt.respName, tt.labels, tt.parent, model, nil)

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

func TestMapFolderCreateFields(t *testing.T) {
	labels := map[string]string{
		"env": "prod",
	}
	resp := &resourcemanager.FolderResponse{
		ContainerId: utils.Ptr("folder-id"),
		Name:        utils.Ptr("my-folder"),
		Labels:      &labels,
		Parent: &resourcemanager.Parent{
			Id: utils.Ptr(uuid.New().String()),
		},
	}

	model := Model{
		ContainerParentId: types.StringValue(*resp.Parent.Id),
	}

	err := mapFolderCreateFields(context.Background(), resp, &model, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	cbLabels, _ := conversion.ToTerraformStringMap(context.Background(), labels)
	expected := Model{
		Id:                types.StringValue("folder-id"),
		ContainerId:       types.StringValue("folder-id"),
		ContainerParentId: types.StringValue(*resp.Parent.Id),
		Name:              types.StringValue("my-folder"),
		Labels:            cbLabels,
	}
	diff := cmp.Diff(model, expected)
	if diff != "" {
		t.Fatalf("mapFolderCreateFields() mismatch: %s", diff)
	}
}

func TestMapFolderDetailsFields(t *testing.T) {
	resp := &resourcemanager.GetFolderDetailsResponse{
		ContainerId: utils.Ptr("folder-id"),
		Name:        utils.Ptr("details-folder"),
		Labels: &map[string]string{
			"foo": "bar",
		},
		Parent: &resourcemanager.Parent{
			ContainerId: utils.Ptr("parent-container"),
		},
	}

	var model Model
	err := mapFolderDetailsFields(context.Background(), resp, &model, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	tfLabels, _ := conversion.ToTerraformStringMap(context.Background(), *resp.Labels)

	expected := Model{
		Id:                types.StringValue("folder-id"),
		ContainerId:       types.StringValue("folder-id"),
		ContainerParentId: types.StringValue("parent-container"),
		Name:              types.StringValue("details-folder"),
		Labels:            tfLabels,
	}

	diff := cmp.Diff(model, expected)
	if diff != "" {
		t.Fatalf("mapFolderDetailsFields() mismatch: %s", diff)
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *ResourceModel
		inputLabels *map[string]string
		expected    *resourcemanager.CreateFolderPayload
		isValid     bool
	}{
		{
			"mapping_with_conversions",
			&ResourceModel{
				Model: Model{
					ContainerParentId: types.StringValue("pid"),
					Name:              types.StringValue("name"),
				},
				OwnerEmail: types.StringValue("john.doe@stackit.cloud"),
			},
			&map[string]string{
				"label1": "1",
				"label2": "2",
			},
			&resourcemanager.CreateFolderPayload{
				ContainerParentId: utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "1",
					"label2": "2",
				},
				Members: &[]resourcemanager.Member{
					{
						Subject: utils.Ptr("john.doe@stackit.cloud"),
						Role:    utils.Ptr("owner"),
					},
				},
				Name: utils.Ptr("name"),
			},
			true,
		},
		{
			"no owner_email fails",
			&ResourceModel{
				Model: Model{
					ContainerParentId: types.StringValue("pid"),
					Name:              types.StringValue("name"),
				},
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

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *ResourceModel
		inputLabels *map[string]string
		expected    *resourcemanager.PartialUpdateFolderPayload
		isValid     bool
	}{
		{
			"default_ok",
			&ResourceModel{},
			nil,
			&resourcemanager.PartialUpdateFolderPayload{
				ContainerParentId: nil,
				Labels:            nil,
				Name:              nil,
			},
			true,
		},
		{
			"mapping_with_conversions_ok",
			&ResourceModel{
				Model: Model{
					ContainerParentId: types.StringValue("pid"),
					Name:              types.StringValue("name"),
				},
				OwnerEmail: types.StringValue("owner_email"),
			},
			&map[string]string{
				"label1": "1",
				"label2": "2",
			},
			&resourcemanager.PartialUpdateFolderPayload{
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

func TestToMembersPayload(t *testing.T) {
	type args struct {
		model *ResourceModel
	}
	tests := []struct {
		name    string
		args    args
		want    *[]resourcemanager.Member
		wantErr bool
	}{
		{
			name:    "missing model",
			args:    args{},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty model",
			args: args{
				model: &ResourceModel{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ok",
			args: args{
				model: &ResourceModel{
					OwnerEmail: types.StringValue("john.doe@stackit.cloud"),
				},
			},
			want: &[]resourcemanager.Member{
				{
					Subject: utils.Ptr("john.doe@stackit.cloud"),
					Role:    utils.Ptr("owner"),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toMembersPayload(tt.args.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toMembersPayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toMembersPayload() got = %v, want %v", got, tt.want)
			}
		})
	}
}
