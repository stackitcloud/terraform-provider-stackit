package artifactories

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

var testTime = time.Now()

func TestArtifactoriesDataSource_Metadata(t *testing.T) {
	ds := NewArtifactoriesDataSource()
	var resp datasource.MetadataResponse
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "stackit"}, &resp)
	if resp.TypeName != "stackit_containerregistry_artifactories" {
		t.Errorf("expected type name %q, got %q", "stackit_containerregistry_artifactories", resp.TypeName)
	}
}

func TestArtifactoriesDataSource_Schema(t *testing.T) {
	ds := NewArtifactoriesDataSource()
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if len(resp.Schema.Attributes) == 0 {
		t.Errorf("expected non-empty schema attributes")
	}
}

func TestArtifactoriesDataSource_Configure(t *testing.T) {
	ds := &artifactoriesDataSource{}
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

func TestMapFields(t *testing.T) {
	ctx := context.Background()

	desc := "upstream docker"
	artifactories := []registry.Artifactory{
		{
			Id:                 "art-1",
			Name:               "artifactory-1",
			ConsumedQuotaBytes: 100,
			CreateTime:         testTime,
			State:              registry.ARTIFACTORYSTATE_ACTIVE,
			Url:                "https://art-1.cr.stackit.cloud",
		},
		{
			Id:                 "art-2",
			Name:               "artifactory-2",
			ConsumedQuotaBytes: 200,
			CreateTime:         testTime,
			State:              registry.ARTIFACTORYSTATE_ACTIVE,
			Url:                "https://art-2.cr.stackit.cloud",
			Proxy: &registry.Proxy{
				RegistryType: "docker-hub",
				Url:          "https://registry-1.docker.io",
				Description:  &desc,
				Credentials: &registry.RedactedCredentials{
					AccessKey: "test-user",
				},
			},
		},
	}

	model := &DataSourceModel{
		ProjectId: types.StringValue("pid"),
	}

	err := mapFields(ctx, artifactories, model, "reg")
	if err != nil {
		t.Fatalf("mapFields() error = %v", err)
	}

	if model.Id.ValueString() != "pid,reg" {
		t.Errorf("expected id 'pid,reg', got %q", model.Id.ValueString())
	}
	if model.Region.ValueString() != "reg" {
		t.Errorf("expected region 'reg', got %q", model.Region.ValueString())
	}

	proxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, proxyItemModel{
		RegistryType: types.StringValue("docker-hub"),
		Url:          types.StringValue("https://registry-1.docker.io"),
		Description:  types.StringValue("upstream docker"),
	})
	if diags.HasError() {
		t.Fatalf("creating proxyObj: %v", diags)
	}

	item1, diags := types.ObjectValueFrom(ctx, artifactoryItemTypes, artifactoryItemModel{
		ArtifactoryId: types.StringValue("art-1"),
		Name:          types.StringValue("artifactory-1"),
		CreateTime:    types.StringValue(testTime.Format(time.RFC3339)),
		State:         types.StringValue(string(registry.ARTIFACTORYSTATE_ACTIVE)),
		Url:           types.StringValue("https://art-1.cr.stackit.cloud"),
		Proxy:         types.ObjectNull(proxyTypes),
	})
	if diags.HasError() {
		t.Fatalf("creating item1: %v", diags)
	}

	item2, diags := types.ObjectValueFrom(ctx, artifactoryItemTypes, artifactoryItemModel{
		ArtifactoryId: types.StringValue("art-2"),
		Name:          types.StringValue("artifactory-2"),
		CreateTime:    types.StringValue(testTime.Format(time.RFC3339)),
		State:         types.StringValue(string(registry.ARTIFACTORYSTATE_ACTIVE)),
		Url:           types.StringValue("https://art-2.cr.stackit.cloud"),
		Proxy:         proxyObj,
	})
	if diags.HasError() {
		t.Fatalf("creating item2: %v", diags)
	}

	expectedList, diags := types.ListValue(types.ObjectType{AttrTypes: artifactoryItemTypes}, []attr.Value{item1, item2})
	if diags.HasError() {
		t.Fatalf("creating expectedList: %v", diags)
	}

	expectedModel := &DataSourceModel{
		Id:            types.StringValue("pid,reg"),
		ProjectId:     types.StringValue("pid"),
		Region:        types.StringValue("reg"),
		Artifactories: expectedList,
	}

	if diff := cmp.Diff(model, expectedModel); diff != "" {
		t.Errorf("mapFields() mismatch (-got +want):\n%s", diff)
	}
}

