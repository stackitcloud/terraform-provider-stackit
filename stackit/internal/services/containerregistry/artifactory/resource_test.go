package artifactory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	registry "github.com/stackitcloud/stackit-sdk-go/services/registry/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

var testTime = time.Now()

func fixtureArtifactory(mods ...func(art *registry.Artifactory)) *registry.Artifactory {
	art := &registry.Artifactory{
		Id:                 "art-id-123",
		Name:               "my-artifactory",
		ConsumedQuotaBytes: 1024,
		CreateTime:         testTime,
		State:              registry.ARTIFACTORYSTATE_ACTIVE,
		Url:                "https://art-id-123.cr.stackit.cloud",
	}
	for _, mod := range mods {
		mod(art)
	}
	return art
}

func fixtureModel(t *testing.T, mods ...func(model *Model)) *Model {
	t.Helper()
	model := &Model{
		Id:            types.StringValue("pid,reg,art-id-123"),
		ProjectId:     types.StringValue("pid"),
		Region:        types.StringValue("reg"),
		ArtifactoryId: types.StringValue("art-id-123"),
		Name:          types.StringValue("my-artifactory"),
		CreateTime:    types.StringValue(testTime.Format(time.RFC3339)),
		State:         types.StringValue(string(registry.ARTIFACTORYSTATE_ACTIVE)),
		Url:           types.StringValue("https://art-id-123.cr.stackit.cloud"),
		Proxy:         types.ObjectNull(proxyTypes),
	}
	for _, mod := range mods {
		mod(model)
	}
	return model
}

