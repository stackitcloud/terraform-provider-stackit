package artifactory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	registry "github.com/stackitcloud/stackit-sdk-go/services/registry/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

func fixtureDataSourceModel(t *testing.T, mods ...func(model *DataSourceModel)) *DataSourceModel {
	t.Helper()
	model := &DataSourceModel{
		Id:            types.StringValue("pid,reg,art-id-123"),
		ProjectId:     types.StringValue("pid"),
		Region:        types.StringValue("reg"),
		ArtifactoryId: types.StringValue("art-id-123"),
		Name:          types.StringValue("my-artifactory"),
		CreateTime:    types.StringValue(testTime.Format(time.RFC3339)),
		State:         types.StringValue(string(registry.ARTIFACTORYSTATE_ACTIVE)),
		Url:           types.StringValue("https://art-id-123.cr.stackit.cloud"),
		Proxy:         types.ObjectNull(dsProxyTypes),
	}
	for _, mod := range mods {
		mod(model)
	}
	return model
}

func TestArtifactoryDataSource_Metadata(t *testing.T) {
	ds := NewArtifactoryDataSource()
	var resp datasource.MetadataResponse
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "stackit"}, &resp)
	if resp.TypeName != "stackit_containerregistry_artifactory" {
		t.Errorf("expected type name %q, got %q", "stackit_containerregistry_artifactory", resp.TypeName)
	}
}

func TestArtifactoryDataSource_Schema(t *testing.T) {
	ds := NewArtifactoryDataSource()
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if len(resp.Schema.Attributes) == 0 {
		t.Errorf("expected non-empty schema attributes")
	}
}

func TestArtifactoryDataSource_Configure(t *testing.T) {
	ds := &artifactoryDataSource{}
	ctx := context.Background()

	t.Run("valid provider data", func(t *testing.T) {
		providerData, err := core.NewProviderDataInternal(core.ProviderData{}, &core.MockClientFactory{})
		if err != nil {
			t.Fatalf("unexpected error creating provider data: %v", err)
		}
		var resp datasource.ConfigureResponse
		ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: providerData}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("unexpected error diagnostics: %v", resp.Diagnostics)
		}
		if ds.client == nil {
			t.Error("expected client to be configured")
		}
	})

	t.Run("nil provider data", func(t *testing.T) {
		var resp datasource.ConfigureResponse
		ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: nil}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("expected no error diagnostics on nil provider data, got %v", resp.Diagnostics)
		}
	})

	t.Run("invalid provider data type", func(t *testing.T) {
		var resp datasource.ConfigureResponse
		ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: "invalid"}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error diagnostics on invalid provider data")
		}
	})
}

func TestMapDataSourceFields(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		description string
		input       *registry.Artifactory
		expected    *DataSourceModel
		wantErr     bool
	}{
		{
			description: "min values (no proxy)",
			input:       fixtureArtifactory(),
			expected:    fixtureDataSourceModel(t),
		},
		{
			description: "values with proxy",
			input: fixtureArtifactory(func(art *registry.Artifactory) {
				desc := "Docker Hub mirror"
				art.Proxy = &registry.Proxy{
					RegistryType: "docker-hub",
					Url:          "https://registry-1.docker.io",
					Description:  &desc,
				}
			}),
			expected: fixtureDataSourceModel(t, func(model *DataSourceModel) {
				proxyObj, diags := types.ObjectValueFrom(ctx, dsProxyTypes, dsProxyModel{
					RegistryType: types.StringValue("docker-hub"),
					Url:          types.StringValue("https://registry-1.docker.io"),
					Description:  types.StringValue("Docker Hub mirror"),
				})
				if diags.HasError() {
					t.Fatalf("unexpected diags creating proxy: %v", diags)
				}
				model.Proxy = proxyObj
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &DataSourceModel{
				ProjectId: types.StringValue("pid"),
			}
			err := mapDataSourceFields(ctx, tt.input, model, "reg")
			if (err != nil) != tt.wantErr {
				t.Fatalf("mapDataSourceFields() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(model, tt.expected); diff != "" {
				t.Errorf("mapDataSourceFields() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestMapDataSourceFields_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("nil artifactory", func(t *testing.T) {
		model := &DataSourceModel{
			ProjectId: types.StringValue("pid"),
		}
		err := mapDataSourceFields(ctx, nil, model, "reg")
		if err == nil {
			t.Error("expected error when artifactory is nil")
		}
	})

	t.Run("nil model", func(t *testing.T) {
		art := fixtureArtifactory()
		err := mapDataSourceFields(ctx, art, nil, "reg")
		if err == nil {
			t.Error("expected error when model is nil")
		}
	})
}

func TestArtifactoryDataSource_Read(t *testing.T) {
	ctx := context.Background()
	dsBase := NewArtifactoryDataSource()
	var schemaResp datasource.SchemaResponse
	dsBase.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)

	configAttrTypes := map[string]attr.Type{
		"id":             types.StringType,
		"project_id":     types.StringType,
		"region":         types.StringType,
		"artifactory_id": types.StringType,
		"name":           types.StringType,
		"create_time":    types.StringType,
		"state":          types.StringType,
		"url":            types.StringType,
		"proxy":          types.ObjectType{AttrTypes: dsProxyTypes},
	}

	configModel := DataSourceModel{
		ProjectId:     types.StringValue("pid"),
		Region:        types.StringValue("reg"),
		ArtifactoryId: types.StringValue("art-id-123"),
		Id:            types.StringNull(),
		Name:          types.StringNull(),
		CreateTime:    types.StringNull(),
		State:         types.StringNull(),
		Url:           types.StringNull(),
		Proxy:         types.ObjectNull(dsProxyTypes),
	}
	configVal, diags := types.ObjectValueFrom(ctx, configAttrTypes, configModel)
	if diags.HasError() {
		t.Fatalf("unexpected diags creating configVal: %v", diags)
	}
	rawVal, err := configVal.ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("unexpected error converting to terraform value: %v", err)
	}

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    rawVal,
		},
	}

	t.Run("read success", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				return fixtureArtifactory(), nil
			}),
		}
		ds := &artifactoryDataSource{client: mockClient}

		resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		ds.Read(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}

		var resultModel DataSourceModel
		diags = resp.State.Get(ctx, &resultModel)
		if diags.HasError() {
			t.Fatalf("unexpected diags getting state: %v", diags)
		}
		if resultModel.Id.ValueString() != "pid,reg,art-id-123" {
			t.Errorf("expected id 'pid,reg,art-id-123', got %q", resultModel.Id.ValueString())
		}
		if resultModel.Name.ValueString() != "my-artifactory" {
			t.Errorf("expected name 'my-artifactory', got %q", resultModel.Name.ValueString())
		}
	})

	t.Run("read API error", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				return nil, fmt.Errorf("api error")
			}),
		}
		ds := &artifactoryDataSource{client: mockClient}

		resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		ds.Read(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics on read API error")
		}
	})

	t.Run("read 404 not found", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				return nil, &oapierror.GenericOpenAPIError{StatusCode: 404}
			}),
		}
		ds := &artifactoryDataSource{client: mockClient}

		resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		ds.Read(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics on read 404")
		}
	})
}
