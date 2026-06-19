package instance

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	workflows "github.com/stackitcloud/stackit-sdk-go/services/workflows/v1alphaapi"
)

var testTime = time.Now().UTC()

func ptrString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func fixtureInstance(mods ...func(instance *workflows.Instance)) *workflows.Instance {
	oauth := &workflows.OAuth2IdentityProvider{
		Type:              workflows.OAUTH2IDENTITYPROVIDERTYPE_OAUTH2,
		Name:              "azure",
		ClientId:          "client-id",
		ClientSecret:      "REDACTED-NEVER-RETURNED",
		Scope:             "openid email",
		DiscoveryEndpoint: "https://idp.example.com/.well-known/openid-configuration",
	}
	instance := &workflows.Instance{
		Id:               "iid",
		ProjectId:        "pid",
		RegionId:         "eu01",
		DisplayName:      "myinst",
		Version:          "workflows-3.0-airflow-3.1",
		Status:           workflows.INSTANCESTATUS_ACTIVE,
		CreatedAt:        testTime,
		Endpoints:        workflows.Endpoints{Url: "https://...stackit.cloud", RedirectUrl: "https://...stackit.cloud/oauth-callback"},
		IdentityProvider: workflows.OAuth2IdentityProviderAsIdentityProvider(oauth),
	}
	for _, mod := range mods {
		mod(instance)
	}
	return instance
}

func fixtureIdentityProviderObject(t *testing.T, mods ...func(*identityProviderModel)) types.Object {
	t.Helper()
	ipm := identityProviderModel{
		Type:              types.StringValue("oauth2"),
		Name:              types.StringValue("azure"),
		ClientID:          types.StringValue("client-id"),
		ClientSecret:      types.StringValue("PLANNED-SECRET"),
		Scope:             types.StringValue("openid email"),
		DiscoveryEndpoint: types.StringValue("https://idp.example.com/.well-known/openid-configuration"),
		APIAudience:       types.SetNull(types.StringType),
		Resource:          types.StringNull(),
		RolesClaim:        types.StringNull(),
	}
	for _, mod := range mods {
		mod(&ipm)
	}
	v, diags := types.ObjectValueFrom(context.Background(), identityProviderTypes, ipm)
	if diags.HasError() {
		t.Fatalf("building identity_provider fixture: %v", diags.Errors())
	}
	return v
}