func TestMapFields(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		description string
		input       *registry.Artifactory
		expected    *Model
		wantErr     bool
	}{
		{
			description: "min values (no proxy)",
			input:       fixtureArtifactory(),
			expected:    fixtureModel(t),
		},
		{
			description: "max values (with proxy and credentials)",
			input: fixtureArtifactory(func(art *registry.Artifactory) {
				desc := "Docker Hub mirror"
				art.Proxy = &registry.Proxy{
					RegistryType: "docker-hub",
					Url:          "https://registry-1.docker.io",
					Description:  &desc,
					Credentials: &registry.RedactedCredentials{
						AccessKey: "test-user",
					},
				}
			}),
			expected: fixtureModel(t, func(model *Model) {
				credsObj, diags := types.ObjectValueFrom(ctx, credentialsTypes, credentialsModel{
					AccessKey:    types.StringValue("test-user"),
					AccessSecret: types.StringNull(),
				})
				if diags.HasError() {
					t.Fatalf("unexpected diags creating creds: %v", diags)
				}
				proxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, proxyModel{
					RegistryType: types.StringValue("docker-hub"),
					Url:          types.StringValue("https://registry-1.docker.io"),
					Description:  types.StringValue("Docker Hub mirror"),
					Credentials:  credsObj,
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
			model := &Model{
				ProjectId: types.StringValue("pid"),
			}
			err := mapFields(ctx, tt.input, model, "reg")
			if (err != nil) != tt.wantErr {
				t.Fatalf("mapFields() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(model, tt.expected); diff != "" {
				t.Errorf("mapFields() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	ctx := context.Background()

	t.Run("without proxy", func(t *testing.T) {
		model := &Model{
			Name:  types.StringValue("my-artifactory"),
			Proxy: types.ObjectNull(proxyTypes),
		}

		payload, err := toCreatePayload(ctx, model, model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payload.Name != "my-artifactory" {
			t.Errorf("expected name %q, got %q", "my-artifactory", payload.Name)
		}
		if payload.Proxy != nil {
			t.Errorf("expected proxy to be nil, got %v", payload.Proxy)
		}
	})

	t.Run("with proxy and credentials", func(t *testing.T) {
		credsObj, diags := types.ObjectValueFrom(ctx, credentialsTypes, credentialsModel{
			AccessKey:    types.StringValue("user"),
			AccessSecret: types.StringValue("secret"),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}

		proxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, proxyModel{
			RegistryType: types.StringValue("docker-hub"),
			Url:          types.StringValue("https://registry-1.docker.io"),
			Description:  types.StringValue("Docker Hub mirror"),
			Credentials:  credsObj,
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}

		model := &Model{
			Name:  types.StringValue("my-artifactory"),
			Proxy: proxyObj,
		}

		payload, err := toCreatePayload(ctx, model, model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payload.Proxy == nil {
			t.Fatal("expected proxy to not be nil")
		}
		if payload.Proxy.RegistryType != "docker-hub" {
			t.Errorf("expected registry_type docker-hub, got %q", payload.Proxy.RegistryType)
		}
		if payload.Proxy.Url != "https://registry-1.docker.io" {
			t.Errorf("expected url https://registry-1.docker.io, got %q", payload.Proxy.Url)
		}
		if payload.Proxy.Description == nil || *payload.Proxy.Description != "Docker Hub mirror" {
			t.Errorf("expected description 'Docker Hub mirror', got %v", payload.Proxy.Description)
		}
		if payload.Proxy.Credentials == nil || payload.Proxy.Credentials.AccessKey != "user" || payload.Proxy.Credentials.AccessSecret != "secret" {
			t.Errorf("expected credentials user/secret, got %v", payload.Proxy.Credentials)
		}
	})

	t.Run("with proxy and write-only credentials", func(t *testing.T) {
		credsObj, diags := types.ObjectValueFrom(ctx, credentialsTypes, credentialsModel{
			AccessKey:             types.StringValue("user"),
			AccessSecret:          types.StringNull(),
			AccessSecretWo:        types.StringValue("wo-secret"),
			AccessSecretWoVersion: types.Int64Value(1),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}

		proxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, proxyModel{
			RegistryType: types.StringValue("docker-hub"),
			Url:          types.StringValue("https://registry-1.docker.io"),
			Description:  types.StringValue("Docker Hub mirror"),
			Credentials:  credsObj,
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}

		model := &Model{
			Name:  types.StringValue("my-artifactory"),
			Proxy: proxyObj,
		}

		payload, err := toCreatePayload(ctx, model, model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payload.Proxy == nil || payload.Proxy.Credentials == nil || payload.Proxy.Credentials.AccessSecret != "wo-secret" {
			t.Errorf("expected access_secret 'wo-secret', got %v", payload.Proxy)
		}
	})
}

func TestToUpdatePayload(t *testing.T) {
	ctx := context.Background()

	t.Run("without proxy", func(t *testing.T) {
		model := &Model{
			Proxy: types.ObjectNull(proxyTypes),
		}

		payload, err := toUpdatePayload(ctx, model, nil, model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payload.Proxy != nil {
			t.Errorf("expected proxy to be nil, got %v", payload.Proxy)
		}
	})

	t.Run("with proxy and credentials", func(t *testing.T) {
		credsObj, diags := types.ObjectValueFrom(ctx, credentialsTypes, credentialsModel{
			AccessKey:    types.StringValue("updated-user"),
			AccessSecret: types.StringValue("updated-secret"),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}

		proxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, proxyModel{
			RegistryType: types.StringValue("docker-hub"),
			Url:          types.StringValue("https://registry-2.docker.io"),
			Description:  types.StringValue("Updated description"),
			Credentials:  credsObj,
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}

		model := &Model{
			Proxy: proxyObj,
		}

		payload, err := toUpdatePayload(ctx, model, nil, model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payload.Proxy == nil {
			t.Fatal("expected proxy to not be nil")
		}
		if payload.Proxy.Url == nil || *payload.Proxy.Url != "https://registry-2.docker.io" {
			t.Errorf("expected url https://registry-2.docker.io, got %v", payload.Proxy.Url)
		}
		if payload.Proxy.Description == nil || *payload.Proxy.Description != "Updated description" {
			t.Errorf("expected description 'Updated description', got %v", payload.Proxy.Description)
		}
		if payload.Proxy.Credentials == nil || payload.Proxy.Credentials.AccessKey != "updated-user" || payload.Proxy.Credentials.AccessSecret != "updated-secret" {
			t.Errorf("expected credentials updated-user/updated-secret, got %v", payload.Proxy.Credentials)
		}
	})

	t.Run("with proxy, stateModel, and configModel write-only rotation", func(t *testing.T) {
		// state: old version 1
		stateCredsObj, diags := types.ObjectValueFrom(ctx, credentialsTypes, credentialsModel{
			AccessKey:             types.StringValue("user"),
			AccessSecret:          types.StringNull(),
			AccessSecretWo:        types.StringNull(),
			AccessSecretWoVersion: types.Int64Value(1),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		stateProxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, proxyModel{
			RegistryType: types.StringValue("docker-hub"),
			Url:          types.StringValue("https://registry-1.docker.io"),
			Credentials:  stateCredsObj,
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		stateModel := &Model{Proxy: stateProxyObj}

		// plan: bumped version 2
		planCredsObj, diags := types.ObjectValueFrom(ctx, credentialsTypes, credentialsModel{
			AccessKey:             types.StringValue("user"),
			AccessSecret:          types.StringNull(),
			AccessSecretWo:        types.StringNull(),
			AccessSecretWoVersion: types.Int64Value(2),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		planProxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, proxyModel{
			RegistryType: types.StringValue("docker-hub"),
			Url:          types.StringValue("https://registry-1.docker.io"),
			Credentials:  planCredsObj,
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		planModel := &Model{Proxy: planProxyObj}

		// config: write-only secret provided
		configCredsObj, diags := types.ObjectValueFrom(ctx, credentialsTypes, credentialsModel{
			AccessKey:             types.StringValue("user"),
			AccessSecret:          types.StringNull(),
			AccessSecretWo:        types.StringValue("new-wo-secret"),
			AccessSecretWoVersion: types.Int64Value(2),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		configProxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, proxyModel{
			RegistryType: types.StringValue("docker-hub"),
			Url:          types.StringValue("https://registry-1.docker.io"),
			Credentials:  configCredsObj,
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		configModel := &Model{Proxy: configProxyObj}

		payload, err := toUpdatePayload(ctx, planModel, stateModel, configModel)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payload.Proxy == nil || payload.Proxy.Credentials == nil || payload.Proxy.Credentials.AccessSecret != "new-wo-secret" {
			t.Errorf("expected access_secret 'new-wo-secret', got %v", payload.Proxy)
		}
	})
}

func TestNewArtifactoryResource(t *testing.T) {
	r := NewArtifactoryResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}
}

func TestArtifactoryResource_ModifyPlan(t *testing.T) {
	ctx := context.Background()
	r := &artifactoryResource{
		providerData: core.ProviderData{},
	}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	t.Run("raw is null", func(t *testing.T) {
		req := resource.ModifyPlanRequest{
			Config: tfsdk.Config{
				Raw: tftypes.NewValue(tftypes.String, nil),
			},
		}
		var resp resource.ModifyPlanResponse
		r.ModifyPlan(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
	})

	t.Run("adapt region from provider data", func(t *testing.T) {
		rWithRegion := &artifactoryResource{
			providerData: core.ProviderData{
				DefaultRegion: "eu01",
			},
		}
		configModel := &Model{
			ProjectId: types.StringValue("pid"),
			Region:    types.StringNull(),
			Name:      types.StringValue("name"),
			Proxy:     types.ObjectNull(proxyTypes),
		}
		configVal, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"id":             types.StringType,
			"project_id":     types.StringType,
			"region":         types.StringType,
			"artifactory_id": types.StringType,
			"name":           types.StringType,
			"create_time":    types.StringType,
			"state":          types.StringType,
			"url":            types.StringType,
			"proxy":          types.ObjectType{AttrTypes: proxyTypes},
		}, configModel)
		rawVal, _ := configVal.ToTerraformValue(ctx)

		planModel := &Model{
			ProjectId: types.StringValue("pid"),
			Region:    types.StringNull(),
			Name:      types.StringValue("name"),
			Proxy:     types.ObjectNull(proxyTypes),
		}
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		_ = plan.Set(ctx, planModel)

		req := resource.ModifyPlanRequest{
			Config: tfsdk.Config{
				Schema: schemaResp.Schema,
				Raw:    rawVal,
			},
			Plan: plan,
		}
		resp := resource.ModifyPlanResponse{
			Plan: plan,
		}
		rWithRegion.ModifyPlan(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}

		var updatedPlan Model
		_ = resp.Plan.Get(ctx, &updatedPlan)
		if updatedPlan.Region.ValueString() != "eu01" {
			t.Errorf("expected adapted region 'eu01', got %q", updatedPlan.Region.ValueString())
		}
	})
}

func TestArtifactoryResource_Metadata(t *testing.T) {
	r := NewArtifactoryResource()
	var resp resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "stackit"}, &resp)
	if resp.TypeName != "stackit_containerregistry_artifactory" {
		t.Errorf("expected type name %q, got %q", "stackit_containerregistry_artifactory", resp.TypeName)
	}
}

func TestArtifactoryResource_Schema(t *testing.T) {
	r := NewArtifactoryResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if len(resp.Schema.Attributes) == 0 {
		t.Errorf("expected non-empty schema attributes")
	}
}

func TestArtifactoryResource_Configure(t *testing.T) {
	r := &artifactoryResource{}
	ctx := context.Background()

	t.Run("valid provider data", func(t *testing.T) {
		providerData, err := core.NewProviderDataInternal(core.ProviderData{}, &core.MockClientFactory{})
		if err != nil {
			t.Fatalf("unexpected error creating provider data: %v", err)
		}
		var resp resource.ConfigureResponse
		r.Configure(ctx, resource.ConfigureRequest{ProviderData: providerData}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("unexpected error diagnostics: %v", resp.Diagnostics)
		}
		if r.client == nil {
			t.Error("expected client to be configured")
		}
	})

	t.Run("nil provider data", func(t *testing.T) {
		var resp resource.ConfigureResponse
		r.Configure(ctx, resource.ConfigureRequest{ProviderData: nil}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("expected no error diagnostics on nil provider data, got %v", resp.Diagnostics)
		}
	})

	t.Run("invalid provider data type", func(t *testing.T) {
		var resp resource.ConfigureResponse
		r.Configure(ctx, resource.ConfigureRequest{ProviderData: "invalid"}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error diagnostics on invalid provider data")
		}
	})
}

func TestMapFields_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("nil artifactory", func(t *testing.T) {
		model := &Model{}
		err := mapFields(ctx, nil, model, "reg")
		if err == nil {
			t.Error("expected error for nil artifactory")
		}
	})

	t.Run("nil model", func(t *testing.T) {
		art := fixtureArtifactory()
		err := mapFields(ctx, art, nil, "reg")
		if err == nil {
			t.Error("expected error for nil model")
		}
	})

	t.Run("proxy without credentials", func(t *testing.T) {
		art := fixtureArtifactory(func(a *registry.Artifactory) {
			a.Proxy = &registry.Proxy{
				RegistryType: "docker-hub",
				Url:          "https://registry-1.docker.io",
			}
		})
		model := &Model{
			ProjectId: types.StringValue("pid"),
		}
		err := mapFields(ctx, art, model, "reg")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model.Proxy.IsNull() {
			t.Error("expected proxy to not be null")
		}
	})

	t.Run("existing model with proxy and credentials preserves version", func(t *testing.T) {
		credsObj, diags := types.ObjectValueFrom(ctx, credentialsTypes, credentialsModel{
			AccessKey:             types.StringValue("initial-user"),
			AccessSecret:          types.StringNull(),
			AccessSecretWo:        types.StringNull(),
			AccessSecretWoVersion: types.Int64Value(42),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		proxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, proxyModel{
			RegistryType: types.StringValue("docker-hub"),
			Url:          types.StringValue("https://registry-1.docker.io"),
			Description:  types.StringValue("desc"),
			Credentials:  credsObj,
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		model := &Model{
			ProjectId: types.StringValue("pid"),
			Proxy:     proxyObj,
		}

		art := fixtureArtifactory(func(a *registry.Artifactory) {
			desc := "updated desc"
			a.Proxy = &registry.Proxy{
				RegistryType: "docker-hub",
				Url:          "https://registry-1.docker.io",
				Description:  &desc,
				Credentials: &registry.RedactedCredentials{
					AccessKey: "new-user",
				},
			}
		})

		err := mapFields(ctx, art, model, "reg")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model.Proxy.IsNull() {
			t.Fatal("expected proxy not to be null")
		}

		var p proxyModel
		diags = model.Proxy.As(ctx, &p, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			t.Fatalf("unexpected diags converting proxy: %v", diags)
		}
		var creds credentialsModel
		diags = p.Credentials.As(ctx, &creds, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			t.Fatalf("unexpected diags converting creds: %v", diags)
		}
		if creds.AccessKey.ValueString() != "new-user" {
			t.Errorf("expected access_key 'new-user', got %q", creds.AccessKey.ValueString())
		}
		if creds.AccessSecretWoVersion.ValueInt64() != 42 {
			t.Errorf("expected access_secret_wo_version 42, got %v", creds.AccessSecretWoVersion.ValueInt64())
		}
	})

	t.Run("artifactory id not present", func(t *testing.T) {
		art := &registry.Artifactory{}
		model := &Model{
			ProjectId: types.StringValue("pid"),
		}
		err := mapFields(ctx, art, model, "reg")
		if err == nil {
			t.Error("expected error when artifactory id is missing")
		}
	})
}

func TestToPayloads_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("toCreatePayload nil model", func(t *testing.T) {
		_, err := toCreatePayload(ctx, nil, nil)
		if err == nil {
			t.Error("expected error for nil model")
		}
	})

	t.Run("toCreatePayload proxy without credentials", func(t *testing.T) {
		proxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, proxyModel{
			RegistryType: types.StringValue("docker-hub"),
			Url:          types.StringValue("https://registry-1.docker.io"),
			Credentials:  types.ObjectNull(credentialsTypes),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		model := &Model{
			Name:  types.StringValue("my-artifactory"),
			Proxy: proxyObj,
		}
		payload, err := toCreatePayload(ctx, model, model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payload.Proxy == nil {
			t.Fatal("expected proxy not nil")
		}
		if payload.Proxy.Credentials != nil {
			t.Errorf("expected credentials to be nil, got %v", payload.Proxy.Credentials)
		}
	})

	t.Run("toUpdatePayload nil model", func(t *testing.T) {
		_, err := toUpdatePayload(ctx, nil, nil, nil)
		if err == nil {
			t.Error("expected error for nil model")
		}
	})

	t.Run("toUpdatePayload proxy without credentials", func(t *testing.T) {
		proxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, proxyModel{
			RegistryType: types.StringValue("docker-hub"),
			Url:          types.StringValue("https://registry-2.docker.io"),
			Credentials:  types.ObjectNull(credentialsTypes),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		model := &Model{
			Proxy: proxyObj,
		}
		payload, err := toUpdatePayload(ctx, model, nil, model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payload.Proxy == nil {
			t.Fatal("expected proxy not nil")
		}
		if payload.Proxy.Credentials != nil {
			t.Errorf("expected credentials to be nil, got %v", payload.Proxy.Credentials)
		}
	})

	t.Run("toCreatePayload credentials without secret returns error", func(t *testing.T) {
		credsObj, diags := types.ObjectValueFrom(ctx, credentialsTypes, credentialsModel{
			AccessKey:    types.StringValue("user"),
			AccessSecret: types.StringNull(),
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		proxyObj, diags := types.ObjectValueFrom(ctx, proxyTypes, proxyModel{
			RegistryType: types.StringValue("docker-hub"),
			Url:          types.StringValue("https://registry-1.docker.io"),
			Credentials:  credsObj,
		})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		model := &Model{
			Name:  types.StringValue("my-artifactory"),
			Proxy: proxyObj,
		}
		_, err := toCreatePayload(ctx, model, model)
		if err == nil {
			t.Error("expected error when neither access_secret nor access_secret_wo is set")
		}
	})

	t.Run("getAccessSecret edge cases", func(t *testing.T) {
		if s := getAccessSecret(nil, nil, nil, false); s != nil {
			t.Errorf("expected nil for nil planCreds, got %v", s)
		}
		creds := &credentialsModel{
			AccessSecret:          types.StringNull(),
			AccessSecretWo:        types.StringValue("secret"),
			AccessSecretWoVersion: types.Int64Value(1),
		}
		stateCreds := &credentialsModel{
			AccessSecretWoVersion: types.Int64Value(1),
		}
		if s := getAccessSecret(creds, stateCreds, creds, true); s != nil {
			t.Errorf("expected nil when version unchanged on update, got %v", s)
		}
	})
}

func TestArtifactoryResource_ImportState(t *testing.T) {
	r, ok := NewArtifactoryResource().(resource.ResourceWithImportState)
	if !ok {
		t.Fatalf("expected resource to implement resource.ResourceWithImportState")
	}
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	t.Run("valid import id", func(t *testing.T) {
		req := resource.ImportStateRequest{
			ID: "pid,reg,art-id-123",
		}
		state := tfsdk.State{Schema: schemaResp.Schema}
		diags := state.Set(ctx, &Model{Proxy: types.ObjectNull(proxyTypes)})
		if diags.HasError() {
			t.Fatalf("unexpected diags initializing state: %v", diags)
		}
		resp := resource.ImportStateResponse{
			State: state,
		}
		r.ImportState(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		var model Model
		diags = resp.State.Get(ctx, &model)
		if diags.HasError() {
			t.Fatalf("unexpected diags getting state: %v", diags)
		}
		if model.ProjectId.ValueString() != "pid" {
			t.Errorf("expected project_id 'pid', got %q", model.ProjectId.ValueString())
		}
		if model.Region.ValueString() != "reg" {
			t.Errorf("expected region 'reg', got %q", model.Region.ValueString())
		}
		if model.ArtifactoryId.ValueString() != "art-id-123" {
			t.Errorf("expected artifactory_id 'art-id-123', got %q", model.ArtifactoryId.ValueString())
		}
	})

	t.Run("invalid import id - missing parts", func(t *testing.T) {
		req := resource.ImportStateRequest{
			ID: "pid,reg",
		}
		var resp resource.ImportStateResponse
		r.ImportState(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error for missing parts")
		}
	})

	t.Run("invalid import id - empty parts", func(t *testing.T) {
		req := resource.ImportStateRequest{
			ID: "pid,,art-id-123",
		}
		var resp resource.ImportStateResponse
		r.ImportState(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error for empty part")
		}
	})
}

func TestArtifactoryResource_Read(t *testing.T) {
	ctx := context.Background()
	rBase := NewArtifactoryResource()
	var schemaResp resource.SchemaResponse
	rBase.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	t.Run("read success", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				return fixtureArtifactory(), nil
			}),
		}
		r := &artifactoryResource{client: mockClient}
		initialModel := &Model{
			ProjectId:     types.StringValue("pid"),
			Region:        types.StringValue("reg"),
			ArtifactoryId: types.StringValue("art-id-123"),
			Proxy:         types.ObjectNull(proxyTypes),
		}
		state := tfsdk.State{Schema: schemaResp.Schema}
		diags := state.Set(ctx, initialModel)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}

		req := resource.ReadRequest{State: state}
		resp := resource.ReadResponse{State: state}
		r.Read(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
	})

	t.Run("read 404 not found removes resource", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				return nil, &oapierror.GenericOpenAPIError{StatusCode: 404}
			}),
		}
		r := &artifactoryResource{client: mockClient}
		initialModel := &Model{
			ProjectId:     types.StringValue("pid"),
			Region:        types.StringValue("reg"),
			ArtifactoryId: types.StringValue("art-id-123"),
			Proxy:         types.ObjectNull(proxyTypes),
		}
		state := tfsdk.State{Schema: schemaResp.Schema}
		diags := state.Set(ctx, initialModel)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}

		req := resource.ReadRequest{State: state}
		resp := resource.ReadResponse{State: state}
		r.Read(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
	})

	t.Run("read deleted state removes resource", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				art := fixtureArtifactory()
				art.State = registry.ARTIFACTORYSTATE_DELETED
				return art, nil
			}),
		}
		r := &artifactoryResource{client: mockClient}
		initialModel := &Model{
			ProjectId:     types.StringValue("pid"),
			Region:        types.StringValue("reg"),
			ArtifactoryId: types.StringValue("art-id-123"),
			Proxy:         types.ObjectNull(proxyTypes),
		}
		state := tfsdk.State{Schema: schemaResp.Schema}
		diags := state.Set(ctx, initialModel)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}

		req := resource.ReadRequest{State: state}
		resp := resource.ReadResponse{State: state}
		r.Read(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
	})

	t.Run("read API error", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				return nil, fmt.Errorf("api error")
			}),
		}
		r := &artifactoryResource{client: mockClient}
		initialModel := &Model{
			ProjectId:     types.StringValue("pid"),
			Region:        types.StringValue("reg"),
			ArtifactoryId: types.StringValue("art-id-123"),
			Proxy:         types.ObjectNull(proxyTypes),
		}
		state := tfsdk.State{Schema: schemaResp.Schema}
		diags := state.Set(ctx, initialModel)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}

		req := resource.ReadRequest{State: state}
		resp := resource.ReadResponse{State: state}
		r.Read(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics on read API error")
		}
	})
}

func TestArtifactoryResource_Create(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	rBase := NewArtifactoryResource()
	rBase.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	t.Run("create success", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			CreateArtifactoryExecuteMock: new(func(_ registry.ApiCreateArtifactoryRequest) (*registry.Artifactory, error) {
				return fixtureArtifactory(), nil
			}),
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				return fixtureArtifactory(), nil
			}),
		}
		r := &artifactoryResource{client: mockClient}
		planModel := &Model{
			ProjectId: types.StringValue("pid"),
			Region:    types.StringValue("reg"),
			Name:      types.StringValue("my-artifactory"),
			Proxy:     types.ObjectNull(proxyTypes),
		}
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		_ = plan.Set(ctx, planModel)
		configVal, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"id":             types.StringType,
			"project_id":     types.StringType,
			"region":         types.StringType,
			"artifactory_id": types.StringType,
			"name":           types.StringType,
			"create_time":    types.StringType,
			"state":          types.StringType,
			"url":            types.StringType,
			"proxy":          types.ObjectType{AttrTypes: proxyTypes},
		}, planModel)
		rawVal, _ := configVal.ToTerraformValue(ctx)
		req := resource.CreateRequest{
			Plan: plan,
			Config: tfsdk.Config{
				Schema: schemaResp.Schema,
				Raw:    rawVal,
			},
		}
		var resp resource.CreateResponse
		resp.State = tfsdk.State{Schema: schemaResp.Schema}
		_ = resp.State.Set(ctx, &Model{Proxy: types.ObjectNull(proxyTypes)})

		r.Create(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics on create success: %v", resp.Diagnostics)
		}

		var resultModel Model
		diags := resp.State.Get(ctx, &resultModel)
		if diags.HasError() {
			t.Fatalf("unexpected diags getting state: %v", diags)
		}
		if resultModel.Id.ValueString() != "pid,reg,art-id-123" {
			t.Errorf("expected id 'pid,reg,art-id-123', got %q", resultModel.Id.ValueString())
		}
		if resultModel.ArtifactoryId.ValueString() != "art-id-123" {
			t.Errorf("expected artifactory_id 'art-id-123', got %q", resultModel.ArtifactoryId.ValueString())
		}
	})

	t.Run("create response missing ID", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			CreateArtifactoryExecuteMock: new(func(_ registry.ApiCreateArtifactoryRequest) (*registry.Artifactory, error) {
				return &registry.Artifactory{}, nil
			}),
		}
		r := &artifactoryResource{client: mockClient}
		planModel := &Model{
			ProjectId: types.StringValue("pid"),
			Region:    types.StringValue("reg"),
			Name:      types.StringValue("my-artifactory"),
			Proxy:     types.ObjectNull(proxyTypes),
		}
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		_ = plan.Set(ctx, planModel)
		configVal, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"id":             types.StringType,
			"project_id":     types.StringType,
			"region":         types.StringType,
			"artifactory_id": types.StringType,
			"name":           types.StringType,
			"create_time":    types.StringType,
			"state":          types.StringType,
			"url":            types.StringType,
			"proxy":          types.ObjectType{AttrTypes: proxyTypes},
		}, planModel)
		rawVal, _ := configVal.ToTerraformValue(ctx)
		req := resource.CreateRequest{
			Plan: plan,
			Config: tfsdk.Config{
				Schema: schemaResp.Schema,
				Raw:    rawVal,
			},
		}
		var resp resource.CreateResponse
		resp.State = tfsdk.State{Schema: schemaResp.Schema}
		_ = resp.State.Set(ctx, &Model{Proxy: types.ObjectNull(proxyTypes)})

		r.Create(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error diagnostics when create response ID is missing")
		}
	})

	t.Run("create wait error", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			CreateArtifactoryExecuteMock: new(func(_ registry.ApiCreateArtifactoryRequest) (*registry.Artifactory, error) {
				return fixtureArtifactory(), nil
			}),
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				return nil, fmt.Errorf("polling error")
			}),
		}
		r := &artifactoryResource{client: mockClient}
		planModel := &Model{
			ProjectId: types.StringValue("pid"),
			Region:    types.StringValue("reg"),
			Name:      types.StringValue("my-artifactory"),
			Proxy:     types.ObjectNull(proxyTypes),
		}
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		_ = plan.Set(ctx, planModel)
		configVal, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"id":             types.StringType,
			"project_id":     types.StringType,
			"region":         types.StringType,
			"artifactory_id": types.StringType,
			"name":           types.StringType,
			"create_time":    types.StringType,
			"state":          types.StringType,
			"url":            types.StringType,
			"proxy":          types.ObjectType{AttrTypes: proxyTypes},
		}, planModel)
		rawVal, _ := configVal.ToTerraformValue(ctx)
		req := resource.CreateRequest{
			Plan: plan,
			Config: tfsdk.Config{
				Schema: schemaResp.Schema,
				Raw:    rawVal,
			},
		}
		var resp resource.CreateResponse
		resp.State = tfsdk.State{Schema: schemaResp.Schema}
		_ = resp.State.Set(ctx, &Model{Proxy: types.ObjectNull(proxyTypes)})

		r.Create(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error diagnostics when waiter fails")
		}
	})

	t.Run("create API error", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			CreateArtifactoryExecuteMock: new(func(_ registry.ApiCreateArtifactoryRequest) (*registry.Artifactory, error) {
				return nil, fmt.Errorf("api create error")
			}),
		}
		r := &artifactoryResource{client: mockClient}
		planModel := &Model{
			ProjectId: types.StringValue("pid"),
			Region:    types.StringValue("reg"),
			Name:      types.StringValue("test-art"),
			Proxy:     types.ObjectNull(proxyTypes),
		}
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		_ = plan.Set(ctx, planModel)
		configVal, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"id":             types.StringType,
			"project_id":     types.StringType,
			"region":         types.StringType,
			"artifactory_id": types.StringType,
			"name":           types.StringType,
			"create_time":    types.StringType,
			"state":          types.StringType,
			"url":            types.StringType,
			"proxy":          types.ObjectType{AttrTypes: proxyTypes},
		}, planModel)
		rawVal, _ := configVal.ToTerraformValue(ctx)
		req := resource.CreateRequest{
			Plan: plan,
			Config: tfsdk.Config{
				Schema: schemaResp.Schema,
				Raw:    rawVal,
			},
		}
		var resp resource.CreateResponse
		resp.State = tfsdk.State{Schema: schemaResp.Schema}
		_ = resp.State.Set(ctx, &Model{Proxy: types.ObjectNull(proxyTypes)})
		r.Create(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error diagnostics on create API failure")
		}
	})
}

