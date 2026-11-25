package replicationadapters

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	registry "github.com/stackitcloud/stackit-sdk-go/services/registry/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

func TestReplicationAdaptersDataSource_Metadata(t *testing.T) {
	ds := NewReplicationAdaptersDataSource()

	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "stackit"}, &resp)

	if resp.TypeName != "stackit_containerregistry_replication_adapters" {
		t.Errorf("expected type name %q, got %q", "stackit_containerregistry_replication_adapters", resp.TypeName)
	}
}

func TestReplicationAdaptersDataSource_Schema(t *testing.T) {
	ds := NewReplicationAdaptersDataSource()

	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}

	if len(resp.Schema.Attributes) == 0 {
		t.Errorf("expected non-empty schema attributes")
	}
}

func TestReplicationAdaptersDataSource_Configure(t *testing.T) {
	ds := &replicationAdaptersDataSource{}
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

func TestReplicationAdaptersDataSource_Read(t *testing.T) {
	ctx := context.Background()
	dsBase := NewReplicationAdaptersDataSource()

	var schemaResp datasource.SchemaResponse

	dsBase.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)

	configAttrTypes := map[string]attr.Type{
		"id":         types.StringType,
		"project_id": types.StringType,
		"region":     types.StringType,
		"results":    types.ListType{ElemType: types.ObjectType{AttrTypes: replicationAdapterTypes}},
	}

	configModel := DataSourceModel{
		ProjectId: types.StringValue("pid"),
		Region:    types.StringValue("reg"),
		Id:        types.StringNull(),
		Results:   types.ListNull(types.ObjectType{AttrTypes: replicationAdapterTypes}),
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
			ListReplicationAdaptersExecuteMock: new(func(_ registry.ApiListReplicationAdaptersRequest) (*registry.ReplicationAdapterList, error) {
				return &registry.ReplicationAdapterList{
					ReplicationAdapters: []registry.ReplicationAdapter{
						{
							CredentialType: registry.REPLICATIONADAPTERCREDENTIALTYPE_BASIC,
							DisplayName:    "Docker Hub",
							RegistryType:   "docker-hub",
						},
					},
				}, nil
			}),
		}
		ds := &replicationAdaptersDataSource{client: mockClient}

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

		if len(resultModel.Results.Elements()) != 1 {
			t.Errorf("expected 1 replication adapter, got %d", len(resultModel.Results.Elements()))
		}
	})

	t.Run("read API error", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			ListReplicationAdaptersExecuteMock: new(func(_ registry.ApiListReplicationAdaptersRequest) (*registry.ReplicationAdapterList, error) {
				return nil, fmt.Errorf("api error")
			}),
		}
		ds := &replicationAdaptersDataSource{client: mockClient}

		resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		ds.Read(ctx, req, &resp)

		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics on read API error")
		}
	})

	t.Run("read config error", func(t *testing.T) {
		ds := &replicationAdaptersDataSource{}
		invalidReq := datasource.ReadRequest{
			Config: tfsdk.Config{
				Schema: schemaResp.Schema,
				Raw:    tftypes.NewValue(tftypes.String, "invalid"),
			},
		}

		resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		ds.Read(ctx, invalidReq, &resp)

		if !resp.Diagnostics.HasError() {
			t.Error("expected error on invalid config")
		}
	})

	t.Run("read 404 not found", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			ListReplicationAdaptersExecuteMock: new(func(_ registry.ApiListReplicationAdaptersRequest) (*registry.ReplicationAdapterList, error) {
				return nil, &oapierror.GenericOpenAPIError{StatusCode: 404}
			}),
		}
		ds := &replicationAdaptersDataSource{client: mockClient}

		resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		ds.Read(ctx, req, &resp)

		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics on read 404")
		}
	})
}

