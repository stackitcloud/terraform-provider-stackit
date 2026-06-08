package dremio

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	dremioSdk "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1alphaapi"
)

func TestMapFields(t *testing.T) {
	instanceId := uuid.New().String()
	userId := uuid.New().String()
	tests := []struct {
		description string
		state       *Model
		input       *dremioSdk.DremioUserResponse
		expected    *Model
		wantErr     bool
	}{
		{
			"all_fields_filled",
			&Model{
				ProjectId:  types.StringValue("pid"),
				Region:     types.StringValue("rid"),
				InstanceId: types.StringValue(instanceId),
			},
			&dremioSdk.DremioUserResponse{
				Id:           userId,
				Description:  utils.Ptr("test description"),
				Email:        "test-user@example.com",
				FirstName:    "Test",
				LastName:     "User",
				Name:         "testUser",
				State:        "active",
				ErrorMessage: utils.Ptr("test error message"),
			},
			&Model{
				Id:           types.StringValue(fmt.Sprintf("pid,rid,%s,%s", instanceId, userId)),
				ProjectId:    types.StringValue("pid"),
				Region:       types.StringValue("rid"),
				InstanceId:   types.StringValue(instanceId),
				UserId:       types.StringValue(userId),
				Description:  types.StringPointerValue(utils.Ptr("test description")),
				Email:        types.StringValue("test-user@example.com"),
				FirstName:    types.StringValue("Test"),
				LastName:     types.StringValue("User"),
				Name:         types.StringValue("testUser"),
				State:        types.StringValue("active"),
				ErrorMessage: types.StringPointerValue(utils.Ptr("test error message")),
			},
			false,
		},
		{
			"nil response",
			&Model{
				Region:     types.StringValue("rid"),
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue(instanceId),
			},
			nil,
			&Model{
				Id:         types.StringValue(fmt.Sprintf("pid,rid,%s,", instanceId)),
				ProjectId:  types.StringValue("pid"),
				Region:     types.StringValue("rid"),
				InstanceId: types.StringValue(instanceId),
			},
			true,
		},
		{
			"nil state",
			nil,
			&dremioSdk.DremioUserResponse{},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(tt.input, tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapFields error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.expected, tt.state); diff != "" {
					t.Errorf("mapping mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	instanceId := uuid.New().String()
	tests := []struct {
		description string
		state       *UserModel
		expected    *dremioSdk.CreateDremioUserPayload
		wantErr     bool
	}{
		{
			"success",
			&UserModel{
				Model: Model{
					ProjectId:  types.StringValue("pid"),
					Region:     types.StringValue("rid"),
					InstanceId: types.StringValue(instanceId),

					Email:       types.StringValue("example@stackit.cloud"),
					Description: types.StringValue("test description"),
					FirstName:   types.StringValue("Test"),
					LastName:    types.StringValue("User"),
					Name:        types.StringValue("testUser"),
				},
				Password: types.StringValue("test-password"),
			},
			&dremioSdk.CreateDremioUserPayload{
				Email:       "example@stackit.cloud",
				Description: utils.Ptr("test description"),
				FirstName:   "Test",
				LastName:    "User",
				Name:        "testUser",
				Password:    "test-password",
			},
			false,
		},
		{
			"nil model",
			nil,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toCreatePayload(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("toCreatePayload error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.expected, payload); diff != "" {
					t.Errorf("toCreatePayload mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