func fixtureEndpointsObject(t *testing.T) types.Object {
	t.Helper()
	v, diags := types.ObjectValueFrom(context.Background(), endpointsTypes, endpointsModel{
		URL:         types.StringValue("https://...stackit.cloud"),
		RedirectURL: types.StringValue("https://...stackit.cloud/oauth-callback"),
	})
	if diags.HasError() {
		t.Fatalf("building endpoints fixture: %v", diags.Errors())
	}
	return v
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *workflows.Instance
		priorModel  *Model
		expected    *Model
		expectErr   bool
	}{
		{
			description: "default — oauth2, no network, no observability, no audience",
			input:       fixtureInstance(),
			priorModel: &Model{
				ProjectID:        types.StringValue("pid"),
				InstanceID:       types.StringValue("iid"),
				IdentityProvider: fixtureIdentityProviderObject(t),
			},
			expected: &Model{
				ID:                       types.StringValue("pid,eu01,iid"),
				InstanceID:               types.StringValue("iid"),
				Region:                   types.StringValue("eu01"),
				ProjectID:                types.StringValue("pid"),
				DisplayName:              types.StringValue("myinst"),
				Description:              types.StringNull(),
				Version:                  types.StringValue("workflows-3.0-airflow-3.1"),
				EnableStackitExampleDags: types.BoolValue(false),
				EnableAirflowExampleDags: types.BoolValue(false),
				ObservabilityID:          types.StringNull(),
				Network:                  types.ObjectNull(networkTypes),
				IdentityProvider:         fixtureIdentityProviderObject(t),
				Endpoints:                fixtureEndpointsObject(t),
				Status:                   types.StringValue("active"),
				CreatedAt:                types.StringValue(testTime.Format(time.RFC3339)),
			},
		},
		{
			description: "with network + observability + audience + flags",
			input: fixtureInstance(func(i *workflows.Instance) {
				i.Description = sdkUtils.Ptr("hello")
				i.EnableStackitExampleDags = sdkUtils.Ptr(true)
				i.EnableAirflowExampleDags = sdkUtils.Ptr(false)
				i.ObservabilityId = sdkUtils.Ptr("00000000-0000-0000-0000-000000000001")
				i.Network = &workflows.Network{Id: sdkUtils.Ptr("00000000-0000-0000-0000-000000000002")}
				oauth := &workflows.OAuth2IdentityProvider{
					Type:              workflows.OAUTH2IDENTITYPROVIDERTYPE_OAUTH2,
					Name:              "azure",
					ClientId:          "client-id",
					ClientSecret:      "REDACTED-NEVER-RETURNED",
					Scope:             "openid email",
					DiscoveryEndpoint: "https://idp.example.com/.well-known/openid-configuration",
					ApiAudience:       []string{"audience-a", "audience-b"},
				}
				i.IdentityProvider = workflows.OAuth2IdentityProviderAsIdentityProvider(oauth)
			}),
			priorModel: &Model{
				ProjectID:        types.StringValue("pid"),
				InstanceID:       types.StringValue("iid"),
				IdentityProvider: fixtureIdentityProviderObject(t),
			},
			expected: &Model{
				ID:                       types.StringValue("pid,eu01,iid"),
				InstanceID:               types.StringValue("iid"),
				Region:                   types.StringValue("eu01"),
				ProjectID:                types.StringValue("pid"),
				DisplayName:              types.StringValue("myinst"),
				Description:              types.StringValue("hello"),
				Version:                  types.StringValue("workflows-3.0-airflow-3.1"),
				EnableStackitExampleDags: types.BoolValue(true),
				EnableAirflowExampleDags: types.BoolValue(false),
				ObservabilityID:          types.StringValue("00000000-0000-0000-0000-000000000001"),
				Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
					"id": types.StringValue("00000000-0000-0000-0000-000000000002"),
				}),
				IdentityProvider: fixtureIdentityProviderObject(t, func(ipm *identityProviderModel) {
					ipm.APIAudience = types.SetValueMust(types.StringType, []attr.Value{
						types.StringValue("audience-a"),
						types.StringValue("audience-b"),
					})
				}),
				Endpoints: fixtureEndpointsObject(t),
				Status:    types.StringValue("active"),
				CreatedAt: types.StringValue(testTime.Format(time.RFC3339)),
			},
		},
		{
			description: "preserves client_secret from prior model when API doesn't return it",
			input:       fixtureInstance(),
			priorModel: &Model{
				ProjectID:  types.StringValue("pid"),
				InstanceID: types.StringValue("iid"),
				IdentityProvider: fixtureIdentityProviderObject(t, func(ipm *identityProviderModel) {
					ipm.ClientSecret = types.StringValue("ORIGINAL-PLAN-SECRET")
				}),
			},
			expected: &Model{
				ID:                       types.StringValue("pid,eu01,iid"),
				InstanceID:               types.StringValue("iid"),
				Region:                   types.StringValue("eu01"),
				ProjectID:                types.StringValue("pid"),
				DisplayName:              types.StringValue("myinst"),
				Description:              types.StringNull(),
				Version:                  types.StringValue("workflows-3.0-airflow-3.1"),
				EnableStackitExampleDags: types.BoolValue(false),
				EnableAirflowExampleDags: types.BoolValue(false),
				ObservabilityID:          types.StringNull(),
				Network:                  types.ObjectNull(networkTypes),
				IdentityProvider: fixtureIdentityProviderObject(t, func(ipm *identityProviderModel) {
					ipm.ClientSecret = types.StringValue("ORIGINAL-PLAN-SECRET")
				}),
				Endpoints: fixtureEndpointsObject(t),
				Status:    types.StringValue("active"),
				CreatedAt: types.StringValue(testTime.Format(time.RFC3339)),
			},
		},
		{
			description: "empty api_audience list yields empty (not null) list in state",
			input: fixtureInstance(func(i *workflows.Instance) {
				oauth := &workflows.OAuth2IdentityProvider{
					Type:              workflows.OAUTH2IDENTITYPROVIDERTYPE_OAUTH2,
					Name:              "azure",
					ClientId:          "client-id",
					ClientSecret:      "REDACTED",
					Scope:             "openid email",
					DiscoveryEndpoint: "https://idp.example.com/.well-known/openid-configuration",
					ApiAudience:       []string{},
				}
				i.IdentityProvider = workflows.OAuth2IdentityProviderAsIdentityProvider(oauth)
			}),
			priorModel: &Model{
				ProjectID:        types.StringValue("pid"),
				InstanceID:       types.StringValue("iid"),
				IdentityProvider: fixtureIdentityProviderObject(t),
			},
			expected: &Model{
				ID:                       types.StringValue("pid,eu01,iid"),
				InstanceID:               types.StringValue("iid"),
				Region:                   types.StringValue("eu01"),
				ProjectID:                types.StringValue("pid"),
				DisplayName:              types.StringValue("myinst"),
				Description:              types.StringNull(),
				Version:                  types.StringValue("workflows-3.0-airflow-3.1"),
				EnableStackitExampleDags: types.BoolValue(false),
				EnableAirflowExampleDags: types.BoolValue(false),
				ObservabilityID:          types.StringNull(),
				Network:                  types.ObjectNull(networkTypes),
				IdentityProvider: fixtureIdentityProviderObject(t, func(ipm *identityProviderModel) {
					ipm.APIAudience = types.SetValueMust(types.StringType, []attr.Value{})
				}),
				Endpoints: fixtureEndpointsObject(t),
				Status:    types.StringValue("active"),
				CreatedAt: types.StringValue(testTime.Format(time.RFC3339)),
			},
		},
		{
			description: "nil instance fails",
			input:       nil,
			priorModel:  &Model{},
			expectErr:   true,
		},
		{
			description: "missing instance id fails",
			input:       fixtureInstance(func(i *workflows.Instance) { i.Id = "" }),
			priorModel:  &Model{ProjectID: types.StringValue("pid")},
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, tt.priorModel, "eu01")
			if (err != nil) != tt.expectErr {
				t.Fatalf("mapFields error = %v, expectErr = %v", err, tt.expectErr)
			}
			if tt.expectErr {
				return
			}
			if diff := cmp.Diff(tt.expected, tt.priorModel, cmp.Comparer(func(a, b basetypes.ObjectValue) bool { return a.Equal(b) })); diff != "" {
				t.Errorf("mapFields mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	ctx := context.Background()
	model := &Model{
		DisplayName:              types.StringValue("myinst"),
		Description:              types.StringValue("hello"),
		Version:                  types.StringValue("workflows-3.0-airflow-3.1"),
		EnableStackitExampleDags: types.BoolValue(true),
		EnableAirflowExampleDags: types.BoolValue(false),
		ObservabilityID:          types.StringValue("00000000-0000-0000-0000-000000000001"),
		Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
			"id": types.StringValue("00000000-0000-0000-0000-000000000002"),
		}),
		IdentityProvider: fixtureIdentityProviderObject(t),
	}

	payload, err := toCreatePayload(ctx, model)
	if err != nil {
		t.Fatalf("toCreatePayload: %v", err)
	}
	if payload.DisplayName != "myinst" {
		t.Errorf("DisplayName = %q, want %q", payload.DisplayName, "myinst")
	}
	if got := ptrString(payload.Description); got != "hello" {
		t.Errorf("Description = %q, want %q", got, "hello")
	}
	if payload.Version != "workflows-3.0-airflow-3.1" {
		t.Errorf("Version = %q", payload.Version)
	}
	if payload.EnableStackitExampleDags == nil || *payload.EnableStackitExampleDags != true {
		t.Errorf("EnableStackitExampleDags = %v, want true", payload.EnableStackitExampleDags)
	}
	if payload.Network == nil || ptrString(payload.Network.Id) != "00000000-0000-0000-0000-000000000002" {
		t.Errorf("Network.Id mismatch: %+v", payload.Network)
	}
	if payload.IdentityProvider == nil || payload.IdentityProvider.OAuth2IdentityProvider == nil {
		t.Fatalf("IdentityProvider not wrapped as OAuth2")
	}
	if payload.IdentityProvider.OAuth2IdentityProvider.ClientSecret != "PLANNED-SECRET" {
		t.Errorf("ClientSecret = %q, want PLANNED-SECRET", payload.IdentityProvider.OAuth2IdentityProvider.ClientSecret)
	}
}

