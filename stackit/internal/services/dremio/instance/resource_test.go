package dremio

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	dremioSdk "github.com/stackitcloud/stackit-sdk-go/services/dremio/v1alphaapi"
)

func TestMapFields(t *testing.T) {
	instanceId := uuid.New().String()
	tests := []struct {
		description string
		state       *Model
		input       *dremioSdk.DremioResponse
		expected    *Model
		wantErr     bool
	}{
		{
			"all_fields_filled",
			&Model{
				Region:    types.StringValue("rid"),
				ProjectId: types.StringValue("pid"),
			},
			&dremioSdk.DremioResponse{
				Id:          instanceId,
				CreateTime:  time.Now(),
				Description: utils.Ptr("minimal-required-values"),
				DisplayName: "greatName",
				Authentication: dremioSdk.Authentication{
					Azuread: &dremioSdk.Azuread{
						AuthorityUrl: "azure-authority",
						ClientId:     "azure-client",
						ClientSecret: "azure-secret",
						RedirectUrl:  utils.Ptr("azure-redirect"),
					},
					Oauth: &dremioSdk.Oauth{
						AuthorityUrl: "oauth-authority",
						ClientId:     "oauth-client",
						ClientSecret: "oauth-secret",
						JwtClaims: dremioSdk.OauthJwtClaims{
							UserName: "oauth-username",
						},
						Parameters: []dremioSdk.AuthParameters{
							{
								Name:  "oauth-parameter",
								Value: "oauth-value",
							},
						},
						RedirectUrl: utils.Ptr("oauth-redirect"),
						Scope:       utils.Ptr("oauth-scope"),
					},
					Type: dremioSdk.AUTHENTICATIONTYPE_LOCAL_ONLY,
				},
				Endpoints: dremioSdk.Endpoints{
					ArrowFlight: "flight",
					Catalog:     "catalog",
					Ui:          "ui",
				},
				State: "active",
			},
			&Model{
				Id: types.StringValue("pid,rid," + instanceId),

				ProjectId:  types.StringValue("pid"),
				Region:     types.StringValue("rid"),
				InstanceId: types.StringValue(instanceId),

				DisplayName: types.StringValue("greatName"),
				Description: types.StringValue("minimal-required-values"),

				Authentication: &AuthenticationModel{
					AzureAD: &AzureADModel{
						AuthorityUrl: types.StringValue("azure-authority"),
						ClientId:     types.StringValue("azure-client"),
						ClientSecret: types.StringValue("azure-secret"),
						RedirectUrl:  types.StringValue("azure-redirect"),
					},
					OAuth: &OAuthModel{
						AuthorityUrl: types.StringValue("oauth-authority"),
						ClientId:     types.StringValue("oauth-client"),
						ClientSecret: types.StringValue("oauth-secret"),
						JwtClaims: &JwtClaimsModel{
							UserName: types.StringValue("oauth-username"),
						},
						Parameters: []AuthParameterModel{
							{
								Name:  types.StringValue("oauth-parameter"),
								Value: types.StringValue("oauth-value"),
							},
						},
						RedirectUrl: types.StringValue("oauth-redirect"),
						Scope:       types.StringValue("oauth-scope"),
					},
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_LOCAL_ONLY)),
				},

				State:        types.StringValue("active"),
				ErrorMessage: types.StringNull(),
				Endpoints: types.ObjectValueMust(
					map[string]attr.Type{
						"arrow_flight": types.StringType,
						"catalog":      types.StringType,
						"ui":           types.StringType,
					},
					map[string]attr.Value{
						"arrow_flight": types.StringValue("flight"),
						"catalog":      types.StringValue("catalog"),
						"ui":           types.StringValue("ui"),
					},
				),
			},
			false,
		},
		{
			"nil response",
			&Model{
				Region:    types.StringValue("rid"),
				ProjectId: types.StringValue("pid"),
			},
			nil,
			&Model{
				Id:        types.StringValue("pid,rid,"),
				ProjectId: types.StringValue("pid"),
				Region:    types.StringValue("rid"),
			},
			true,
		},
		{
			"nil state",
			nil,
			&dremioSdk.DremioResponse{},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(tt.input, tt.state, "rid")
			if (err != nil) != tt.wantErr {
				t.Errorf("mapFields error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.expected, tt.state); diff != "" {
					t.Errorf("mapping mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		state       *Model
		expected    *dremioSdk.CreateDremioInstancePayload
		wantErr     bool
	}{
		{
			"success-local",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_LOCAL_ONLY)),
				},
			},
			&dremioSdk.CreateDremioInstancePayload{
				Authentication: &dremioSdk.Authentication{
					Type: dremioSdk.AUTHENTICATIONTYPE_LOCAL_ONLY,
				},
				Description: utils.Ptr("test description"),
				DisplayName: "displayName",
			},
			false,
		},
		{
			"success-oauth",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					OAuth: &OAuthModel{
						AuthorityUrl: types.StringValue("oauth-authority"),
						ClientId:     types.StringValue("oauth-client"),
						ClientSecret: types.StringValue("oauth-secret"),
						JwtClaims: &JwtClaimsModel{
							UserName: types.StringValue("oauth-username"),
						},
						Parameters: []AuthParameterModel{
							{
								Name:  types.StringValue("oauth-parameter"),
								Value: types.StringValue("oauth-value"),
							},
						},
						RedirectUrl: types.StringValue("oauth-redirect"),
						Scope:       types.StringValue("oauth-scope"),
					},
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_OAUTH)),
				},
			},
			&dremioSdk.CreateDremioInstancePayload{
				Authentication: &dremioSdk.Authentication{
					Oauth: &dremioSdk.Oauth{
						AuthorityUrl: "oauth-authority",
						ClientId:     "oauth-client",
						ClientSecret: "oauth-secret",
						JwtClaims: dremioSdk.OauthJwtClaims{
							UserName: "oauth-username",
						},
						Parameters: []dremioSdk.AuthParameters{
							{
								Name:  "oauth-parameter",
								Value: "oauth-value",
							},
						},
						RedirectUrl: utils.Ptr("oauth-redirect"),
						Scope:       utils.Ptr("oauth-scope"),
					},
					Type: dremioSdk.AUTHENTICATIONTYPE_OAUTH,
				},
				Description: utils.Ptr("test description"),
				DisplayName: "displayName",
			},
			false,
		},
		{
			"success-azuread",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					AzureAD: &AzureADModel{
						AuthorityUrl: types.StringValue("azure-authority"),
						ClientId:     types.StringValue("azure-client"),
						ClientSecret: types.StringValue("azure-secret"),
						RedirectUrl:  types.StringValue("azure-redirect"),
					},
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_AZUREAD)),
				},
			},
			&dremioSdk.CreateDremioInstancePayload{
				Authentication: &dremioSdk.Authentication{
					Azuread: &dremioSdk.Azuread{
						AuthorityUrl: "azure-authority",
						ClientId:     "azure-client",
						ClientSecret: "azure-secret",
						RedirectUrl:  utils.Ptr("azure-redirect"),
					},
					Type: dremioSdk.AUTHENTICATIONTYPE_AZUREAD,
				},
				Description: utils.Ptr("test description"),
				DisplayName: "displayName",
			},
			false,
		},
		{
			"idp-config-mismatch-local",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					AzureAD: &AzureADModel{
						AuthorityUrl: types.StringValue("azure-authority"),
						ClientId:     types.StringValue("azure-client"),
						ClientSecret: types.StringValue("azure-secret"),
						RedirectUrl:  types.StringValue("azure-redirect"),
					},
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_LOCAL_ONLY)),
				},
			},
			nil,
			true,
		},
		{
			"idp-config-mismatch-oauth",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					AzureAD: &AzureADModel{
						AuthorityUrl: types.StringValue("azure-authority"),
						ClientId:     types.StringValue("azure-client"),
						ClientSecret: types.StringValue("azure-secret"),
						RedirectUrl:  types.StringValue("azure-redirect"),
					},
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_OAUTH)),
				},
			},
			nil,
			true,
		},
		{
			"idp-config-mismatch-azuread",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_AZUREAD)),
					OAuth: &OAuthModel{
						AuthorityUrl: types.StringValue("oauth-authority"),
						ClientId:     types.StringValue("oauth-client"),
						ClientSecret: types.StringValue("oauth-secret"),
						JwtClaims: &JwtClaimsModel{
							UserName: types.StringValue("oauth-username"),
						},
						Parameters: []AuthParameterModel{
							{
								Name:  types.StringValue("oauth-parameter"),
								Value: types.StringValue("oauth-value"),
							},
						},
						RedirectUrl: types.StringValue("oauth-redirect"),
						Scope:       types.StringValue("oauth-scope"),
					},
				},
			},
			nil,
			true,
		},
		{
			"missing-idp-config-oauth",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_OAUTH)),
				},
			},
			nil,
			true,
		},
		{
			"missing-idp-config-azuread",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_AZUREAD)),
				},
			},
			nil,
			true,
		},
		{
			"nil model",
			nil,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toCreatePayload(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("toCreatePayload error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.expected, payload); diff != "" {
					t.Errorf("toCreatePayload mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		state       *Model
		expected    *dremioSdk.UpdateDremioInstancePayload
		wantErr     bool
	}{
		{
			"success",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_LOCAL_ONLY)),
				},
			},
			&dremioSdk.UpdateDremioInstancePayload{
				Authentication: &dremioSdk.Authentication{
					Type: dremioSdk.AUTHENTICATIONTYPE_LOCAL_ONLY,
				},
				Description: utils.Ptr("test description"),
				DisplayName: utils.Ptr("displayName"),
			},
			false,
		},
		{
			"success-oauth",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					OAuth: &OAuthModel{
						AuthorityUrl: types.StringValue("oauth-authority"),
						ClientId:     types.StringValue("oauth-client"),
						ClientSecret: types.StringValue("oauth-secret"),
						JwtClaims: &JwtClaimsModel{
							UserName: types.StringValue("oauth-username"),
						},
						Parameters: []AuthParameterModel{
							{
								Name:  types.StringValue("oauth-parameter"),
								Value: types.StringValue("oauth-value"),
							},
						},
						RedirectUrl: types.StringValue("oauth-redirect"),
						Scope:       types.StringValue("oauth-scope"),
					},
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_OAUTH)),
				},
			},
			&dremioSdk.UpdateDremioInstancePayload{
				Authentication: &dremioSdk.Authentication{
					Oauth: &dremioSdk.Oauth{
						AuthorityUrl: "oauth-authority",
						ClientId:     "oauth-client",
						ClientSecret: "oauth-secret",
						JwtClaims: dremioSdk.OauthJwtClaims{
							UserName: "oauth-username",
						},
						Parameters: []dremioSdk.AuthParameters{
							{
								Name:  "oauth-parameter",
								Value: "oauth-value",
							},
						},
						RedirectUrl: utils.Ptr("oauth-redirect"),
						Scope:       utils.Ptr("oauth-scope"),
					},
					Type: dremioSdk.AUTHENTICATIONTYPE_OAUTH,
				},
				Description: utils.Ptr("test description"),
				DisplayName: utils.Ptr("displayName"),
			},
			false,
		},
		{
			"success-azuread",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					AzureAD: &AzureADModel{
						AuthorityUrl: types.StringValue("azure-authority"),
						ClientId:     types.StringValue("azure-client"),
						ClientSecret: types.StringValue("azure-secret"),
						RedirectUrl:  types.StringValue("azure-redirect"),
					},
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_AZUREAD)),
				},
			},
			&dremioSdk.UpdateDremioInstancePayload{
				Authentication: &dremioSdk.Authentication{
					Azuread: &dremioSdk.Azuread{
						AuthorityUrl: "azure-authority",
						ClientId:     "azure-client",
						ClientSecret: "azure-secret",
						RedirectUrl:  utils.Ptr("azure-redirect"),
					},
					Type: dremioSdk.AUTHENTICATIONTYPE_AZUREAD,
				},
				Description: utils.Ptr("test description"),
				DisplayName: utils.Ptr("displayName"),
			},
			false,
		},
		{
			"idp-config-mismatch-local",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					AzureAD: &AzureADModel{
						AuthorityUrl: types.StringValue("azure-authority"),
						ClientId:     types.StringValue("azure-client"),
						ClientSecret: types.StringValue("azure-secret"),
						RedirectUrl:  types.StringValue("azure-redirect"),
					},
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_LOCAL_ONLY)),
				},
			},
			nil,
			true,
		},
		{
			"idp-config-mismatch-oauth",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					AzureAD: &AzureADModel{
						AuthorityUrl: types.StringValue("azure-authority"),
						ClientId:     types.StringValue("azure-client"),
						ClientSecret: types.StringValue("azure-secret"),
						RedirectUrl:  types.StringValue("azure-redirect"),
					},
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_OAUTH)),
				},
			},
			nil,
			true,
		},
		{
			"idp-config-mismatch-azuread",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_AZUREAD)),
					OAuth: &OAuthModel{
						AuthorityUrl: types.StringValue("oauth-authority"),
						ClientId:     types.StringValue("oauth-client"),
						ClientSecret: types.StringValue("oauth-secret"),
						JwtClaims: &JwtClaimsModel{
							UserName: types.StringValue("oauth-username"),
						},
						Parameters: []AuthParameterModel{
							{
								Name:  types.StringValue("oauth-parameter"),
								Value: types.StringValue("oauth-value"),
							},
						},
						RedirectUrl: types.StringValue("oauth-redirect"),
						Scope:       types.StringValue("oauth-scope"),
					},
				},
			},
			nil,
			true,
		},
		{
			"missing-idp-config-oauth",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_OAUTH)),
				},
			},
			nil,
			true,
		},
		{
			"missing-idp-config-azuread",
			&Model{
				Description: types.StringValue("test description"),
				DisplayName: types.StringValue("displayName"),
				Authentication: &AuthenticationModel{
					Type: types.StringValue(string(dremioSdk.AUTHENTICATIONTYPE_AZUREAD)),
				},
			},
			nil,
			true,
		},
		{
			"nil model",
			nil,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toUpdatePayload(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("toUpdatePayload error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.expected, payload); diff != "" {
					t.Errorf("toUpdatePayload mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