func TestArtifactoryResource_Update(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	rBase := NewArtifactoryResource()
	rBase.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	t.Run("update success", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			PatchArtifactoryExecuteMock: new(func(_ registry.ApiPatchArtifactoryRequest) (*registry.Artifactory, error) {
				return fixtureArtifactory(), nil
			}),
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				return fixtureArtifactory(), nil
			}),
		}
		r := &artifactoryResource{client: mockClient}
		planModel := &Model{
			ProjectId:     types.StringValue("pid"),
			Region:        types.StringValue("reg"),
			ArtifactoryId: types.StringValue("art-id-123"),
			Name:          types.StringValue("my-artifactory"),
			Proxy:         types.ObjectNull(proxyTypes),
		}
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		_ = plan.Set(ctx, planModel)
		state := tfsdk.State{Schema: schemaResp.Schema}
		_ = state.Set(ctx, planModel)
		configVal, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"id":             types.StringType,
			"project_id":     types.StringType,
			"region":         types.StringType,
			"artifactory_id": types.StringType,
			"name":           types.StringType,
			"create_time":    types.StringType,
			"state":          types.StringType,
			"url":            types.StringType,
			"proxy":          types.ObjectType{AttrTypes: proxyTypes},
		}, planModel)
		rawVal, _ := configVal.ToTerraformValue(ctx)
		req := resource.UpdateRequest{
			Plan:  plan,
			State: state,
			Config: tfsdk.Config{
				Schema: schemaResp.Schema,
				Raw:    rawVal,
			},
		}
		var resp resource.UpdateResponse
		resp.State = tfsdk.State{Schema: schemaResp.Schema}
		_ = resp.State.Set(ctx, &Model{Proxy: types.ObjectNull(proxyTypes)})
		r.Update(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics on update success: %v", resp.Diagnostics)
		}
	})

	t.Run("update API error", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			PatchArtifactoryExecuteMock: new(func(_ registry.ApiPatchArtifactoryRequest) (*registry.Artifactory, error) {
				return nil, fmt.Errorf("api patch error")
			}),
		}
		r := &artifactoryResource{client: mockClient}
		planModel := &Model{
			ProjectId:     types.StringValue("pid"),
			Region:        types.StringValue("reg"),
			ArtifactoryId: types.StringValue("art-id-123"),
			Name:          types.StringValue("test-art"),
			Proxy:         types.ObjectNull(proxyTypes),
		}
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		_ = plan.Set(ctx, planModel)
		state := tfsdk.State{Schema: schemaResp.Schema}
		_ = state.Set(ctx, planModel)
		configVal, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"id":             types.StringType,
			"project_id":     types.StringType,
			"region":         types.StringType,
			"artifactory_id": types.StringType,
			"name":           types.StringType,
			"create_time":    types.StringType,
			"state":          types.StringType,
			"url":            types.StringType,
			"proxy":          types.ObjectType{AttrTypes: proxyTypes},
		}, planModel)
		rawVal, _ := configVal.ToTerraformValue(ctx)
		req := resource.UpdateRequest{
			Plan:  plan,
			State: state,
			Config: tfsdk.Config{
				Schema: schemaResp.Schema,
				Raw:    rawVal,
			},
		}
		var resp resource.UpdateResponse
		resp.State = tfsdk.State{Schema: schemaResp.Schema}
		_ = resp.State.Set(ctx, &Model{Proxy: types.ObjectNull(proxyTypes)})
		r.Update(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error diagnostics on update API failure")
		}
	})

	t.Run("update wait error", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			PatchArtifactoryExecuteMock: new(func(_ registry.ApiPatchArtifactoryRequest) (*registry.Artifactory, error) {
				return fixtureArtifactory(), nil
			}),
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				return nil, fmt.Errorf("wait polling error")
			}),
		}
		r := &artifactoryResource{client: mockClient}
		planModel := &Model{
			ProjectId:     types.StringValue("pid"),
			Region:        types.StringValue("reg"),
			ArtifactoryId: types.StringValue("art-id-123"),
			Name:          types.StringValue("my-artifactory"),
			Proxy:         types.ObjectNull(proxyTypes),
		}
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		_ = plan.Set(ctx, planModel)
		state := tfsdk.State{Schema: schemaResp.Schema}
		_ = state.Set(ctx, planModel)
		configVal, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"id":             types.StringType,
			"project_id":     types.StringType,
			"region":         types.StringType,
			"artifactory_id": types.StringType,
			"name":           types.StringType,
			"create_time":    types.StringType,
			"state":          types.StringType,
			"url":            types.StringType,
			"proxy":          types.ObjectType{AttrTypes: proxyTypes},
		}, planModel)
		rawVal, _ := configVal.ToTerraformValue(ctx)
		req := resource.UpdateRequest{
			Plan:  plan,
			State: state,
			Config: tfsdk.Config{
				Schema: schemaResp.Schema,
				Raw:    rawVal,
			},
		}
		var resp resource.UpdateResponse
		resp.State = tfsdk.State{Schema: schemaResp.Schema}
		_ = resp.State.Set(ctx, &Model{Proxy: types.ObjectNull(proxyTypes)})
		r.Update(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error diagnostics on update wait error")
		}
	})
}