func TestBuildUpdateInstancePayload_OmitsDisplayName(t *testing.T) {
	plan := &Model{
		DisplayName:              types.StringValue("does-not-matter"),
		Description:              types.StringValue("hello"),
		Version:                  types.StringValue("v"),
		EnableStackitExampleDags: types.BoolValue(true),
		EnableAirflowExampleDags: types.BoolValue(false),
	}
	state := &Model{Description: types.StringValue("hello")}
	payload := toUpdateInstancePayload(plan, state)

	raw, err := payload.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(raw); strings.Contains(got, "displayName") {
		t.Errorf("payload must not include displayName, got: %s", got)
	}

	if got := ptrString(payload.Description); got != "hello" {
		t.Errorf("Description = %q", got)
	}
	if got := ptrString(payload.Version); got != "v" {
		t.Errorf("Version = %q", got)
	}
	if payload.EnableStackitExampleDags == nil || *payload.EnableStackitExampleDags != true {
		t.Errorf("EnableStackitExampleDags = %v", payload.EnableStackitExampleDags)
	}
	if payload.EnableAirflowExampleDags == nil || *payload.EnableAirflowExampleDags != false {
		t.Errorf("EnableAirflowExampleDags = %v", payload.EnableAirflowExampleDags)
	}
}

// TestBuildUpdateInstancePayload_ClearsDescription verifies that removing
// description from the config (plan-null while state had a value) results in
// `""` being sent — the server uses empty string as the clear signal.
func TestBuildUpdateInstancePayload_ClearsDescription(t *testing.T) {
	plan := &Model{Description: types.StringNull()}
	state := &Model{Description: types.StringValue("had this")}
	payload := toUpdateInstancePayload(plan, state)
	if payload.Description == nil {
		t.Fatalf("Description should be set to \"\" to clear, got nil")
	}
	if *payload.Description != "" {
		t.Errorf("Description = %q, want \"\"", *payload.Description)
	}
}

