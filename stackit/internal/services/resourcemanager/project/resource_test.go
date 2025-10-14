package project

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
)

func TestMapProjectFields(t *testing.T) {
	testUUID := uuid.New().String()
	baseTime := time.Now()
	createTime := baseTime
	updateTime := baseTime.Add(1 * time.Hour)

	tests := []struct {
		description           string
		uuidContainerParentId bool
		projectResp           *resourcemanager.GetProjectResponse
		expected              Model
		expectedLabels        *map[string]string
		isValid               bool
	}{
		{
			description:           "default_ok",
			uuidContainerParentId: false,
			projectResp: &resourcemanager.GetProjectResponse{
				ContainerId:  utils.Ptr("cid"),
				ProjectId:    utils.Ptr("pid"),
				CreationTime: &createTime,
				UpdateTime:   &updateTime,
			},
			expected: Model{
				Id:                types.StringValue("cid"),
				ContainerId:       types.StringValue("cid"),
				ProjectId:         types.StringValue("pid"),
				ContainerParentId: types.StringNull(),
				Name:              types.StringNull(),
				CreationTime:      types.StringValue(createTime.Format(time.RFC3339)),
				UpdateTime:        types.StringValue(updateTime.Format(time.RFC3339)),
			},
			expectedLabels: nil,
			isValid:        true,
		},
		{
			description:           "container_parent_id_ok",
			uuidContainerParentId: false,
			projectResp: &resourcemanager.GetProjectResponse{
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
				Name:         utils.Ptr("name"),
				CreationTime: &createTime,
				UpdateTime:   &updateTime,
			},
			expected: Model{
				Id:                types.StringValue("cid"),
				ContainerId:       types.StringValue("cid"),
				ProjectId:         types.StringValue("pid"),
				ContainerParentId: types.StringValue("parent_cid"),
				Name:              types.StringValue("name"),
				CreationTime:      types.StringValue(createTime.Format(time.RFC3339)),
				UpdateTime:        types.StringValue(updateTime.Format(time.RFC3339)),
			},
			expectedLabels: &map[string]string{
				"label1": "ref1",
				"label2": "ref2",
			},
			isValid: true,
		},
		{
			description:           "uuid_parent_id_ok",
			uuidContainerParentId: true,
			projectResp: &resourcemanager.GetProjectResponse{
				ContainerId: utils.Ptr("cid"),
				ProjectId:   utils.Ptr("pid"),
				Labels: &map[string]string{
					"label1": "ref1",
					"label2": "ref2",
				},
				Parent: &resourcemanager.Parent{
					ContainerId: utils.Ptr("parent_cid"),
					Id:          utils.Ptr(testUUID), // simulate UUID logic
				},
				Name:         utils.Ptr("name"),
				CreationTime: &createTime,
				UpdateTime:   &updateTime,
			},
			expected: Model{
				Id:                types.StringValue("cid"),
				ContainerId:       types.StringValue("cid"),
				ProjectId:         types.StringValue("pid"),
				ContainerParentId: types.StringValue(testUUID),
				Name:              types.StringValue("name"),
				CreationTime:      types.StringValue(createTime.Format(time.RFC3339)),
				UpdateTime:        types.StringValue(updateTime.Format(time.RFC3339)),
			},
			expectedLabels: &map[string]string{
				"label1": "ref1",
				"label2": "ref2",
			},
			isValid: true,
		},
		{
			description:           "response_nil_fail",
			uuidContainerParentId: false,
			projectResp:           nil,
			expected:              Model{},
			expectedLabels:        nil,
			isValid:               false,
		},
		{
			description:           "no_resource_id",
			uuidContainerParentId: false,
			projectResp:           &resourcemanager.GetProjectResponse{},
			expected:              Model{},
			expectedLabels:        nil,
			isValid:               false,
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
			} else if tt.projectResp != nil && tt.projectResp.Parent != nil && tt.projectResp.Parent.ContainerId != nil {
				containerParentId = types.StringValue(*tt.projectResp.Parent.ContainerId)
			}

			model := &Model{
				ContainerId:       tt.expected.ContainerId,
				ContainerParentId: containerParentId,
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

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *ResourceModel
		inputLabels *map[string]string
		expected    *resourcemanager.CreateProjectPayload
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
			&resourcemanager.CreateProjectPayload{
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
		expected    *resourcemanager.PartialUpdateProjectPayload
		isValid     bool
	}{
		{
			"default_ok",
			&ResourceModel{},
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
