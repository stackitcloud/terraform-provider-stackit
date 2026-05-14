package destination

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

func fixtureDestinationResponse(mods ...func(destination *telemetryrouter.DestinationResponse)) *telemetryrouter.DestinationResponse {
	destination := &telemetryrouter.DestinationResponse{
		Id:           "dsid",
		DisplayName:  "test",
		CreationTime: testTime,
		Status:       "active",
	}
	for _, mod := range mods {
		mod(destination)
	}
	return destination
}

func fixtureModel(mods ...func(model *Model)) *Model {
	cfg, _ := types.ObjectValueFrom(context.Background(), configTypes, config{
		ConfigType:    types.StringValue(""),
		Filter:        types.ObjectNull(filterTypes),
		OpenTelemetry: types.ObjectNull(openTelemetryTypes),
		S3:            types.ObjectNull(s3Types),
	})
	model := &Model{
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

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *telemetryrouter.DestinationResponse
		expected    *Model
		wantErr     bool
	}{
		{
			description: "min values",
			input:       fixtureDestinationResponse(),
			expected:    fixtureModel(),
		},
		{
			description: "OpenTelemetry bearer token with filter",
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
			expected: fixtureModel(func(model *Model) {
				ctx := context.Background()
				vals, _ := types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
				attrs, _ := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, []attribute{
					{
						Key:     types.StringValue("test"),
						Level:   types.StringValue("logRecord"),
						Matcher: types.StringValue("="),
						Values:  vals,
					},
				})
				fltr, _ := types.ObjectValueFrom(ctx, filterTypes, filter{
					Attributes: attrs,
				})
				openTelemetryVal, _ := types.ObjectValueFrom(ctx, openTelemetryTypes, openTelemetry{
					BasicAuth:   types.ObjectNull(basicAuthTypes),
					BearerToken: types.StringPointerValue(new("bearer-token")),
					Uri:         types.StringValue("https://example.test"),
				})
				cfg, _ := types.ObjectValueFrom(context.Background(), configTypes, config{
					ConfigType:    types.StringValue("OpenTelemetry"),
					Filter:        fltr,
					OpenTelemetry: openTelemetryVal,
					S3:            types.ObjectNull(s3Types),
				})
				model.Config = cfg
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CredentialType = types.StringValue("bearerToken")
			}),
		},
		{
			description: "OpenTelemetry basic auth",
			input: fixtureDestinationResponse(func(destination *telemetryrouter.DestinationResponse) {
				destination.Description = new("description")
				destination.DisplayName = "display-name"
				destination.CredentialType = "bearerToken"
				destination.Config = telemetryrouter.DestinationConfig{
					ConfigType: "OpenTelemetry",
					OpenTelemetry: &telemetryrouter.DestinationConfigOpenTelemetry{
						BasicAuth: &telemetryrouter.DestinationConfigOpenTelemetryBasicAuth{
							Password: "pass",
							Username: "user",
						},
						Uri: "https://example.test",
					},
				}
			}),
			expected: fixtureModel(func(model *Model) {
				ctx := context.Background()
				basicAuthVal, _ := types.ObjectValueFrom(ctx, basicAuthTypes, basicAuth{
					Username: types.StringValue("user"),
					Password: types.StringValue("pass"),
				})
				openTelemetryVal, _ := types.ObjectValueFrom(ctx, openTelemetryTypes, openTelemetry{
					BasicAuth:   basicAuthVal,
					BearerToken: types.StringNull(),
					Uri:         types.StringValue("https://example.test"),
				})
				cfg, _ := types.ObjectValueFrom(context.Background(), configTypes, config{
					ConfigType:    types.StringValue("OpenTelemetry"),
					Filter:        types.ObjectNull(filterTypes),
					OpenTelemetry: openTelemetryVal,
					S3:            types.ObjectNull(s3Types),
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
			expected: fixtureModel(func(model *Model) {
				ctx := context.Background()
				vals, _ := types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
				attrs, _ := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, []attribute{
					{
						Key:     types.StringValue("test"),
						Level:   types.StringValue("logRecord"),
						Matcher: types.StringValue("="),
						Values:  vals,
					},
				})
				fltr, _ := types.ObjectValueFrom(ctx, filterTypes, filter{
					Attributes: attrs,
				})
				ak, _ := types.ObjectValueFrom(ctx, accessKeyTypes, accessKey{
					ID:     types.StringValue("id"),
					Secret: types.StringValue("secret"),
				})
				s3Val, _ := types.ObjectValueFrom(ctx, s3Types, s3{
					AccessKey: ak,
					Bucket:    types.StringValue("bucket"),
					Endpoint:  types.StringValue("https://example.test"),
				})
				cfg, _ := types.ObjectValueFrom(context.Background(), configTypes, config{
					ConfigType:    types.StringValue("S3"),
					Filter:        fltr,
					OpenTelemetry: types.ObjectNull(openTelemetryTypes),
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
			expected:    fixtureModel(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectID:  tt.expected.ProjectID,
				Region:     tt.expected.Region,
				InstanceID: tt.expected.InstanceID,
				Config:     tt.expected.Config,
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
		expected       *telemetryrouter.CreateDestinationPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected: &telemetryrouter.CreateDestinationPayload{
				DisplayName: "test",
			},
		},
		{
			description: "open telemetry basic auth values",
			model: fixtureModel(func(model *Model) {
				ctx := context.Background()
				basicAuthVal, _ := types.ObjectValueFrom(ctx, basicAuthTypes, basicAuth{
					Username: types.StringValue("user"),
					Password: types.StringValue("pass"),
				})
				openTelemetryVal, _ := types.ObjectValueFrom(ctx, openTelemetryTypes, openTelemetry{
					BasicAuth:   basicAuthVal,
					BearerToken: types.StringNull(),
					Uri:         types.StringValue("https://example.test"),
				})
				cfg, _ := types.ObjectValueFrom(context.Background(), configTypes, config{
					ConfigType:    types.StringValue("OpenTelemetry"),
					Filter:        types.ObjectNull(filterTypes),
					OpenTelemetry: openTelemetryVal,
					S3:            types.ObjectNull(s3Types),
				})
				model.Config = cfg
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CredentialType = types.StringValue("bearerToken")
			}),
			expected: &telemetryrouter.CreateDestinationPayload{
				Description: new("description"),
				DisplayName: "display-name",
				Config: telemetryrouter.DestinationConfig{
					ConfigType: "OpenTelemetry",
					OpenTelemetry: &telemetryrouter.DestinationConfigOpenTelemetry{
						BasicAuth: &telemetryrouter.DestinationConfigOpenTelemetryBasicAuth{
							Username: "user",
							Password: "pass",
						},
						Uri: "https://example.test",
					},
				},
			},
		},
		{
			description: "open telemetry bearer token with filter values",
			model: fixtureModel(func(model *Model) {
				ctx := context.Background()
				vals, _ := types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
				attrs, _ := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, []attribute{
					{
						Key:     types.StringValue("test"),
						Level:   types.StringValue("logRecord"),
						Matcher: types.StringValue("="),
						Values:  vals,
					},
				})
				fltr, _ := types.ObjectValueFrom(ctx, filterTypes, filter{
					Attributes: attrs,
				})
				openTelemetryVal, _ := types.ObjectValueFrom(ctx, openTelemetryTypes, openTelemetry{
					BasicAuth:   types.ObjectNull(basicAuthTypes),
					BearerToken: types.StringValue("bearerToken"),
					Uri:         types.StringValue("https://example.test"),
				})
				cfg, _ := types.ObjectValueFrom(context.Background(), configTypes, config{
					ConfigType:    types.StringValue("OpenTelemetry"),
					Filter:        fltr,
					OpenTelemetry: openTelemetryVal,
					S3:            types.ObjectNull(s3Types),
				})
				model.Config = cfg
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CredentialType = types.StringValue("bearerToken")
			}),
			expected: &telemetryrouter.CreateDestinationPayload{
				Description: new("description"),
				DisplayName: "display-name",
				Config: telemetryrouter.DestinationConfig{
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
						BearerToken: new("bearerToken"),
						Uri:         "https://example.test",
					},
				},
			},
		},
		{
			description: "s3 values",
			model: fixtureModel(func(model *Model) {
				ctx := context.Background()
				ak, _ := types.ObjectValueFrom(ctx, accessKeyTypes, accessKey{
					ID:     types.StringValue("id"),
					Secret: types.StringValue("secret"),
				})
				s3Val, _ := types.ObjectValueFrom(ctx, s3Types, s3{
					AccessKey: ak,
					Bucket:    types.StringValue("bucket"),
					Endpoint:  types.StringValue("https://example.test"),
				})
				cfg, _ := types.ObjectValueFrom(context.Background(), configTypes, config{
					ConfigType:    types.StringValue("S3"),
					Filter:        types.ObjectNull(filterTypes),
					OpenTelemetry: types.ObjectNull(openTelemetryTypes),
					S3:            s3Val,
				})
				model.Config = cfg
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CredentialType = types.StringValue("accessKey")
			}),
			expected: &telemetryrouter.CreateDestinationPayload{
				Description: new("description"),
				DisplayName: "display-name",
				Config: telemetryrouter.DestinationConfig{
					ConfigType: "S3",
					S3: &telemetryrouter.DestinationConfigS3{
						AccessKey: &telemetryrouter.DestinationConfigS3AccessKey{
							Id:     "id",
							Secret: "secret",
						},
						Bucket:   "bucket",
						Endpoint: "https://example.test",
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
		expected       *telemetryrouter.UpdateDestinationPayload
		wantErrMessage string
	}{
		{
			description: "min values",
			model:       fixtureModel(),
			expected: &telemetryrouter.UpdateDestinationPayload{
				DisplayName: new("test"),
				Config: &telemetryrouter.DestinationConfig{
					ConfigType: "",
				},
			},
		},
		{
			description: "open telemetry basic auth values",
			model: fixtureModel(func(model *Model) {
				ctx := context.Background()
				basicAuthVal, _ := types.ObjectValueFrom(ctx, basicAuthTypes, basicAuth{
					Username: types.StringValue("user"),
					Password: types.StringValue("pass"),
				})
				openTelemetryVal, _ := types.ObjectValueFrom(ctx, openTelemetryTypes, openTelemetry{
					BasicAuth:   basicAuthVal,
					BearerToken: types.StringNull(),
					Uri:         types.StringValue("https://example.test"),
				})
				cfg, _ := types.ObjectValueFrom(context.Background(), configTypes, config{
					ConfigType:    types.StringValue("OpenTelemetry"),
					Filter:        types.ObjectNull(filterTypes),
					OpenTelemetry: openTelemetryVal,
					S3:            types.ObjectNull(s3Types),
				})
				model.Config = cfg
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CredentialType = types.StringValue("bearerToken")
			}),
			expected: &telemetryrouter.UpdateDestinationPayload{
				Description: new("description"),
				DisplayName: new("display-name"),
				Config: &telemetryrouter.DestinationConfig{
					ConfigType: "OpenTelemetry",
					OpenTelemetry: &telemetryrouter.DestinationConfigOpenTelemetry{
						BasicAuth: &telemetryrouter.DestinationConfigOpenTelemetryBasicAuth{
							Username: "user",
							Password: "pass",
						},
						Uri: "https://example.test",
					},
				},
			},
		},
		{
			description: "open telemetry bearer token with filter values",
			model: fixtureModel(func(model *Model) {
				ctx := context.Background()
				vals, _ := types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
				attrs, _ := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attributeTypes}, []attribute{
					{
						Key:     types.StringValue("test"),
						Level:   types.StringValue("logRecord"),
						Matcher: types.StringValue("="),
						Values:  vals,
					},
				})
				fltr, _ := types.ObjectValueFrom(ctx, filterTypes, filter{
					Attributes: attrs,
				})
				openTelemetryVal, _ := types.ObjectValueFrom(ctx, openTelemetryTypes, openTelemetry{
					BasicAuth:   types.ObjectNull(basicAuthTypes),
					BearerToken: types.StringValue("bearerToken"),
					Uri:         types.StringValue("https://example.test"),
				})
				cfg, _ := types.ObjectValueFrom(context.Background(), configTypes, config{
					ConfigType:    types.StringValue("OpenTelemetry"),
					Filter:        fltr,
					OpenTelemetry: openTelemetryVal,
					S3:            types.ObjectNull(s3Types),
				})
				model.Config = cfg
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CredentialType = types.StringValue("bearerToken")
			}),
			expected: &telemetryrouter.UpdateDestinationPayload{
				Description: new("description"),
				DisplayName: new("display-name"),
				Config: &telemetryrouter.DestinationConfig{
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
						BearerToken: new("bearerToken"),
						Uri:         "https://example.test",
					},
				},
			},
		},
		{
			description: "s3 values",
			model: fixtureModel(func(model *Model) {
				ctx := context.Background()
				ak, _ := types.ObjectValueFrom(ctx, accessKeyTypes, accessKey{
					ID:     types.StringValue("id"),
					Secret: types.StringValue("secret"),
				})
				s3Val, _ := types.ObjectValueFrom(ctx, s3Types, s3{
					AccessKey: ak,
					Bucket:    types.StringValue("bucket"),
					Endpoint:  types.StringValue("https://example.test"),
				})
				cfg, _ := types.ObjectValueFrom(context.Background(), configTypes, config{
					ConfigType:    types.StringValue("S3"),
					Filter:        types.ObjectNull(filterTypes),
					OpenTelemetry: types.ObjectNull(openTelemetryTypes),
					S3:            s3Val,
				})
				model.Config = cfg
				model.Description = types.StringValue("description")
				model.DisplayName = types.StringValue("display-name")
				model.CredentialType = types.StringValue("accessKey")
			}),
			expected: &telemetryrouter.UpdateDestinationPayload{
				Description: new("description"),
				DisplayName: new("display-name"),
				Config: &telemetryrouter.DestinationConfig{
					ConfigType: "S3",
					S3: &telemetryrouter.DestinationConfigS3{
						AccessKey: &telemetryrouter.DestinationConfigS3AccessKey{
							Id:     "id",
							Secret: "secret",
						},
						Bucket:   "bucket",
						Endpoint: "https://example.test",
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

			diff := cmp.Diff(got, tt.expected /*, cmp.Comparer(compareNullableString), cmp.Comparer(compareNullableInt32)*/)
			if diff != "" {
				t.Fatalf("Payload does not match: %s", diff)
			}
		})
	}
}