// TestBuildUpdateInstancePayload_OmitsUnsetDescription verifies that an
// always-unset description doesn't end up in the payload as "". Sending "" on
// every update would clobber server-side defaults / future schema changes.
func TestBuildUpdateInstancePayload_OmitsUnsetDescription(t *testing.T) {
	plan := &Model{Description: types.StringNull()}
	state := &Model{Description: types.StringNull()}
	payload := toUpdateInstancePayload(plan, state)
	if payload.Description != nil {
		t.Errorf("Description should be nil when never set, got %q", *payload.Description)
	}
}

func TestBuildUpdateIdentityProviderPayload_OAuth2_OmitsType(t *testing.T) {
	plan := &Model{IdentityProvider: fixtureIdentityProviderObject(t)}
	state := &Model{IdentityProvider: fixtureIdentityProviderObject(t)}
	payload, err := toUpdateIdentityProviderPayload(context.Background(), plan, state)
	if err != nil {
		t.Fatalf("toUpdateIdentityProviderPayload: %v", err)
	}
	if payload.OAuth2IdentityProviderPatch == nil {
		t.Fatalf("expected OAuth2IdentityProviderPatch variant")
	}

	raw, err := payload.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(raw); strings.Contains(got, `"type"`) {
		t.Errorf("OAuth2IdentityProviderPatch payload must not include 'type', got: %s", got)
	}
	if ptrString(payload.OAuth2IdentityProviderPatch.ClientId) != "client-id" {
		t.Errorf("ClientId mismatch")
	}
	if ptrString(payload.OAuth2IdentityProviderPatch.ClientSecret) != "PLANNED-SECRET" {
		t.Errorf("ClientSecret mismatch")
	}
}