func TestMapFields(t *testing.T) {
	ctx := context.Background()

	adapters := []registry.ReplicationAdapter{
		{
			CredentialType: registry.REPLICATIONADAPTERCREDENTIALTYPE_BASIC,
			DisplayName:    "Docker Hub",
			RegistryType:   "docker-hub",
		},
		{
			CredentialType: registry.REPLICATIONADAPTERCREDENTIALTYPE_OAUTH,
			DisplayName:    "GitHub Container Registry",
			RegistryType:   "ghcr",
		},
	}

	model := &DataSourceModel{
		ProjectId: types.StringValue("pid"),
	}

	err := mapFields(ctx, adapters, model, "reg")
	if err != nil {
		t.Fatalf("mapFields() error = %v", err)
	}

	if model.Id.ValueString() != "pid,reg" {
		t.Errorf("expected id 'pid,reg', got %q", model.Id.ValueString())
	}

	if model.Region.ValueString() != "reg" {
		t.Errorf("expected region 'reg', got %q", model.Region.ValueString())
	}

	item1, diags := types.ObjectValueFrom(ctx, replicationAdapterTypes, replicationAdapterModel{
		CredentialType: types.StringValue("basic"),
		DisplayName:    types.StringValue("Docker Hub"),
		RegistryType:   types.StringValue("docker-hub"),
	})
	if diags.HasError() {
		t.Fatalf("creating item1: %v", diags)
	}

	item2, diags := types.ObjectValueFrom(ctx, replicationAdapterTypes, replicationAdapterModel{
		CredentialType: types.StringValue("oauth"),
		DisplayName:    types.StringValue("GitHub Container Registry"),
		RegistryType:   types.StringValue("ghcr"),
	})
	if diags.HasError() {
		t.Fatalf("creating item2: %v", diags)
	}

	expectedList, diags := types.ListValue(types.ObjectType{AttrTypes: replicationAdapterTypes}, []attr.Value{item1, item2})
	if diags.HasError() {
		t.Fatalf("creating expectedList: %v", diags)
	}

	expectedModel := &DataSourceModel{
		Id:        types.StringValue("pid,reg"),
		ProjectId: types.StringValue("pid"),
		Region:    types.StringValue("reg"),
		Results:   expectedList,
	}

	if diff := cmp.Diff(model, expectedModel); diff != "" {
		t.Errorf("mapFields() mismatch (-got +want):\n%s", diff)
	}
}

func TestMapFields_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("nil model", func(t *testing.T) {
		err := mapFields(ctx, []registry.ReplicationAdapter{}, nil, "reg")
		if err == nil {
			t.Error("expected error for nil model")
		}
	})

	t.Run("empty adapters", func(t *testing.T) {
		model := &DataSourceModel{
			ProjectId: types.StringValue("pid"),
		}

		err := mapFields(ctx, []registry.ReplicationAdapter{}, model, "reg")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if model.Results.IsNull() {
			t.Error("expected adapters list not to be null")
		}

		if len(model.Results.Elements()) != 0 {
			t.Errorf("expected 0 elements, got %d", len(model.Results.Elements()))
		}
	})
}

func TestReplicationAdapterPageRequest_Methods(t *testing.T) {
	req := &replicationAdapterPageRequest{}
	req = req.PageSize(10)
	req = req.PageToken("token")

	if req == nil {
		t.Fatal("expected non-nil request")
	}

	respNil := &replicationAdapterPageResponse{resp: nil}
	if items := respNil.GetItems(); items != nil {
		t.Errorf("expected nil items, got %v", items)
	}

	if tok := respNil.GetNextPageToken(); tok != "" {
		t.Errorf("expected empty token, got %q", tok)
	}

	token := "tok"
	respNonNil := &replicationAdapterPageResponse{
		resp: &registry.ReplicationAdapterList{
			ReplicationAdapters: []registry.ReplicationAdapter{{RegistryType: "docker-hub"}},
			NextPageToken:       &token,
		},
	}

	if items := respNonNil.GetItems(); len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	if tok := respNonNil.GetNextPageToken(); tok != "tok" {
		t.Errorf("expected 'tok', got %q", tok)
	}
}
