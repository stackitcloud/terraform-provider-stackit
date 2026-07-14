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
				BucketName:                 types.StringValue("bucketName"),
				DeletedExperimentRetention: types.StringValue("30d"),
				Url:                        types.StringValue("url"),
				Labels:                     types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := mapInstance(ctx, &tt.input, tt.state, "eu01")
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