// TestBuildUpdateIdentityProviderPayload_ClearsOptionalStrings verifies that
// removing optional IdP fields (resource, roles_claim) from the config
// translates to "" so the server clears them.
func TestBuildUpdateIdentityProviderPayload_ClearsOptionalStrings(t *testing.T) {
	plan := &Model{
		IdentityProvider: fixtureIdentityProviderObject(t, func(ipm *identityProviderModel) {
			ipm.Resource = types.StringNull()
			ipm.RolesClaim = types.StringNull()
		}),
	}
	state := &Model{
		IdentityProvider: fixtureIdentityProviderObject(t, func(ipm *identityProviderModel) {
			ipm.Resource = types.StringValue("had-resource")
			ipm.RolesClaim = types.StringValue("had-claim")
		}),
	}
	payload, err := toUpdateIdentityProviderPayload(context.Background(), plan, state)
	if err != nil {
		t.Fatalf("toUpdateIdentityProviderPayload: %v", err)
	}
	if payload.OAuth2IdentityProviderPatch.Resource == nil || *payload.OAuth2IdentityProviderPatch.Resource != "" {
		t.Errorf("Resource = %v, want pointer to \"\"", payload.OAuth2IdentityProviderPatch.Resource)
	}
	if payload.OAuth2IdentityProviderPatch.RolesClaim == nil || *payload.OAuth2IdentityProviderPatch.RolesClaim != "" {
		t.Errorf("RolesClaim = %v, want pointer to \"\"", payload.OAuth2IdentityProviderPatch.RolesClaim)
	}
}

func TestBuildIdentityProvider_UnsupportedType(t *testing.T) {
	model := &Model{
		IdentityProvider: fixtureIdentityProviderObject(t, func(ipm *identityProviderModel) {
			ipm.Type = types.StringValue("ftp")
		}),
	}
	if _, err := buildIdentityProvider(context.Background(), model); err == nil {
		t.Errorf("expected error for unsupported type")
	}
}

