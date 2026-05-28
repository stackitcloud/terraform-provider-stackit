package instance

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi"
)

var testTime = time.Now()

func fixtureInstance(mods ...func(instance *telemetryrouter.TelemetryRouterResponse)) *telemetryrouter.TelemetryRouterResponse {
	instance := &telemetryrouter.TelemetryRouterResponse{
		Id:           "iid",
		CreationTime: testTime,
		Uri:          "uri",
		Status:       "active",
	}
	for _, mod := range mods {
		mod(instance)
	}
	return instance
}

func fixtureModel(mods ...func(model *Model)) *Model {
	model := &Model{
		ID:           types.StringValue("pid,rid,iid"),
		InstanceID:   types.StringValue("iid"),
		Region:       types.StringValue("rid"),
		ProjectID:    types.StringValue("pid"),
		DisplayName:  types.String{},
		Description:  types.String{},
		CreationTime: types.StringValue(testTime.Format(time.RFC3339)),
		URI:          types.StringValue("uri"),
		Status:       types.StringValue("active"),
	}
	for _, mod := range mods {
		mod(model)
	}
	return model
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *telemetryrouter.TelemetryRouterResponse
		expected    *Model
		wantErr     bool
	}{
		{
			description: "min values",
			input: fixtureInstance(func(instance *telemetryrouter.TelemetryRouterResponse) {
				instance.DisplayName = "display-name"
			}),
			expected: fixtureModel(func(model *Model) {
				model.DisplayName = types.StringValue("display-name")
			}),
		},
		{
			description: "max values",
			input: fixtureInstance(func(instance *telemetryrouter.TelemetryRouterResponse) {
				instance.DisplayName = "display-name"
				instance.Description = new("description")
				instance.Uri = "query-url"
				instance.Filter = &telemetryrouter.ConfigFilter{
					Attributes: []telemetryrouter.ConfigFilterAttributes{
						{
							Key:     "test",
							Level:   "resource",
							Matcher: "!=",
							Values:  []string{"a", "b"},
						},
					},
				}
			}),
			expected: fixtureModel(func(model *Model) {
				ctx := context.Background()
				vals, _ := types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
				attrs, _ := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, []attribute{
					{
						Key:     types.StringValue("test"),
						Level:   types.StringValue("resource"),
						Matcher: types.StringValue("!="),
						Values:  vals,
					},
				})
				fltr, _ := types.ObjectValueFrom(ctx, filterTypes, filter{
					Attributes: attrs,
				})
				model.Filter = fltr
				model.DisplayName = types.StringValue("display-name")
				model.Description = types.StringValue("description")
				model.URI = types.StringValue("query-url")
			}),
		},
		{
			description: "nil input",
			wantErr:     true,
			expected:    fixtureModel(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectID: tt.expected.ProjectID,
				Region:    tt.expected.Region,
			}
			err := mapFields(context.Background(), tt.input, state)
			if tt.wantErr && err == nil {
				t.Fatalf("Should have failed")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if !tt.wantErr {
				diff := cmp.Diff(state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		expected       *telemetryrouter.CreateTelemetryRouterPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected:    &telemetryrouter.CreateTelemetryRouterPayload{},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				ctx := context.Background()
				vals, _ := types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
				attrs, _ := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, []attribute{
					{
						Key:     types.StringValue("test"),
						Level:   types.StringValue("resource"),
						Matcher: types.StringValue("!="),
						Values:  vals,
					},
				})
				fltr, _ := types.ObjectValueFrom(ctx, filterTypes, filter{
					Attributes: attrs,
				})
				model.Filter = fltr
				model.DisplayName = types.StringValue("display-name")
				model.Description = types.StringValue("description")
			}),
			expected: &telemetryrouter.CreateTelemetryRouterPayload{
				DisplayName: "display-name",
				Description: new("description"),
				Filter: &telemetryrouter.ConfigFilter{
					Attributes: []telemetryrouter.ConfigFilterAttributes{
						{
							Key:     "test",
							Level:   "resource",
							Matcher: "!=",
							Values:  []string{"a", "b"},
						},
					},
				},
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toCreatePayload(t.Context(), diag.Diagnostics{}, tt.model)
			if tt.wantErrMessage != "" && (err == nil || err.Error() != tt.wantErrMessage) {
				t.Fatalf("Expected error: %v, got: %v", tt.wantErrMessage, err)
			}
			if tt.wantErrMessage == "" && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			diff := cmp.Diff(got, tt.expected)
			if diff != "" {
				t.Fatalf("Payload does not match: %s", diff)
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description    string
		model          *Model
		expected       *telemetryrouter.UpdateTelemetryRouterPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected:    &telemetryrouter.UpdateTelemetryRouterPayload{},
		},
		{
			description: "max values",
			model: fixtureModel(func(model *Model) {
				ctx := context.Background()
				vals, _ := types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
				attrs, _ := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, []attribute{
					{
						Key:     types.StringValue("test"),
						Level:   types.StringValue("resource"),
						Matcher: types.StringValue("!="),
						Values:  vals,
					},
				})
				fltr, _ := types.ObjectValueFrom(ctx, filterTypes, filter{
					Attributes: attrs,
				})
				model.Filter = fltr
				model.DisplayName = types.StringValue("display-name")
				model.Description = types.StringValue("description")
			}),
			expected: &telemetryrouter.UpdateTelemetryRouterPayload{
				DisplayName: new("display-name"),
				Description: new("description"),
				Filter: &telemetryrouter.ConfigFilter{
					Attributes: []telemetryrouter.ConfigFilterAttributes{
						{
							Key:     "test",
							Level:   "resource",
							Matcher: "!=",
							Values:  []string{"a", "b"},
						},
					},
				},
			},
		},
		{
			description:    "nil model",
			wantErrMessage: "missing model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := toUpdatePayload(t.Context(), diag.Diagnostics{}, tt.model)
			if tt.wantErrMessage != "" && (err == nil || err.Error() != tt.wantErrMessage) {
				t.Fatalf("Expected error: %v, got: %v", tt.wantErrMessage, err)
			}
			if tt.wantErrMessage == "" && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			diff := cmp.Diff(got, tt.expected)
			if diff != "" {
				t.Fatalf("Payload does not match: %s", diff)
			}
		})
	}
}
