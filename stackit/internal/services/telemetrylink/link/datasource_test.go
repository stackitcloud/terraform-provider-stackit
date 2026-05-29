package link

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	telemetrylink "github.com/stackitcloud/stackit-sdk-go/services/telemetrylink/v1betaapi"
)

func fixtureDataSourceModel(mods ...func(model *DataSourceModel)) *DataSourceModel {
	model := &DataSourceModel{
		ID:                types.StringValue("rtp,rid,reg"),
		LinkID:            types.StringValue("lid"),
		Region:            types.StringValue("reg"),
		ResourceType:      types.StringValue("rtp"),
		ResourceID:        types.StringValue("rid"),
		DisplayName:       types.StringValue("name"),
		Description:       types.String{},
		TelemetryRouterID: types.StringValue("tlmrid"),
		CreateTime:        types.StringValue(testTime.Format(time.RFC3339)),
		Status:            types.StringValue("active"),
	}
	for _, mod := range mods {
		mod(model)
	}
	return model
}

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		input       *telemetrylink.TelemetryLinkResponse
		expected    *DataSourceModel
		wantErr     bool
	}{
		{
			description: "min values",
			input:       fixtureLink(),
			expected:    fixtureDataSourceModel(),
		},
		{
			description: "max values",
			input: fixtureLink(func(link *telemetrylink.TelemetryLinkResponse) {
				link.Description = new("description")
				link.DisplayName = "display-name"
				link.AccessToken = new("access-token")
				link.TelemetryRouterId = "tlmr-id"
			}),
			expected: fixtureDataSourceModel(func(model *DataSourceModel) {
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.TelemetryRouterID = types.StringValue("tlmr-id")
			}),
		},
		{
			description: "nil input",
			wantErr:     true,
			expected:    fixtureDataSourceModel(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &DataSourceModel{
				ResourceType: tt.expected.ResourceType,
				ResourceID:   tt.expected.ResourceID,
				Region:       tt.expected.Region,
			}
			err := mapDataSourceFields(context.Background(), tt.input, state, tt.expected.Region.ValueString())
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
