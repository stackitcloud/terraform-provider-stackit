package instance

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
)

func TestMapInstanceFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description string
		state       *Model
		input       modelexperiments.Instance
		expected    Model
		isValid     bool
	}{
		{
			description: "should error when state is nil",
			state:       nil,
			input: modelexperiments.Instance{
				Id: "id",
			},
			expected: Model{},
			isValid:  false,
		},
		{
			description: "should error when instance id is not present",
			state:       &Model{},
			input:       modelexperiments.Instance{},
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "should map fields correctly",
			state: &Model{
				Id:         types.StringValue("pid,eu01,id"),
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("id"),
				Region:     types.StringValue("eu01"),
			},
			input: modelexperiments.Instance{
				Id:                         "id",
				BucketName:                 new("bucketName"),
				Description:                new("description"),
				DeletedExperimentRetention: new("30d"),
				ErrorMessage:               nil,
				Labels:                     &map[string]string{"key": "value"},
				State:                      "active",
				Url:                        "url",
				Name:                       "name",
			},
			expected: Model{
				Id:                         types.StringValue("pid,eu01,id"),
				ProjectId:                  types.StringValue("pid"),
				Region:                     types.StringValue("eu01"),
				InstanceId:                 types.StringValue("id"),
				Name:                       types.StringValue("name"),
				Description:                types.StringValue("description"),
				State:                      types.StringValue("active"),
				BucketName:                 types.StringValue("bucketName"),
				DeletedExperimentRetention: types.StringValue("30d"),
				Url:                        types.StringValue("url"),
				ErrorMessage:               types.StringNull(),
				Labels:                     types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := mapInstance(ctx, &tt.input, tt.state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}

			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}

			if tt.isValid {
				diff := cmp.Diff(tt.state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapCreateResponseFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description         string
		state               *Model
		inputCreateResponse *modelexperiments.CreateInstanceResponse
		inputGetResponse    *modelexperiments.GetInstanceResponse
		expected            Model
		isValid             bool
	}{
		{
			description:         "should error when instance create response is nil",
			state:               &Model{},
			inputCreateResponse: nil,
			inputGetResponse:    &modelexperiments.GetInstanceResponse{},
			expected:            Model{},
			isValid:             false,
		},
		{
			description:         "should error when state is nil",
			state:               nil,
			inputCreateResponse: &modelexperiments.CreateInstanceResponse{},
			inputGetResponse:    &modelexperiments.GetInstanceResponse{},
			expected:            Model{},
			isValid:             false,
		},
		{
			description: "should error when instance id is not present",
			state:       &Model{},
			inputCreateResponse: &modelexperiments.CreateInstanceResponse{
				Instance: modelexperiments.Instance{},
			},
			inputGetResponse: &modelexperiments.GetInstanceResponse{},
			expected:         Model{},
			isValid:          false,
		},
		{
			description: "should map fields correctly even if Get Response is nil",
			state: &Model{
				Id:         types.StringValue("pid,eu01,id"),
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("id"),
				Region:     types.StringValue("eu01"),
			},
			inputCreateResponse: &modelexperiments.CreateInstanceResponse{
				Instance: modelexperiments.Instance{
					Id:                         "id",
					Description:                new("description"),
					DeletedExperimentRetention: new("30d"),
					ErrorMessage:               nil,
					Labels:                     &map[string]string{"key": "value"},
					State:                      "pending",
					Url:                        "url",
					Name:                       "name",
				}},
			inputGetResponse: nil,
			expected: Model{
				Id:                         types.StringValue("pid,eu01,id"),
				ProjectId:                  types.StringValue("pid"),
				Region:                     types.StringValue("eu01"),
				InstanceId:                 types.StringValue("id"),
				Name:                       types.StringValue("name"),
				Description:                types.StringValue("description"),
				State:                      types.StringValue("unknown"),
				DeletedExperimentRetention: types.StringValue("30d"),
				Url:                        types.StringValue("url"),
				ErrorMessage:               types.StringNull(),
				Labels:                     types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
			},
			isValid: true,
		},
		{
			description: "should map fields correctly",
			state: &Model{
				Id:         types.StringValue("pid,eu01,id"),
				ProjectId:  types.StringValue("pid"),
				InstanceId: types.StringValue("id"),
				Region:     types.StringValue("eu01"),
			},
			inputCreateResponse: &modelexperiments.CreateInstanceResponse{
				Instance: modelexperiments.Instance{
					Id:                         "id",
					Description:                new("description"),
					DeletedExperimentRetention: new("30d"),
					ErrorMessage:               nil,
					Labels:                     &map[string]string{"key": "value"},
					State:                      "pending",
					Url:                        "url",
					Name:                       "name",
				}},
			inputGetResponse: &modelexperiments.GetInstanceResponse{
				Instance: modelexperiments.Instance{
					State:      "active",
					BucketName: new("bucketName"),
				},
			},
			expected: Model{
				Id:                         types.StringValue("pid,eu01,id"),
				ProjectId:                  types.StringValue("pid"),
				Region:                     types.StringValue("eu01"),
				InstanceId:                 types.StringValue("id"),
				Name:                       types.StringValue("name"),
				Description:                types.StringValue("description"),
				State:                      types.StringValue("active"),
				BucketName:                 types.StringValue("bucketName"),
				DeletedExperimentRetention: types.StringValue("30d"),
				Url:                        types.StringValue("url"),
				ErrorMessage:               types.StringNull(),
				Labels:                     types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := mapCreateResponse(ctx, tt.inputCreateResponse, tt.inputGetResponse, tt.state, "eu01")
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}

			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}

			if tt.isValid {
				diff := cmp.Diff(tt.state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description string
		input       *Model
		expected    *modelexperiments.CreateInstancePayload
		isValid     bool
	}{
		{
			description: "should error on nil input",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
		{
			description: "should error when map is not correct",
			input: &Model{
				Name:                       types.StringValue("name"),
				Description:                types.StringValue("desc"),
				Labels:                     types.MapValueMust(types.Int64Type, map[string]attr.Value{"key": types.Int64Value(33)}),
				DeletedExperimentRetention: types.StringValue("50d"),
			},
			expected: nil,
			isValid:  false,
		},
		{
			description: "should convert correctly",
			input: &Model{
				Name:                       types.StringValue("name"),
				Description:                types.StringValue("desc"),
				Labels:                     types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				DeletedExperimentRetention: types.StringNull(),
			},
			expected: &modelexperiments.CreateInstancePayload{
				Name:                       "name",
				Description:                new("desc"),
				Labels:                     &map[string]string{"key": "value"},
				DeletedExperimentRetention: nil,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

	tests := []struct {
		description string
		input       *Model
		expected    *modelexperiments.PartialUpdateInstancePayload
		isValid     bool
	}{
		{
			description: "should error on nil input",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
		{
			description: "should error when map is not correct",
			input: &Model{
				Name:                       types.StringValue("name"),
				Description:                types.StringValue("desc"),
				Labels:                     types.MapValueMust(types.Int64Type, map[string]attr.Value{"key": types.Int64Value(33)}),
				DeletedExperimentRetention: types.StringValue("50d"),
			},
			expected: nil,
			isValid:  false,
		},
		{
			description: "should convert correctly",
			input: &Model{
				Name:                       types.StringValue("name"),
				Description:                types.StringValue("desc"),
				Labels:                     types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				DeletedExperimentRetention: types.StringValue("50d"),
			},
			expected: &modelexperiments.PartialUpdateInstancePayload{
				Name:                       new("name"),
				Description:                new("desc"),
				Labels:                     &map[string]string{"key": "value"},
				DeletedExperimentRetention: new("50d"),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

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
