package argus

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/argus"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *argus.Job
		expected    Model
		isValid     bool
	}{
		{
			"default_ok",
			&argus.Job{
				JobName: utils.Ptr("name"),
			},
			Model{
				Id:                    types.StringValue("pid,iid,name"),
				ProjectId:             types.StringValue("pid"),
				InstanceId:            types.StringValue("iid"),
				Name:                  types.StringValue("name"),
				MetricsPath:           types.StringNull(),
				Scheme:                types.StringNull(),
				ScrapeInterval:        types.StringNull(),
				ScrapeTimeout:         types.StringNull(),
				SAML2:                 types.ObjectNull(saml2Types),
				BasicAuth:             types.ObjectNull(basicAuthTypes),
				Targets:               types.ListNull(types.ObjectType{AttrTypes: targetTypes}),
				BearerToken:           types.StringNull(),
				HonorLabels:           types.BoolNull(),
				HonorTimeStamps:       types.BoolNull(),
				HttpSdConfigs:         types.ListNull(types.ObjectType{AttrTypes: httpSdConfigsTypes}),
				MetricsRelabelConfigs: types.ListNull(types.ObjectType{AttrTypes: metricsRelabelConfigsTypes}),
				Oauth2:                types.ObjectNull(oauth2Types),
				TlsConfig:             types.ObjectNull(tlsConfigTypes),
			},
			true,
		},
		{
			description: "values_ok",
			input: &argus.Job{
				JobName:     utils.Ptr("name"),
				MetricsPath: utils.Ptr("/m"),
				BasicAuth: &argus.BasicAuth{
					Password: utils.Ptr("p"),
					Username: utils.Ptr("u"),
				},
				Params:         &map[string][]string{"saml2": {"disabled"}, "x": {"y", "z"}},
				SampleLimit:    utils.Ptr(int64(17)),
				Scheme:         utils.Ptr("scheme"),
				ScrapeInterval: utils.Ptr("1"),
				ScrapeTimeout:  utils.Ptr("2"),
				StaticConfigs: &[]argus.StaticConfigs{
					{
						Labels:  &map[string]string{"k1": "v1"},
						Targets: &[]string{"url1"},
					},
					{
						Labels:  &map[string]string{"k2": "v2", "k3": "v3"},
						Targets: &[]string{"url1", "url3"},
					},
					{
						Labels:  nil,
						Targets: &[]string{},
					},
				},
				HonorLabels:     utils.Ptr(false),
				HonorTimeStamps: utils.Ptr(false),
				HttpSdConfigs: &[]argus.HTTPServiceSD{
					{
						BasicAuth: &argus.BasicAuth{
							Password: utils.Ptr("p"),
							Username: utils.Ptr("u"),
						},
						RefreshInterval: utils.Ptr("60s"),
						TlsConfig: &argus.TLSConfig{
							InsecureSkipVerify: utils.Ptr(false),
						},
						Oauth2: &argus.OAuth2{
							ClientId:     utils.Ptr(""),
							ClientSecret: utils.Ptr(""),
							TokenUrl:     utils.Ptr(""),
							Scopes:       &[]string{""},
							TlsConfig: &argus.TLSConfig{
								InsecureSkipVerify: utils.Ptr(false),
							},
						},
						Url: utils.Ptr("url"),
					},
				},
				MetricsRelabelConfigs: &[]argus.MetricsRelabelConfig{
					{
						Action:       utils.Ptr("replace"),
						Modulus:      utils.Ptr(int64(1)),
						Regex:        utils.Ptr("reg"),
						Replacement:  utils.Ptr("rep"),
						Separator:    utils.Ptr(";"),
						TargetLabel:  utils.Ptr("target"),
						SourceLabels: &[]string{"source"},
					},
				},
				TlsConfig: &argus.TLSConfig{
					InsecureSkipVerify: utils.Ptr(false),
				},
			},
			expected: Model{
				Id:             types.StringValue("pid,iid,name"),
				ProjectId:      types.StringValue("pid"),
				InstanceId:     types.StringValue("iid"),
				Name:           types.StringValue("name"),
				MetricsPath:    types.StringValue("/m"),
				Scheme:         types.StringValue("scheme"),
				ScrapeInterval: types.StringValue("1"),
				ScrapeTimeout:  types.StringValue("2"),
				SampleLimit:    types.Int64Value(17),
				SAML2: types.ObjectValueMust(saml2Types, map[string]attr.Value{
					"enable_url_parameters": types.BoolValue(false),
				}),
				BasicAuth: types.ObjectValueMust(basicAuthTypes, map[string]attr.Value{
					"username": types.StringValue("u"),
					"password": types.StringValue("p"),
				}),
				Targets: types.ListValueMust(types.ObjectType{AttrTypes: targetTypes}, []attr.Value{
					types.ObjectValueMust(targetTypes, map[string]attr.Value{
						"urls": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url1")}),
						"labels": types.MapValueMust(types.StringType, map[string]attr.Value{
							"k1": types.StringValue("v1"),
						}),
					}),
					types.ObjectValueMust(targetTypes, map[string]attr.Value{
						"urls": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url1"), types.StringValue("url3")}),
						"labels": types.MapValueMust(types.StringType, map[string]attr.Value{
							"k2": types.StringValue("v2"),
							"k3": types.StringValue("v3"),
						}),
					}),
					types.ObjectValueMust(targetTypes, map[string]attr.Value{
						"urls":   types.ListValueMust(types.StringType, []attr.Value{}),
						"labels": types.MapNull(types.StringType),
					}),
				}),
				HonorLabels:     types.BoolValue(false),
				HonorTimeStamps: types.BoolValue(false),
				HttpSdConfigs: types.ListValueMust(types.ObjectType{AttrTypes: httpSdConfigsTypes}, []attr.Value{
					types.ObjectValueMust(httpSdConfigsTypes, map[string]attr.Value{
						"basic_auth": types.ObjectValueMust(basicAuthTypes, map[string]attr.Value{
							"username": types.StringValue("u"),
							"password": types.StringValue("p"),
						}),

						"refresh_interval": types.StringValue("60s"),
						"tls_config": types.ObjectValueMust(tlsConfigTypes, map[string]attr.Value{
							"insecure_skip_verify": types.BoolValue(false),
						}),
						"url": types.StringValue("url"),
						"oauth2": types.ObjectValueMust(oauth2Types, map[string]attr.Value{
							"client_id":     types.StringValue(""),
							"client_secret": types.StringValue(""),
							"token_url":     types.StringValue(""),
							"scopes": types.ListValueMust(types.StringType, []attr.Value{
								types.StringValue(""),
							}),
							"tls_config": types.ObjectValueMust(tlsConfigTypes, map[string]attr.Value{
								"insecure_skip_verify": types.BoolValue(false),
							}),
						}),
					}),
				}),
				MetricsRelabelConfigs: types.ListValueMust(types.ObjectType{AttrTypes: metricsRelabelConfigsTypes}, []attr.Value{
					types.ObjectValueMust(metricsRelabelConfigsTypes, map[string]attr.Value{
						"action":       types.StringValue("replace"),
						"modulus":      types.Int64Value(1),
						"regex":        types.StringValue("reg"),
						"replacement":  types.StringValue("rep"),
						"separator":    types.StringValue(";"),
						"target_label": types.StringValue("target"),
						"source_labels": types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("source"),
						}),
					}),
				}),
				TlsConfig: types.ObjectValueMust(tlsConfigTypes, map[string]attr.Value{
					"insecure_skip_verify": types.BoolValue(false),
				}),
			},
			isValid: true,
		},
		{
			"response_nil_fail",
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&argus.Job{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			err := mapFields(context.Background(), tt.input, state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description                     string
		input                           *Model
		inputSAML2                      *saml2Model
		inputBasicAuth                  *basicAuthModel
		inputTargets                    *[]targetModel
		inputHttpSdConfigsModel         *[]httpSdConfigModel
		inputMetricsRelabelConfigsModel *[]metricsRelabelConfigModel
		inputOauth2Model                *oauth2Model
		inputTlsConfigModel             *tlsConfigModel
		expected                        *argus.CreateScrapeConfigPayload
		isValid                         bool
	}{
		{
			"basic_ok",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
			},
			&saml2Model{},
			&basicAuthModel{},
			&[]targetModel{},
			&[]httpSdConfigModel{},
			&[]metricsRelabelConfigModel{},
			&oauth2Model{},
			&tlsConfigModel{},
			&argus.CreateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				// Defaults
				Scheme:                utils.Ptr("https"),
				ScrapeInterval:        utils.Ptr("5m"),
				ScrapeTimeout:         utils.Ptr("2m"),
				SampleLimit:           utils.Ptr(float64(5000)),
				StaticConfigs:         &[]argus.CreateScrapeConfigPayloadStaticConfigsInner{},
				Params:                &map[string]any{"saml2": []string{"enabled"}},
				HttpSdConfigs:         &[]argus.CreateScrapeConfigPayloadHttpSdConfigsInner{},
				MetricsRelabelConfigs: &[]argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner{},
			},
			true,
		},
		{
			"ok - false enable_url_parameters",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Name:        types.StringValue("Name"),
			},
			&saml2Model{
				EnableURLParameters: types.BoolValue(false),
			},
			&basicAuthModel{},
			&[]targetModel{},
			&[]httpSdConfigModel{},
			&[]metricsRelabelConfigModel{},
			&oauth2Model{},
			&tlsConfigModel{},
			&argus.CreateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				JobName:     utils.Ptr("Name"),
				Params:      &map[string]any{"saml2": []string{"disabled"}},
				// Defaults
				Scheme:                utils.Ptr("https"),
				ScrapeInterval:        utils.Ptr("5m"),
				ScrapeTimeout:         utils.Ptr("2m"),
				SampleLimit:           utils.Ptr(float64(5000)),
				StaticConfigs:         &[]argus.CreateScrapeConfigPayloadStaticConfigsInner{},
				HttpSdConfigs:         &[]argus.CreateScrapeConfigPayloadHttpSdConfigsInner{},
				MetricsRelabelConfigs: &[]argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner{},
			},
			true,
		},
		{
			"ok -  true enable_url_parameters",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Name:        types.StringValue("Name"),
			},
			&saml2Model{
				EnableURLParameters: types.BoolValue(true),
			},
			&basicAuthModel{},
			&[]targetModel{},
			&[]httpSdConfigModel{},
			&[]metricsRelabelConfigModel{},
			&oauth2Model{},
			&tlsConfigModel{},
			&argus.CreateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				JobName:     utils.Ptr("Name"),
				Params:      &map[string]any{"saml2": []string{"enabled"}},
				// Defaults
				Scheme:                utils.Ptr("https"),
				ScrapeInterval:        utils.Ptr("5m"),
				ScrapeTimeout:         utils.Ptr("2m"),
				SampleLimit:           utils.Ptr(float64(5000)),
				StaticConfigs:         &[]argus.CreateScrapeConfigPayloadStaticConfigsInner{},
				HttpSdConfigs:         &[]argus.CreateScrapeConfigPayloadHttpSdConfigsInner{},
				MetricsRelabelConfigs: &[]argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner{},
			},
			true,
		},
		{
			"ok - with basic auth",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Name:        types.StringValue("Name"),
			},
			&saml2Model{},
			&basicAuthModel{
				Username: types.StringValue("u"),
				Password: types.StringValue("p"),
			},
			&[]targetModel{},
			&[]httpSdConfigModel{},
			&[]metricsRelabelConfigModel{},
			&oauth2Model{},
			&tlsConfigModel{},
			&argus.CreateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				JobName:     utils.Ptr("Name"),
				BasicAuth: &argus.CreateScrapeConfigPayloadBasicAuth{
					Username: utils.Ptr("u"),
					Password: utils.Ptr("p"),
				},
				// Defaults
				Scheme:                utils.Ptr("https"),
				ScrapeInterval:        utils.Ptr("5m"),
				ScrapeTimeout:         utils.Ptr("2m"),
				SampleLimit:           utils.Ptr(float64(5000)),
				StaticConfigs:         &[]argus.CreateScrapeConfigPayloadStaticConfigsInner{},
				Params:                &map[string]any{"saml2": []string{"enabled"}},
				HttpSdConfigs:         &[]argus.CreateScrapeConfigPayloadHttpSdConfigsInner{},
				MetricsRelabelConfigs: &[]argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner{},
			},
			true,
		},
		{
			"ok - with targets",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Name:        types.StringValue("Name"),
			},
			&saml2Model{},
			&basicAuthModel{},
			&[]targetModel{
				{
					URLs:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url1")}),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{"k1": types.StringValue("v1")}),
				},
				{
					URLs:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url1"), types.StringValue("url3")}),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{"k2": types.StringValue("v2"), "k3": types.StringValue("v3")}),
				},
				{
					URLs:   types.ListValueMust(types.StringType, []attr.Value{}),
					Labels: types.MapNull(types.StringType),
				},
			},
			&[]httpSdConfigModel{},
			&[]metricsRelabelConfigModel{},
			&oauth2Model{},
			&tlsConfigModel{},
			&argus.CreateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				JobName:     utils.Ptr("Name"),
				StaticConfigs: &[]argus.CreateScrapeConfigPayloadStaticConfigsInner{
					{
						Targets: &[]string{"url1"},
						Labels:  &map[string]interface{}{"k1": "v1"},
					},
					{
						Targets: &[]string{"url1", "url3"},
						Labels:  &map[string]interface{}{"k2": "v2", "k3": "v3"},
					},
					{
						Targets: &[]string{},
						Labels:  &map[string]interface{}{},
					},
				},
				// Defaults
				Scheme:                utils.Ptr("https"),
				ScrapeInterval:        utils.Ptr("5m"),
				ScrapeTimeout:         utils.Ptr("2m"),
				SampleLimit:           utils.Ptr(float64(5000)),
				Params:                &map[string]any{"saml2": []string{"enabled"}},
				HttpSdConfigs:         &[]argus.CreateScrapeConfigPayloadHttpSdConfigsInner{},
				MetricsRelabelConfigs: &[]argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner{},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input, tt.inputSAML2, tt.inputBasicAuth, tt.inputTargets, tt.inputHttpSdConfigsModel, tt.inputMetricsRelabelConfigsModel, tt.inputOauth2Model, tt.inputTlsConfigModel)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description                     string
		input                           *Model
		inputSAML2                      *saml2Model
		basicAuthModel                  *basicAuthModel
		inputTargets                    *[]targetModel
		inputMetricsRelabelConfigsModel *[]metricsRelabelConfigModel
		inputTlsConfigModel             *tlsConfigModel
		expected                        *argus.UpdateScrapeConfigPayload
		isValid                         bool
	}{
		{
			"basic_ok",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
			},
			&saml2Model{},
			&basicAuthModel{},
			&[]targetModel{},
			&[]metricsRelabelConfigModel{},
			&tlsConfigModel{},
			&argus.UpdateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				// Defaults
				Scheme:                utils.Ptr("https"),
				ScrapeInterval:        utils.Ptr("5m"),
				ScrapeTimeout:         utils.Ptr("2m"),
				SampleLimit:           utils.Ptr(float64(5000)),
				StaticConfigs:         &[]argus.UpdateScrapeConfigPayloadStaticConfigsInner{},
				MetricsRelabelConfigs: &[]argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner{},
			},
			true,
		},
		{
			"ok -  true enable_url_parameters",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Scheme:      types.StringValue("http"),
			},
			&saml2Model{
				EnableURLParameters: types.BoolValue(true),
			},
			&basicAuthModel{},
			&[]targetModel{},
			&[]metricsRelabelConfigModel{},
			&tlsConfigModel{},
			&argus.UpdateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				// Defaults
				Scheme:                utils.Ptr("http"),
				ScrapeInterval:        utils.Ptr("5m"),
				ScrapeTimeout:         utils.Ptr("2m"),
				SampleLimit:           utils.Ptr(float64(5000)),
				StaticConfigs:         &[]argus.UpdateScrapeConfigPayloadStaticConfigsInner{},
				Params:                &map[string]any{"saml2": []string{"enabled"}},
				MetricsRelabelConfigs: &[]argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner{},
			},
			true,
		},
		{
			"ok -  false enable_url_parameters",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Scheme:      types.StringValue("http"),
			},
			&saml2Model{
				EnableURLParameters: types.BoolValue(false),
			},
			&basicAuthModel{},
			&[]targetModel{},
			&[]metricsRelabelConfigModel{},
			&tlsConfigModel{},
			&argus.UpdateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				// Defaults
				Scheme:                utils.Ptr("http"),
				ScrapeInterval:        utils.Ptr("5m"),
				ScrapeTimeout:         utils.Ptr("2m"),
				SampleLimit:           utils.Ptr(float64(5000)),
				StaticConfigs:         &[]argus.UpdateScrapeConfigPayloadStaticConfigsInner{},
				Params:                &map[string]any{"saml2": []string{"disabled"}},
				MetricsRelabelConfigs: &[]argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner{},
			},
			true,
		},
		{
			"ok - with basic auth",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Name:        types.StringValue("Name"),
			},
			&saml2Model{},
			&basicAuthModel{
				Username: types.StringValue("u"),
				Password: types.StringValue("p"),
			},
			&[]targetModel{},
			&[]metricsRelabelConfigModel{},
			&tlsConfigModel{},
			&argus.UpdateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				BasicAuth: &argus.CreateScrapeConfigPayloadBasicAuth{
					Username: utils.Ptr("u"),
					Password: utils.Ptr("p"),
				},
				// Defaults
				Scheme:                utils.Ptr("https"),
				ScrapeInterval:        utils.Ptr("5m"),
				ScrapeTimeout:         utils.Ptr("2m"),
				SampleLimit:           utils.Ptr(float64(5000)),
				StaticConfigs:         &[]argus.UpdateScrapeConfigPayloadStaticConfigsInner{},
				MetricsRelabelConfigs: &[]argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner{},
			},
			true,
		},
		{
			"ok - with targets",
			&Model{
				MetricsPath: types.StringValue("/metrics"),
				Name:        types.StringValue("Name"),
			},
			&saml2Model{},
			&basicAuthModel{},
			&[]targetModel{
				{
					URLs:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url1")}),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{"k1": types.StringValue("v1")}),
				},
				{
					URLs:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url1"), types.StringValue("url3")}),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{"k2": types.StringValue("v2"), "k3": types.StringValue("v3")}),
				},
				{
					URLs:   types.ListValueMust(types.StringType, []attr.Value{}),
					Labels: types.MapNull(types.StringType),
				},
			},
			&[]metricsRelabelConfigModel{},
			&tlsConfigModel{},
			&argus.UpdateScrapeConfigPayload{
				MetricsPath: utils.Ptr("/metrics"),
				StaticConfigs: &[]argus.UpdateScrapeConfigPayloadStaticConfigsInner{
					{
						Targets: &[]string{"url1"},
						Labels:  &map[string]interface{}{"k1": "v1"},
					},
					{
						Targets: &[]string{"url1", "url3"},
						Labels:  &map[string]interface{}{"k2": "v2", "k3": "v3"},
					},
					{
						Targets: &[]string{},
						Labels:  &map[string]interface{}{},
					},
				},
				// Defaults
				Scheme:                utils.Ptr("https"),
				ScrapeInterval:        utils.Ptr("5m"),
				ScrapeTimeout:         utils.Ptr("2m"),
				SampleLimit:           utils.Ptr(float64(5000)),
				MetricsRelabelConfigs: &[]argus.CreateScrapeConfigPayloadMetricsRelabelConfigsInner{},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			nil,
			nil,
			&[]metricsRelabelConfigModel{},
			&tlsConfigModel{},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input, tt.inputSAML2, tt.basicAuthModel, tt.inputTargets, tt.inputMetricsRelabelConfigsModel, tt.inputTlsConfigModel)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