func TestInstanceFieldsChanged(t *testing.T) {
	base := &Model{
		DisplayName:              types.StringValue("a"),
		Description:              types.StringValue("a"),
		Version:                  types.StringValue("v"),
		EnableStackitExampleDags: types.BoolValue(false),
		EnableAirflowExampleDags: types.BoolValue(false),
	}
	tests := []struct {
		desc string
		mut  func(*Model)
		want bool
	}{
		{"unchanged", func(m *Model) {}, false},
		// display_name is RequiresReplace (server rejects on update) — not in instanceFieldsChanged.
		{"displayName changed → no UpdateInstance call", func(m *Model) { m.DisplayName = types.StringValue("b") }, false},
		{"description changed", func(m *Model) { m.Description = types.StringValue("b") }, true},
		{"version changed", func(m *Model) { m.Version = types.StringValue("v2") }, true},
		{"enable_stackit_example_dags changed", func(m *Model) { m.EnableStackitExampleDags = types.BoolValue(true) }, true},
		{"enable_airflow_example_dags changed", func(m *Model) { m.EnableAirflowExampleDags = types.BoolValue(true) }, true},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			plan := *base
			tt.mut(&plan)
			if got := instanceFieldsChanged(&plan, base); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestToCreatePayload_EmptyDescriptionOmitted regression: writing
// `description = ""` on Create must not send "" to the server — that would
// race with the server's normalize-on-Update logic.
func TestToCreatePayload_EmptyDescriptionOmitted(t *testing.T) {
	model := &Model{
		DisplayName:      types.StringValue("x"),
		Description:      types.StringValue(""),
		Version:          types.StringValue("v"),
		IdentityProvider: fixtureIdentityProviderObject(t),
	}
	payload, err := toCreatePayload(context.Background(), model)
	if err != nil {
		t.Fatalf("toCreatePayload: %v", err)
	}
	if payload.Description != nil {
		t.Errorf("Description should be nil for empty-string plan, got %q", *payload.Description)
	}
}

func TestValidateInstanceConfig(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		desc      string
		mut       func(*identityProviderModel)
		idpType   string // "oauth2" (default), "stackit", or "none" to test top-level nil/skip paths
		wantErrs  []string
	}{
		{desc: "oauth2 happy path", mut: nil},
		{desc: "oauth2 missing client_id", mut: func(ipm *identityProviderModel) { ipm.ClientID = types.StringValue("") }, wantErrs: []string{"client_id"}},
		{desc: "oauth2 null client_secret", mut: func(ipm *identityProviderModel) { ipm.ClientSecret = types.StringNull() }, wantErrs: []string{"client_secret"}},
		{desc: "oauth2 missing scope", mut: func(ipm *identityProviderModel) { ipm.Scope = types.StringValue("") }, wantErrs: []string{"scope"}},
		{desc: "oauth2 missing discovery_endpoint", mut: func(ipm *identityProviderModel) { ipm.DiscoveryEndpoint = types.StringValue("") }, wantErrs: []string{"discovery_endpoint"}},
		{desc: "oauth2 multiple missing", mut: func(ipm *identityProviderModel) {
			ipm.ClientID = types.StringValue("")
			ipm.Scope = types.StringNull()
		}, wantErrs: []string{"client_id", "scope"}},
		{desc: "unknown defers (e.g. unresolved variable)", mut: func(ipm *identityProviderModel) { ipm.ClientSecret = types.StringUnknown() }},
		{desc: "stackit type skipped (no oauth2 required fields)", idpType: "stackit"},
		{desc: "null identity_provider skipped (schema marks Required, framework catches null)", idpType: "none"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var idp types.Object
			switch tt.idpType {
			case "stackit":
				idp = fixtureIdentityProviderObject(t, func(ipm *identityProviderModel) { ipm.Type = types.StringValue("stackit") })
			case "none":
				idp = types.ObjectNull(identityProviderTypes)
			default:
				if tt.mut != nil {
					idp = fixtureIdentityProviderObject(t, tt.mut)
				} else {
					idp = fixtureIdentityProviderObject(t)
				}
			}
			model := &Model{IdentityProvider: idp}
			var diags diag.Diagnostics
			validateInstanceConfig(ctx, model, &diags)

			if len(tt.wantErrs) == 0 {
				if diags.HasError() {
					t.Fatalf("expected no errors, got: %v", diags.Errors())
				}
				return
			}
			gotMsgs := make([]string, 0, len(diags.Errors()))
			for _, d := range diags.Errors() {
				gotMsgs = append(gotMsgs, d.Detail())
			}
			for _, want := range tt.wantErrs {
				found := false
				for _, msg := range gotMsgs {
					if strings.Contains(msg, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected an error mentioning %q, got: %v", want, gotMsgs)
				}
			}
		})
	}
}
