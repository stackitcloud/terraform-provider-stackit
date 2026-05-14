package destination

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi"
)

func fixtureDatasourceModel(mods ...func(model *DatasourceModel)) *DatasourceModel {
	cfg, _ := types.ObjectValueFrom(context.Background(), datasourceConfigTypes, datasourceConfig{
		ConfigType:    types.StringValue(""),
		Filter:        types.ObjectNull(datasourceFilterTypes),
		OpenTelemetry: types.ObjectNull(datasourceOpenTelemetryTypes),
		S3:            types.ObjectNull(datasourceS3Types),
	})
	model := &DatasourceModel{
		ID:             types.StringValue("pid,rid,iid,dsid"),
		DestinationID:  types.StringValue("dsid"),
		InstanceID:     types.StringValue("iid"),
		Region:         types.StringValue("rid"),
		ProjectID:      types.StringValue("pid"),
		Description:    types.String{},
		DisplayName:    types.StringValue("test"),
		Config:         cfg,
		CredentialType: types.StringValue(""),
		CreationTime:   types.StringValue(testTime.Format(time.RFC3339)),
		Status:         types.StringValue("active"),
	}
	for _, mod := range mods {
		mod(model)
	}
	return model
}

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		input       *telemetryrouter.DestinationResponse
		expected    *DatasourceModel
		wantErr     bool
	}{
		{
			description: "min values",
			input:       fixtureDestinationResponse(),
			expected:    fixtureDatasourceModel(),
		},
		{
			description: "OpenTelemetry with filter",
			input: fixtureDestinationResponse(func(destination *telemetryrouter.DestinationResponse) {
				destination.Description = new("description")
				destination.DisplayName = "display-name"
				destination.CredentialType = "bearerToken"
				destination.Config = telemetryrouter.DestinationConfig{
					ConfigType: "OpenTelemetry",
					Filter: &telemetryrouter.ConfigFilter{
						Attributes: []telemetryrouter.ConfigFilterAttributes{
							{
								Key:     "test",
								Level:   "logRecord",
								Matcher: "=",
								Values:  []string{"a", "b"},
							},
						},
					},
					OpenTelemetry: &telemetryrouter.DestinationConfigOpenTelemetry{
						BearerToken: new("bearer-token"),
						Uri:         "https://example.test",
					},
				}
			}),
			expected: fixtureDatasourceModel(func(model *DatasourceModel) {
				ctx := context.Background()
				vals, _ := types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
				attrs, _ := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, []datasourceAttribute{
					{
						Key:     types.StringValue("test"),
						Level:   types.StringValue("logRecord"),
						Matcher: types.StringValue("="),
						Values:  vals,
					},
				})
				fltr, _ := types.ObjectValueFrom(ctx, datasourceFilterTypes, datasourceFilter{
					Attributes: attrs,
				})
				openTelemetryVal, _ := types.ObjectValueFrom(ctx, datasourceOpenTelemetryTypes, datasourceOpenTelemetry{
					Uri: types.StringValue("https://example.test"),
				})
				cfg, _ := types.ObjectValueFrom(context.Background(), datasourceConfigTypes, datasourceConfig{
					ConfigType:    types.StringValue("OpenTelemetry"),
					Filter:        fltr,
					OpenTelemetry: openTelemetryVal,
					S3:            types.ObjectNull(datasourceS3Types),
				})
				model.Config = cfg
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CredentialType = types.StringValue("bearerToken")
			}),
		},
		{
			description: "S3 with filter",
			input: fixtureDestinationResponse(func(destination *telemetryrouter.DestinationResponse) {
				destination.Description = new("description")
				destination.DisplayName = "display-name"
				destination.CredentialType = "accessKey"
				destination.Config = telemetryrouter.DestinationConfig{
					ConfigType: "S3",
					Filter: &telemetryrouter.ConfigFilter{
						Attributes: []telemetryrouter.ConfigFilterAttributes{
							{
								Key:     "test",
								Level:   "logRecord",
								Matcher: "=",
								Values:  []string{"a", "b"},
							},
						},
					},
					S3: &telemetryrouter.DestinationConfigS3{
						AccessKey: &telemetryrouter.DestinationConfigS3AccessKey{
							Id:     "id",
							Secret: "secret",
						},
						Bucket:   "bucket",
						Endpoint: "https://example.test",
					},
				}
			}),
			expected: fixtureDatasourceModel(func(model *DatasourceModel) {
				ctx := context.Background()
				vals, _ := types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
				attrs, _ := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, []datasourceAttribute{
					{
						Key:     types.StringValue("test"),
						Level:   types.StringValue("logRecord"),
						Matcher: types.StringValue("="),
						Values:  vals,
					},
				})
				fltr, _ := types.ObjectValueFrom(ctx, datasourceFilterTypes, datasourceFilter{
					Attributes: attrs,
				})
				s3Val, _ := types.ObjectValueFrom(ctx, datasourceS3Types, datasourceS3{
					Bucket:   types.StringValue("bucket"),
					Endpoint: types.StringValue("https://example.test"),
				})
				cfg, _ := types.ObjectValueFrom(context.Background(), datasourceConfigTypes, datasourceConfig{
					ConfigType:    types.StringValue("S3"),
					Filter:        fltr,
					OpenTelemetry: types.ObjectNull(datasourceOpenTelemetryTypes),
					S3:            s3Val,
				})
				model.Config = cfg
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CredentialType = types.StringValue("accessKey")
			}),
		},
		{
			description: "nil input",
			wantErr:     true,
			expected:    fixtureDatasourceModel(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &DatasourceModel{
				ProjectID:  tt.expected.ProjectID,
				Region:     tt.expected.Region,
				InstanceID: tt.expected.InstanceID,
			}
			err := mapDatasourceFields(context.Background(), tt.input, state)
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