func TestArtifactoryResource_Delete(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	rBase := NewArtifactoryResource()
	rBase.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	t.Run("delete success", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			DeleteArtifactoryExecuteMock: new(func(_ registry.ApiDeleteArtifactoryRequest) error {
				return nil
			}),
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				return nil, &oapierror.GenericOpenAPIError{StatusCode: 404}
			}),
		}
		r := &artifactoryResource{client: mockClient}
		initialModel := &Model{
			ProjectId:     types.StringValue("pid"),
			Region:        types.StringValue("reg"),
			ArtifactoryId: types.StringValue("art-id-123"),
			Proxy:         types.ObjectNull(proxyTypes),
		}
		state := tfsdk.State{Schema: schemaResp.Schema}
		diags := state.Set(ctx, initialModel)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		req := resource.DeleteRequest{State: state}
		var resp resource.DeleteResponse
		r.Delete(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics on delete success: %v", resp.Diagnostics)
		}
	})

	t.Run("delete wait error", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			DeleteArtifactoryExecuteMock: new(func(_ registry.ApiDeleteArtifactoryRequest) error {
				return nil
			}),
			GetArtifactoryExecuteMock: new(func(_ registry.ApiGetArtifactoryRequest) (*registry.Artifactory, error) {
				return nil, fmt.Errorf("wait polling error")
			}),
		}
		r := &artifactoryResource{client: mockClient}
		initialModel := &Model{
			ProjectId:     types.StringValue("pid"),
			Region:        types.StringValue("reg"),
			ArtifactoryId: types.StringValue("art-id-123"),
			Proxy:         types.ObjectNull(proxyTypes),
		}
		state := tfsdk.State{Schema: schemaResp.Schema}
		diags := state.Set(ctx, initialModel)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		req := resource.DeleteRequest{State: state}
		var resp resource.DeleteResponse
		r.Delete(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error diagnostics on delete wait error")
		}
	})

	t.Run("delete API error", func(t *testing.T) {
		mockClient := &registry.DefaultAPIServiceMock{
			DeleteArtifactoryExecuteMock: new(func(_ registry.ApiDeleteArtifactoryRequest) error {
				return fmt.Errorf("api delete error")
			}),
		}
		r := &artifactoryResource{client: mockClient}
		initialModel := &Model{
			ProjectId:     types.StringValue("pid"),
			Region:        types.StringValue("reg"),
			ArtifactoryId: types.StringValue("art-id-123"),
			Proxy:         types.ObjectNull(proxyTypes),
		}
		state := tfsdk.State{Schema: schemaResp.Schema}
		diags := state.Set(ctx, initialModel)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		req := resource.DeleteRequest{State: state}
		var resp resource.DeleteResponse
		r.Delete(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics on delete error")
		}
	})
}
