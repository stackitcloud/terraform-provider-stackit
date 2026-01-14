package instance

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/edge"
)

func TestMapFields(t *testing.T) {
	testTime, _ := time.Parse(time.RFC3339, "2023-09-04T10:00:00Z")
	uuidString := uuid.NewString()
	tests := []struct {
		description string
		input       *edge.Instance
		model       *Model
		expected    Model
		isValid     bool
	}{
		{
			"all_parameter_set",
			&edge.Instance{
				Id:          utils.Ptr("iid-123"),
				Created:     &testTime,
				DisplayName: utils.Ptr("test-instance"),
				Description: utils.Ptr("Test description"),
				PlanId:      utils.Ptr(uuidString),
				Status:      utils.Ptr(edge.InstanceStatus("CREATING")),
				FrontendUrl: utils.Ptr("https://iid-123.example.com"),
			},
			&Model{
				ProjectId: types.StringValue(uuidString),
				Region:    types.StringValue("eu01"),
			},
			Model{
				Id:          types.StringValue(fmt.Sprintf("%s,eu01,iid-123", uuidString)),
				ProjectId:   types.StringValue(uuidString),
				Region:      types.StringValue("eu01"),
				InstanceId:  types.StringValue("iid-123"),
				Created:     types.StringValue("2023-09-04 10:00:00 +0000 UTC"),
				DisplayName: types.StringValue("test-instance"),
				Description: types.StringValue("Test description"),
				PlanID:      types.StringValue(uuidString),
				Status:      types.StringValue("CREATING"),
				FrontendUrl: types.StringValue("https://iid-123.example.com"),
			},
			true,
		},
		{
			"empty_description",
			&edge.Instance{
				Id:          utils.Ptr("iid-123"),
				Created:     &testTime,
				DisplayName: utils.Ptr("test-instance"),
				Description: utils.Ptr(""),
				PlanId:      utils.Ptr(uuidString),
				Status:      utils.Ptr(edge.InstanceStatus("ACTIVE")),
				FrontendUrl: utils.Ptr("https://iid-123.example.com"),
			},
			&Model{
				ProjectId: types.StringValue(uuidString),
				Region:    types.StringValue("eu01"),
			},
			Model{
				Id:          types.StringValue(fmt.Sprintf("%s,eu01,iid-123", uuidString)),
				ProjectId:   types.StringValue(uuidString),
				Region:      types.StringValue("eu01"),
				InstanceId:  types.StringValue("iid-123"),
				Created:     types.StringValue("2023-09-04 10:00:00 +0000 UTC"),
				DisplayName: types.StringValue("test-instance"),
				Description: types.StringValue(""),
				PlanID:      types.StringValue(uuidString),
				Status:      types.StringValue("ACTIVE"),
				FrontendUrl: types.StringValue("https://iid-123.example.com"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			&Model{},
			Model{},
			false,
		},
		{
			"nil_model",
			&edge.Instance{},
			nil,
			Model{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(tt.input, tt.model)
			if !tt.isValid {
				if err == nil {
					t.Fatalf("Should have failed")
				}
				return
			}
			if err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			diff := cmp.Diff(tt.model, &tt.expected)
			if diff != "" {
				t.Errorf("Data does not match: %s", diff)
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	uuidString := uuid.NewString()
	tests := []struct {
		description string
		input       *Model
		expected    edge.CreateInstancePayload
		isValid     bool
	}{
		{
			"all_parameter_set",
			&Model{
				DisplayName: types.StringValue("new-instance"),
				Description: types.StringValue("A new test instance"),
				PlanID:      types.StringValue(uuidString),
			},
			edge.CreateInstancePayload{
				DisplayName: utils.Ptr("new-instance"),
				Description: utils.Ptr("A new test instance"),
				PlanId:      utils.Ptr(uuidString),
			},
			true,
		},
		{
			"no_description",
			&Model{
				DisplayName: types.StringValue("new-instance"),
				Description: types.StringNull(),
				PlanID:      types.StringValue(uuidString),
			},
			edge.CreateInstancePayload{
				DisplayName: utils.Ptr("new-instance"),
				Description: nil,
				PlanId:      utils.Ptr(uuidString),
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload := toCreatePayload(tt.input)
			diff := cmp.Diff(payload, tt.expected)
			if diff != "" {
				t.Errorf("Payload does not match: %s", diff)
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	var uuidOne = uuid.NewString()

	tests := []struct {
		description string
		input       *Model
		expected    edge.UpdateInstancePayload
		isValid     bool
	}{
		{
			"all_updatable_parameter_set",
			&Model{
				Description: types.StringValue("Updated description"),
				PlanID:      types.StringValue(uuidOne),
			},
			edge.UpdateInstancePayload{
				Description: utils.Ptr("Updated description"),
				PlanId:      utils.Ptr(uuidOne),
			},
			true,
		},
		{
			"description_null_plan_updated",
			&Model{
				Description: types.StringNull(),
				PlanID:      types.StringValue(uuidOne),
			},
			edge.UpdateInstancePayload{
				Description: nil,
				PlanId:      utils.Ptr(uuidOne),
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload := toUpdatePayload(tt.input)
			diff := cmp.Diff(payload, tt.expected)
			if diff != "" {
				t.Errorf("Payload does not match: %s", diff)
			}
		})
	}
}