func TestMapFields_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("nil model", func(t *testing.T) {
		err := mapFields(ctx, []registry.Artifactory{}, nil, "reg")
		if err == nil {
			t.Error("expected error for nil model")
		}
	})

	t.Run("empty artifactories", func(t *testing.T) {
		model := &DataSourceModel{
			ProjectId: types.StringValue("pid"),
		}
		err := mapFields(ctx, []registry.Artifactory{}, model, "reg")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model.Artifactories.IsNull() {
			t.Error("expected artifactories list not to be null")
		}
		if len(model.Artifactories.Elements()) != 0 {
			t.Errorf("expected 0 elements, got %d", len(model.Artifactories.Elements()))
		}
	})

	t.Run("proxy without credentials", func(t *testing.T) {
		artifactories := []registry.Artifactory{
			{
				Id:                 "art-proxy-no-creds",
				Name:               "artifactory-no-creds",
				ConsumedQuotaBytes: 50,
				CreateTime:         testTime,
				State:              registry.ARTIFACTORYSTATE_ACTIVE,
				Url:                "https://art-no-creds.cr.stackit.cloud",
				Proxy: &registry.Proxy{
					RegistryType: "docker-hub",
					Url:          "https://registry-1.docker.io",
				},
			},
		}
		model := &DataSourceModel{
			ProjectId: types.StringValue("pid"),
		}
		err := mapFields(ctx, artifactories, model, "reg")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(model.Artifactories.Elements()) != 1 {
			t.Fatalf("expected 1 element, got %d", len(model.Artifactories.Elements()))
		}
	})
}

func TestArtifactoriesDataSource_Read(t *testing.T) {
	ctx := context.Background()
	dsBase := NewArtifactoriesDataSource()
	var schemaResp datasource.SchemaResponse
	dsBase.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)

	configAttrTypes := map[string]attr.Type{
		"id":            types.StringType,
		"project_id":    types.StringType,
		"region":        types.StringType,
		"artifactories": types.ListType{ElemType: types.ObjectType{AttrTypes: artifactoryItemTypes}},
	}

	configModel := DataSourceModel{
		ProjectId:     types.StringValue("pid"),
		Region:        types.StringValue("reg"),
		Id:            types.StringNull(),
		Artifactories: types.ListNull(types.ObjectType{AttrTypes: artifactoryItemTypes}),
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
			ListArtifactoriesExecuteMock: new(func(_ registry.ApiListArtifactoriesRequest) (*registry.ArtifactoryList, error) {
				return &registry.ArtifactoryList{
					Artifactories: []registry.Artifactory{
						{
							Id:                 "art-1",
							Name:               "artifactory-1",
							ConsumedQuotaBytes: 100,
							CreateTime:         testTime,
							State:              registry.ARTIFACTORYSTATE_ACTIVE,
							Url:                "https://art-1.cr.stackit.cloud",
						},
					},
				}, nil
			}),
		}
		ds := &artifactoriesDataSource{client: mockClient}

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
		if resultModel.Id.ValueString() != "pid,reg" {
			t.Errorf("expected id 'pid,reg', got %q", resultModel.Id.ValueString())
		}
		if len(resultModel.Artifactories.Elements()) != 1 {
			t.Errorf("expected 1 artifactory, got %d", len(resultModel.Artifactories.Elements()))
		}
	})

	t.Run("read API error", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			ListArtifactoriesExecuteMock: new(func(_ registry.ApiListArtifactoriesRequest) (*registry.ArtifactoryList, error) {
				return nil, fmt.Errorf("api error")
			}),
		}
		ds := &artifactoriesDataSource{client: mockClient}

		resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		ds.Read(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics on read API error")
		}
	})

	t.Run("read 404 not found", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			ListArtifactoriesExecuteMock: new(func(_ registry.ApiListArtifactoriesRequest) (*registry.ArtifactoryList, error) {
				return nil, &oapierror.GenericOpenAPIError{StatusCode: 404}
			}),
		}
		ds := &artifactoriesDataSource{client: mockClient}

		resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		ds.Read(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics on read 404")
		}
	})
}
