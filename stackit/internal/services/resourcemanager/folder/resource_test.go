package folder

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
)

func TestMapFolderFields(t *testing.T) {
	parentContainerUUID := uuid.New().String()
	folderUUID := uuid.New().String()

	// Create base timestamps for reuse
	baseTime := time.Now()
	createTime := baseTime
	updateTime := baseTime.Add(1 * time.Hour)

	tests := []struct {
		description           string
		uuidContainerParentId bool
		respFolderId          *string
		respContainerId       *string
		respName              *string
		respCreateTime        *time.Time
		respUpdateTime        *time.Time
		labels                *map[string]string
		parent                *resourcemanager.Parent
		expected              Model
		expectedLabels        *map[string]string
		isValid               bool
	}{
		{
			description:           "valid input with UUID parent ID",
			uuidContainerParentId: true,
			respFolderId:          &folderUUID,
			respContainerId:       utils.Ptr("folder-human-readable-id"),
			respName:              utils.Ptr("folder-name"),
			respCreateTime:        &createTime,
			respUpdateTime:        &updateTime,
			labels: &map[string]string{
				"env": "prod",
			},
			parent: &resourcemanager.Parent{
				Id: utils.Ptr(parentContainerUUID),
			},
			expected: Model{
				Id:                types.StringValue("folder-human-readable-id"),
				FolderId:          types.StringValue(folderUUID),
				ContainerId:       types.StringValue("folder-human-readable-id"),
				ContainerParentId: types.StringValue(parentContainerUUID),
				Name:              types.StringValue("folder-name"),
				CreationTime:      types.StringValue(createTime.Format(time.RFC3339)),
				UpdateTime:        types.StringValue(updateTime.Format(time.RFC3339)),
			},
			expectedLabels: &map[string]string{
				"env": "prod",
			},
			isValid: true,
		},
		{
			description:           "valid input with UUID parent ID no labels",
			uuidContainerParentId: true,
			respFolderId:          &folderUUID,
			respContainerId:       utils.Ptr("folder-human-readable-id"),
			respName:              utils.Ptr("folder-name"),
			respCreateTime:        &createTime,
			respUpdateTime:        &updateTime,
			labels:                nil,
			parent: &resourcemanager.Parent{
				Id: utils.Ptr(parentContainerUUID),
			},
			expected: Model{
				Id:                types.StringValue("folder-human-readable-id"),
				FolderId:          types.StringValue(folderUUID),
				ContainerId:       types.StringValue("folder-human-readable-id"),
				ContainerParentId: types.StringValue(parentContainerUUID),
				Name:              types.StringValue("folder-name"),
				CreationTime:      types.StringValue(createTime.Format(time.RFC3339)),
				UpdateTime:        types.StringValue(updateTime.Format(time.RFC3339)),
			},
			expectedLabels: nil,
			isValid:        true,
		},
		{
			description:           "valid input with ContainerId as parent",
			uuidContainerParentId: false,
			respFolderId:          &folderUUID,
			respContainerId:       utils.Ptr("folder-human-readable-id"),
			respName:              utils.Ptr("folder-name"),
			respCreateTime:        &createTime,
			respUpdateTime:        &updateTime,
			labels: &map[string]string{
				"env": "dev",
			},
			parent: &resourcemanager.Parent{
				ContainerId: utils.Ptr("parent-container-id"),
			},
			expected: Model{
				Id:                types.StringValue("folder-human-readable-id"),
				FolderId:          types.StringValue(folderUUID),
				ContainerId:       types.StringValue("folder-human-readable-id"),
				ContainerParentId: types.StringValue("parent-container-id"),
				Name:              types.StringValue("folder-name"),
				CreationTime:      types.StringValue(createTime.Format(time.RFC3339)),
				UpdateTime:        types.StringValue(updateTime.Format(time.RFC3339)),
			},
			expectedLabels: &map[string]string{
				"env": "dev",
			},
			isValid: true,
		},
		{
			description:           "valid input with ContainerId as parent no labels",
			uuidContainerParentId: false,
			respFolderId:          &folderUUID,
			respContainerId:       utils.Ptr("folder-human-readable-id"),
			respName:              utils.Ptr("folder-name"),
			respCreateTime:        &createTime,
			respUpdateTime:        &updateTime,
			labels:                nil,
			parent: &resourcemanager.Parent{
				ContainerId: utils.Ptr("parent-container-id"),
			},
			expected: Model{
				Id:                types.StringValue("folder-human-readable-id"),
				FolderId:          types.StringValue(folderUUID),
				ContainerId:       types.StringValue("folder-human-readable-id"),
				ContainerParentId: types.StringValue("parent-container-id"),
				Name:              types.StringValue("folder-name"),
				CreationTime:      types.StringValue(createTime.Format(time.RFC3339)),
				UpdateTime:        types.StringValue(updateTime.Format(time.RFC3339)),
			},
			expectedLabels: nil,
			isValid:        true,
		},
		{
			description:           "nil labels",
			uuidContainerParentId: false,
			respFolderId:          &folderUUID,
			respContainerId:       utils.Ptr("folder-human-readable-id"),
			respName:              utils.Ptr("folder-name"),
			respCreateTime:        &createTime,
			respUpdateTime:        &updateTime,
			labels:                nil,
			parent:                nil,
			expected: Model{
				Id:                types.StringValue("folder-human-readable-id"),
				FolderId:          types.StringValue(folderUUID),
				ContainerId:       types.StringValue("folder-human-readable-id"),
				ContainerParentId: types.StringNull(),
				Name:              types.StringValue("folder-name"),
				CreationTime:      types.StringValue(createTime.Format(time.RFC3339)),
				UpdateTime:        types.StringValue(updateTime.Format(time.RFC3339)),
			},
			expectedLabels: nil,
			isValid:        true,
		},
		{
			description:           "nil container ID, should fail",
			uuidContainerParentId: false,
			respContainerId:       nil,
			respName:              utils.Ptr("name"),
			respCreateTime:        nil,
			respUpdateTime:        nil,
			labels:                nil,
			parent:                nil,
			expected:              Model{},
			expectedLabels:        nil,
			isValid:               false,
		},
		{
			description:           "empty container ID, should fail",
			uuidContainerParentId: false,
			respContainerId:       utils.Ptr(""),
			respName:              utils.Ptr("name"),
			respCreateTime:        nil,
			respUpdateTime:        nil,
			labels:                nil,
			parent:                nil,
			expected:              Model{},
			expectedLabels:        nil,
			isValid:               false,
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
				containerParentId = types.StringValue(parentContainerUUID)
			} else if tt.parent != nil && tt.parent.ContainerId != nil {
				containerParentId = types.StringValue(*tt.parent.ContainerId)
			} else {
				containerParentId = types.StringNull()
			}

			model := &Model{
				ContainerId:       tt.expected.ContainerId,
				ContainerParentId: containerParentId,
			}

			err := mapFolderFields(
				context.Background(),
				tt.respContainerId,
				tt.respName,
				tt.respFolderId,
				tt.labels,
				tt.parent,
				tt.respCreateTime,
				tt.respUpdateTime,
				model,
				nil,
			)

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
	baseTime := time.Now()
	createTime := baseTime
	updateTime := baseTime.Add(1 * time.Hour)

	resp := &resourcemanager.FolderResponse{
		FolderId:    utils.Ptr("folder-uuid"),
		ContainerId: utils.Ptr("folder-human-readable-id"),
		Name:        utils.Ptr("my-folder"),
		Labels:      &labels,
		Parent: &resourcemanager.Parent{
			Id: utils.Ptr(uuid.New().String()),
		},
		CreationTime: &createTime,
		UpdateTime:   &updateTime,
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
		Id:                types.StringValue("folder-human-readable-id"),
		FolderId:          types.StringValue("folder-uuid"),
		ContainerId:       types.StringValue("folder-human-readable-id"),
		ContainerParentId: types.StringValue(*resp.Parent.Id),
		Name:              types.StringValue("my-folder"),
		Labels:            cbLabels,
		CreationTime:      types.StringValue(createTime.Format(time.RFC3339)),
		UpdateTime:        types.StringValue(updateTime.Format(time.RFC3339)),
	}
	diff := cmp.Diff(model, expected)
	if diff != "" {
		t.Fatalf("mapFolderCreateFields() mismatch: %s", diff)
	}
}

func TestMapFolderDetailsFields(t *testing.T) {
	baseTime := time.Now()
	createTime := baseTime
	updateTime := baseTime.Add(1 * time.Hour)

	resp := &resourcemanager.GetFolderDetailsResponse{
		FolderId:    utils.Ptr("folder-uuid"),
		ContainerId: utils.Ptr("folder-human-readable-id"),
		Name:        utils.Ptr("details-folder"),
		Labels: &map[string]string{
			"foo": "bar",
		},
		Parent: &resourcemanager.Parent{
			ContainerId: utils.Ptr("parent-container"),
		},
		CreationTime: &createTime,
		UpdateTime:   &updateTime,
	}

	var model Model
	err := mapFolderDetailsFields(context.Background(), resp, &model, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	tfLabels, _ := conversion.ToTerraformStringMap(context.Background(), *resp.Labels)

	expected := Model{
		Id:                types.StringValue("folder-human-readable-id"),
		FolderId:          types.StringValue("folder-uuid"),
		ContainerId:       types.StringValue("folder-human-readable-id"),
		ContainerParentId: types.StringValue("parent-container"),
		Name:              types.StringValue("details-folder"),
		Labels:            tfLabels,
		CreationTime:      types.StringValue(createTime.Format(time.RFC3339)),
		UpdateTime:        types.StringValue(updateTime.Format(time.RFC3339)),
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
