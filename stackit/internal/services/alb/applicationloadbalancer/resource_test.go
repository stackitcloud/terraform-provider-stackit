package alb

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	albSdk "github.com/stackitcloud/stackit-sdk-go/services/alb"
)

const (
	projectID       = "b8c3fbaa-3ab4-4a8e-9584-de22453d046f"
	region          = "eu01"
	lbName          = "example-lb2"
	externalAddress = "188.34.80.229"
	lbVersion       = "lb-1"
	lbID            = projectID + "," + region + "," + lbName
	sgLBID          = "8c06e3b6-531b-43a0-b965-3ae73da83d1b"
	sgTargetID      = "19cc8a91-d590-4166-b27d-211da3cb44d3"
	targetPoolName  = "my-pool"
	credentialsRef  = "credentials-nzkp4"
)

func fixtureModel(explicitBool *bool, mods ...func(m *Model)) *Model {
	resp := &Model{
		Id:                             types.StringValue(lbID),
		ProjectId:                      types.StringValue(projectID),
		DisableSecurityGroupAssignment: types.BoolPointerValue(explicitBool),
		Errors: types.SetValueMust(
			types.ObjectType{AttrTypes: errorsType},
			[]attr.Value{
				types.ObjectValueMust(
					errorsType,
					map[string]attr.Value{
						"description": types.StringValue("quota test error"),
						"type":        types.StringValue(string(albSdk.LOADBALANCERERRORTYPE_QUOTA_SECGROUP_EXCEEDED)),
					},
				),
				types.ObjectValueMust(
					errorsType,
					map[string]attr.Value{
						"description": types.StringValue("fip test error"),
						"type":        types.StringValue(string(albSdk.LOADBALANCERERRORTYPE_FIP_NOT_CONFIGURED)),
					},
				),
			},
		),
		ExternalAddress: types.StringValue(externalAddress),
		Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
			"key":  types.StringValue("value"),
			"key2": types.StringValue("value2"),
		}),
		Listeners: types.ListValueMust(
			types.ObjectType{AttrTypes: listenerTypes},
			[]attr.Value{
				types.ObjectValueMust(
					listenerTypes,
					map[string]attr.Value{
						"name":            types.StringValue("http-80"),
						"port":            types.Int64Value(80),
						"protocol":        types.StringValue("PROTOCOL_HTTP"),
						"waf_config_name": types.StringValue("my-waf-config"),
						"http": types.ObjectValueMust(
							httpTypes,
							map[string]attr.Value{
								"hosts": types.ListValueMust(
									types.ObjectType{AttrTypes: hostConfigTypes},
									[]attr.Value{types.ObjectValueMust(
										hostConfigTypes,
										map[string]attr.Value{
											"host": types.StringValue("*"),
											"rules": types.ListValueMust(
												types.ObjectType{AttrTypes: ruleTypes},
												[]attr.Value{types.ObjectValueMust(
													ruleTypes,
													map[string]attr.Value{
														"target_pool": types.StringValue(targetPoolName),
														"web_socket":  types.BoolPointerValue(explicitBool),
														"path": types.ObjectValueMust(pathTypes,
															map[string]attr.Value{
																"exact_match": types.StringNull(),
																"prefix":      types.StringValue("/"),
															},
														),
														"headers": types.SetValueMust(
															types.ObjectType{AttrTypes: headersTypes},
															[]attr.Value{types.ObjectValueMust(
																headersTypes,
																map[string]attr.Value{
																	"name":        types.StringValue("a-header"),
																	"exact_match": types.StringValue("value"),
																}),
															},
														),
														"query_parameters": types.SetValueMust(
															types.ObjectType{AttrTypes: queryParameterTypes},
															[]attr.Value{types.ObjectValueMust(
																queryParameterTypes,
																map[string]attr.Value{
																	"name":        types.StringValue("a_query_parameter"),
																	"exact_match": types.StringValue("value"),
																}),
															},
														),
														"cookie_persistence": types.ObjectValueMust(
															cookiePersistenceTypes,
															map[string]attr.Value{
																"name": types.StringValue("cookie_name"),
																"ttl":  types.StringValue("3s"),
															},
														),
													},
												),
												}),
										},
									),
									}),
							},
						),
						"https": types.ObjectValueMust(
							httpsTypes,
							map[string]attr.Value{
								"certificate_config": types.ObjectValueMust(
									certificateConfigTypes,
									map[string]attr.Value{
										"certificate_ids": types.SetValueMust(
											types.StringType,
											[]attr.Value{
												types.StringValue(credentialsRef),
											},
										),
									},
								),
							},
						),
					},
				),
			},
		),
		LoadBalancerSecurityGroup: types.ObjectValueMust(
			loadBalancerSecurityGroupType,
			map[string]attr.Value{
				"id":   types.StringValue(sgLBID),
				"name": types.StringValue("loadbalancer/" + lbName + "/backend-port"),
			},
		),
		Name: types.StringValue(lbName),
		Networks: types.SetValueMust(
			types.ObjectType{AttrTypes: networkTypes},
			[]attr.Value{
				types.ObjectValueMust(
					networkTypes,
					map[string]attr.Value{
						"network_id": types.StringValue("c7c92cc1-a6bd-4e15-a129-b6e2b9899bbc"),
						"role":       types.StringValue("ROLE_LISTENERS"),
					},
				),
				types.ObjectValueMust(
					networkTypes,
					map[string]attr.Value{
						"network_id": types.StringValue("ed3f1822-ca1c-4969-bea6-74c6b3e9aa40"),
						"role":       types.StringValue("ROLE_TARGETS"),
					},
				),
			},
		),
		Options: types.ObjectValueMust(
			optionsTypes,
			map[string]attr.Value{
				"access_control": types.ObjectValueMust(
					accessControlTypes,
					map[string]attr.Value{
						"allowed_source_ranges": types.SetValueMust(
							types.StringType,
							[]attr.Value{
								types.StringValue("192.168.0.0"),
								types.StringValue("192.168.0.1"),
							},
						),
					},
				),
				"ephemeral_address":    types.BoolPointerValue(explicitBool),
				"private_network_only": types.BoolPointerValue(explicitBool),
				"observability": types.ObjectValueMust(
					observabilityTypes,
					map[string]attr.Value{
						"logs": types.ObjectValueMust(
							observabilityOptionTypes,
							map[string]attr.Value{
								"credentials_ref": types.StringValue(credentialsRef),
								"push_url":        types.StringValue("http://www.example.org/push"),
							},
						),
						"metrics": types.ObjectValueMust(
							observabilityOptionTypes,
							map[string]attr.Value{
								"credentials_ref": types.StringValue(credentialsRef),
								"push_url":        types.StringValue("http://www.example.org/pull"),
							},
						),
					},
				),
			},
		),
		PlanId:         types.StringValue("p10"),
		PrivateAddress: types.StringValue("10.1.11.0"),
		Region:         types.StringValue(region),
		TargetPools: types.ListValueMust(
			types.ObjectType{AttrTypes: targetPoolTypes},
			[]attr.Value{
				types.ObjectValueMust(
					targetPoolTypes,
					map[string]attr.Value{
						"name":        types.StringValue(targetPoolName),
						"target_port": types.Int64Value(80),
						"targets": types.SetValueMust(
							types.ObjectType{AttrTypes: targetTypes},
							[]attr.Value{
								types.ObjectValueMust(
									targetTypes,
									map[string]attr.Value{
										"display_name": types.StringValue("test-backend-server"),
										"ip":           types.StringValue("192.168.0.218"),
									},
								),
							},
						),
						"tls_config": types.ObjectValueMust(
							tlsConfigTypes,
							map[string]attr.Value{
								"enabled":                     types.BoolPointerValue(explicitBool),
								"custom_ca":                   types.StringValue("-----BEGIN CERTIFICATE-----\nMIIDCzCCAfOgAwIBAgIUTyPsTWC9ly7o+wNFYm0uu1+P8IEwDQYJKoZIhvcNAQEL\nBQAwFTETMBEGA1UEAwwKTXlDdXN0b21DQTAeFw0yNTAyMTkxOTI0MjBaFw0yNjAy\nMTkxOTI0MjBaMBUxEzARBgNVBAMMCk15Q3VzdG9tQ0EwggEiMA0GCSqGSIb3DQEB\nAQUAA4IBDwAwggEKAoIBAQCQMEYKbiNxU37fEwBOxkvCshBR+0MwxwLW8Mi3/pvo\nn3huxjcm7EaKW9r7kIaoHXbTS1tnO6rHAHKBDxzuoYD7C2SMSiLxddquNRvpkLaP\n8qAXneQY2VP7LzsAgsC04PKG0YC1NgF5sJGsiWIRGIm+csYLnPMnwaAGx4IvY6mH\nAmM64b6QRCg36LK+P6N9KTvSQLvvmFdkA2sDToCmN/Amp6xNDFq+aQGLwdQQqHDP\nTaUqPmEyiFHKvFUaFMNQVk8B1Om8ASo69m8U3Eat4ZOVW1titE393QkOdA6ZypMC\nrJJpeNNLLJq3mIOWOd7GEyAvjUfmJwGhqEFS7lMG67hnAgMBAAGjUzBRMB0GA1Ud\nDgQWBBSk/IM5jaOAJL3/Knyq3cVva04YZDAfBgNVHSMEGDAWgBSk/IM5jaOAJL3/\nKnyq3cVva04YZDAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQBe\nZ/mE8rNIbNbHQep/VppshaZUzgdy4nsmh0wvxMuHIQP0KHrxLCkhOn7A9fu4mY/P\nQ+8QqlnjTsM4cqiuFcd5V1Nk9VF/e5X3HXCDHh/jBFw+O5TGVAR/7DBw31lYv/Lt\nHakkjQCdawuvH3osO/UkElM/i2KC+iYBavTenm97AR7WGgW15/MIqxNaYE+nJth/\ndcVD0b5qSuYQaEmZ3CzMUi188R+go5ozCf2cOaa+3/LEYAaI3vKiSE8KTsshyoKm\nO6YZqrVxQCWCDTOsd28k7lHt8wJ+jzYcjCu60DUpg1ZpY+ZnmrE8vPPDb/zXhBn6\n/llXTWOUjmuTKnGsIDP5\n-----END CERTIFICATE-----"),
								"skip_certificate_validation": types.BoolPointerValue(explicitBool),
							},
						),
						"active_health_check": types.ObjectValueMust(
							activeHealthCheckTypes,
							map[string]attr.Value{
								"healthy_threshold":   types.Int64Value(1),
								"interval":            types.StringValue("2s"),
								"interval_jitter":     types.StringValue("3s"),
								"timeout":             types.StringValue("4s"),
								"unhealthy_threshold": types.Int64Value(5),
								"http_health_checks": types.ObjectValueMust(
									httpHealthChecksTypes,
									map[string]attr.Value{
										"ok_status": types.SetValueMust(
											types.StringType,
											[]attr.Value{
												types.StringValue("200"),
												types.StringValue("201"),
											},
										),
										"path": types.StringValue("/health"),
									},
								),
							},
						),
					},
				),
			},
		),
		TargetSecurityGroup: types.ObjectValueMust(
			targetSecurityGroupType,
			map[string]attr.Value{
				"id":   types.StringValue(sgTargetID),
				"name": types.StringValue("loadbalancer/" + lbName + "/backend"),
			},
		),
		Version: types.StringValue(lbVersion),
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func fixtureModelNull(mods ...func(m *Model)) *Model {
	resp := &Model{
		Id:                             types.StringNull(),
		ProjectId:                      types.StringNull(),
		DisableSecurityGroupAssignment: types.BoolNull(),
		Errors:                         types.SetNull(types.ObjectType{AttrTypes: errorsType}),
		ExternalAddress:                types.StringNull(),
		Labels:                         types.MapNull(types.StringType),
		Listeners:                      types.ListNull(types.ObjectType{AttrTypes: listenerTypes}),
		LoadBalancerSecurityGroup:      types.ObjectNull(loadBalancerSecurityGroupType),
		Name:                           types.StringNull(),
		Networks:                       types.SetNull(types.ObjectType{AttrTypes: networkTypes}),
		Options:                        types.ObjectNull(optionsTypes),
		PlanId:                         types.StringNull(),
		PrivateAddress:                 types.StringNull(),
		Region:                         types.StringNull(),
		TargetPools:                    types.ListNull(types.ObjectType{AttrTypes: targetPoolTypes}),
		TargetSecurityGroup:            types.ObjectNull(targetSecurityGroupType),
		Version:                        types.StringNull(),
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func fixtureCreatePayload(lb *albSdk.LoadBalancer) *albSdk.CreateLoadBalancerPayload {
	return &albSdk.CreateLoadBalancerPayload{
		DisableTargetSecurityGroupAssignment: lb.DisableTargetSecurityGroupAssignment,
		ExternalAddress:                      lb.ExternalAddress,
		Labels:                               lb.Labels,
		Listeners:                            lb.Listeners,
		Name:                                 lb.Name,
		Networks:                             lb.Networks,
		Options:                              lb.Options,
		PlanId:                               lb.PlanId,
		TargetPools:                          lb.TargetPools,
	}
}

func fixtureUpdatePayload(lb *albSdk.LoadBalancer) *albSdk.UpdateLoadBalancerPayload {
	return &albSdk.UpdateLoadBalancerPayload{
		DisableTargetSecurityGroupAssignment: lb.DisableTargetSecurityGroupAssignment,
		ExternalAddress:                      lb.ExternalAddress,
		Labels:                               lb.Labels,
		Listeners:                            lb.Listeners,
		Name:                                 lb.Name,
		Networks:                             lb.Networks,
		Options:                              lb.Options,
		PlanId:                               lb.PlanId,
		TargetPools:                          lb.TargetPools,
		Version:                              lb.Version,
	}
}

func fixtureApplicationLoadBalancer(explicitBool *bool, mods ...func(m *albSdk.LoadBalancer)) *albSdk.LoadBalancer {
	resp := &albSdk.LoadBalancer{
		DisableTargetSecurityGroupAssignment: explicitBool,
		ExternalAddress:                      utils.Ptr(externalAddress),
		Errors: utils.Ptr([]albSdk.LoadBalancerError{
			{
				Description: utils.Ptr("quota test error"),
				Type:        utils.Ptr(albSdk.LOADBALANCERERRORTYPE_QUOTA_SECGROUP_EXCEEDED),
			},
			{
				Description: utils.Ptr("fip test error"),
				Type:        utils.Ptr(albSdk.LOADBALANCERERRORTYPE_FIP_NOT_CONFIGURED),
			},
		}),
		Name:           utils.Ptr(lbName),
		PlanId:         utils.Ptr("p10"),
		PrivateAddress: utils.Ptr("10.1.11.0"),
		Region:         utils.Ptr(region),
		Status:         utils.Ptr(albSdk.LoadBalancerStatus("STATUS_READY")),
		Version:        utils.Ptr(lbVersion),
		Labels: &map[string]string{
			"key":  "value",
			"key2": "value2",
		},
		Networks: &[]albSdk.Network{
			{
				NetworkId: utils.Ptr("c7c92cc1-a6bd-4e15-a129-b6e2b9899bbc"),
				Role:      utils.Ptr(albSdk.NetworkRole("ROLE_LISTENERS")),
			},
			{
				NetworkId: utils.Ptr("ed3f1822-ca1c-4969-bea6-74c6b3e9aa40"),
				Role:      utils.Ptr(albSdk.NetworkRole("ROLE_TARGETS")),
			},
		},
		Listeners: &[]albSdk.Listener{
			{
				Name:     utils.Ptr("http-80"),
				Port:     utils.Ptr(int64(80)),
				Protocol: utils.Ptr(albSdk.ListenerProtocol("PROTOCOL_HTTP")),
				Http: &albSdk.ProtocolOptionsHTTP{
					Hosts: &[]albSdk.HostConfig{
						{
							Host: utils.Ptr("*"),
							Rules: &[]albSdk.Rule{
								{
									TargetPool: utils.Ptr(targetPoolName),
									WebSocket:  explicitBool,
									Path: &albSdk.Path{
										Prefix: utils.Ptr("/"),
									},
									Headers: &[]albSdk.HttpHeader{
										{Name: utils.Ptr("a-header"), ExactMatch: utils.Ptr("value")},
									},
									QueryParameters: &[]albSdk.QueryParameter{
										{Name: utils.Ptr("a_query_parameter"), ExactMatch: utils.Ptr("value")},
									},
									CookiePersistence: &albSdk.CookiePersistence{
										Name: utils.Ptr("cookie_name"),
										Ttl:  utils.Ptr("3s"),
									},
								},
							},
						},
					},
				},
				Https: &albSdk.ProtocolOptionsHTTPS{
					CertificateConfig: utils.Ptr(albSdk.CertificateConfig{
						CertificateIds: &[]string{
							credentialsRef,
						},
					}),
				},
				WafConfigName: utils.Ptr("my-waf-config"),
			},
		},
		TargetPools: &[]albSdk.TargetPool{
			{
				Name:       utils.Ptr(targetPoolName),
				TargetPort: utils.Ptr(int64(80)),
				Targets: &[]albSdk.Target{
					{
						DisplayName: utils.Ptr("test-backend-server"),
						Ip:          utils.Ptr("192.168.0.218"),
					},
				},
				TlsConfig: &albSdk.TargetPoolTlsConfig{
					Enabled:                   explicitBool,
					SkipCertificateValidation: explicitBool,
					CustomCa:                  utils.Ptr("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURDekNDQWZPZ0F3SUJBZ0lVVHlQc1RXQzlseTdvK3dORlltMHV1MStQOElFd0RRWUpLb1pJaHZjTkFRRUwKQlFBd0ZURVRNQkVHQTFVRUF3d0tUWGxEZFhOMGIyMURRVEFlRncweU5UQXlNVGt4T1RJME1qQmFGdzB5TmpBeQpNVGt4T1RJME1qQmFNQlV4RXpBUkJnTlZCQU1NQ2sxNVEzVnpkRzl0UTBFd2dnRWlNQTBHQ1NxR1NJYjNEUUVCCkFRVUFBNElCRHdBd2dnRUtBb0lCQVFDUU1FWUtiaU54VTM3ZkV3Qk94a3ZDc2hCUiswTXd4d0xXOE1pMy9wdm8KbjNodXhqY203RWFLVzlyN2tJYW9IWGJUUzF0bk82ckhBSEtCRHh6dW9ZRDdDMlNNU2lMeGRkcXVOUnZwa0xhUAo4cUFYbmVRWTJWUDdMenNBZ3NDMDRQS0cwWUMxTmdGNXNKR3NpV0lSR0ltK2NzWUxuUE1ud2FBR3g0SXZZNm1ICkFtTTY0YjZRUkNnMzZMSytQNk45S1R2U1FMdnZtRmRrQTJzRFRvQ21OL0FtcDZ4TkRGcSthUUdMd2RRUXFIRFAKVGFVcVBtRXlpRkhLdkZVYUZNTlFWazhCMU9tOEFTbzY5bThVM0VhdDRaT1ZXMXRpdEUzOTNRa09kQTZaeXBNQwpySkpwZU5OTExKcTNtSU9XT2Q3R0V5QXZqVWZtSndHaHFFRlM3bE1HNjdobkFnTUJBQUdqVXpCUk1CMEdBMVVkCkRnUVdCQlNrL0lNNWphT0FKTDMvS255cTNjVnZhMDRZWkRBZkJnTlZIU01FR0RBV2dCU2svSU01amFPQUpMMy8KS255cTNjVnZhMDRZWkRBUEJnTlZIUk1CQWY4RUJUQURBUUgvTUEwR0NTcUdTSWIzRFFFQkN3VUFBNElCQVFCZQpaL21FOHJOSWJOYkhRZXAvVnBwc2hhWlV6Z2R5NG5zbWgwd3Z4TXVISVFQMEtIcnhMQ2toT243QTlmdTRtWS9QClErOFFxbG5qVHNNNGNxaXVGY2Q1VjFOazlWRi9lNVgzSFhDREhoL2pCRncrTzVUR1ZBUi83REJ3MzFsWXYvTHQKSGFra2pRQ2Rhd3V2SDNvc08vVWtFbE0vaTJLQytpWUJhdlRlbm05N0FSN1dHZ1cxNS9NSXF4TmFZRStuSnRoLwpkY1ZEMGI1cVN1WVFhRW1aM0N6TVVpMTg4UitnbzVvekNmMmNPYWErMy9MRVlBYUkzdktpU0U4S1Rzc2h5b0ttCk82WVpxclZ4UUNXQ0RUT3NkMjhrN2xIdDh3SitqelljakN1NjBEVXBnMVpwWStabm1yRTh2UFBEYi96WGhCbjYKL2xsWFRXT1VqbXVUS25Hc0lEUDUKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQ=="),
				},
				ActiveHealthCheck: &albSdk.ActiveHealthCheck{
					HealthyThreshold:   utils.Ptr(int64(1)),
					UnhealthyThreshold: utils.Ptr(int64(5)),
					Interval:           utils.Ptr("2s"),
					IntervalJitter:     utils.Ptr("3s"),
					Timeout:            utils.Ptr("4s"),
					HttpHealthChecks: &albSdk.HttpHealthChecks{
						Path:       utils.Ptr("/health"),
						OkStatuses: &[]string{"200", "201"},
					},
				},
			},
		},
		Options: utils.Ptr(albSdk.LoadBalancerOptions{
			EphemeralAddress:   explicitBool,
			PrivateNetworkOnly: explicitBool,
			Observability: &albSdk.LoadbalancerOptionObservability{
				Logs: &albSdk.LoadbalancerOptionLogs{
					CredentialsRef: utils.Ptr(credentialsRef),
					PushUrl:        utils.Ptr("http://www.example.org/push"),
				},
				Metrics: &albSdk.LoadbalancerOptionMetrics{
					CredentialsRef: utils.Ptr(credentialsRef),
					PushUrl:        utils.Ptr("http://www.example.org/pull"),
				},
			},
			AccessControl: &albSdk.LoadbalancerOptionAccessControl{
				AllowedSourceRanges: &[]string{"192.168.0.0", "192.168.0.1"},
			},
		}),
		LoadBalancerSecurityGroup: &albSdk.CreateLoadBalancerPayloadLoadBalancerSecurityGroup{
			Id:   utils.Ptr(sgLBID),
			Name: utils.Ptr("loadbalancer/" + lbName + "/backend-port"),
		},
		TargetSecurityGroup: &albSdk.CreateLoadBalancerPayloadTargetSecurityGroup{
			Id:   utils.Ptr(sgTargetID),
			Name: utils.Ptr("loadbalancer/" + lbName + "/backend"),
		},
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *albSdk.CreateLoadBalancerPayload
		isValid     bool
	}{
		{
			description: "valid",
			input:       fixtureModel(nil),
			expected:    fixtureCreatePayload(fixtureApplicationLoadBalancer(nil)),
			isValid:     true,
		},
		{
			description: "valid empty",
			input:       fixtureModelNull(),
			expected:    &albSdk.CreateLoadBalancerPayload{},
			isValid:     true,
		},
		{
			description: "model nil",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input)
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

func TestToTargetPoolUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *albSdk.UpdateLoadBalancerPayload
		isValid     bool
	}{
		{
			description: "valid",
			input:       fixtureModel(nil),
			expected:    fixtureUpdatePayload(fixtureApplicationLoadBalancer(nil)),
			isValid:     true,
		},
		{
			description: "valid empty",
			input:       fixtureModelNull(),
			expected:    &albSdk.UpdateLoadBalancerPayload{},
			isValid:     true,
		},
		{
			description: "model nil",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input)
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

func TestMapFields(t *testing.T) {
	const testRegion = "eu01"
	tests := []struct {
		description             string
		input                   *albSdk.LoadBalancer
		output                  *Model
		modelPrivateNetworkOnly *bool
		region                  string
		expected                *Model
		isValid                 bool
	}{
		{
			description: "valid full model",
			input:       fixtureApplicationLoadBalancer(nil),
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region:   testRegion,
			expected: fixtureModel(utils.Ptr(false)),
			isValid:  true,
		},
		{
			description: "error alb nil",
			input:       nil,
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region:   testRegion,
			expected: fixtureModel(nil),
			isValid:  false,
		},
		{
			description: "error model nil",
			input:       fixtureApplicationLoadBalancer(nil),
			output:      nil,
			region:      testRegion,
			expected:    fixtureModel(nil),
			isValid:     false,
		},
		{
			description: "error no name",
			input: fixtureApplicationLoadBalancer(nil, func(m *albSdk.LoadBalancer) {
				m.Name = nil
			}),
			output: &Model{
				ProjectId: types.StringValue(projectID),
				Name:      types.StringValue(""),
			},
			region:   testRegion,
			expected: fixtureModel(nil),
			isValid:  false,
		},
		{
			description: "valid name in model",
			input:       fixtureApplicationLoadBalancer(nil),
			output: &Model{
				ProjectId: types.StringValue(projectID),
				Name:      types.StringValue(lbName),
			},
			region:   testRegion,
			expected: fixtureModel(utils.Ptr(false)),
			isValid:  true,
		},
		{
			description: "false - explicitly set",
			input:       fixtureApplicationLoadBalancer(utils.Ptr(false)),
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region:   testRegion,
			expected: fixtureModel(utils.Ptr(false)),
			isValid:  true,
		},
		{
			description: "true - explicitly set",
			input:       fixtureApplicationLoadBalancer(utils.Ptr(true)),
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region:   testRegion,
			expected: fixtureModel(utils.Ptr(true)),
			isValid:  true,
		},
		{
			description: "false - only in model set",
			input:       fixtureApplicationLoadBalancer(nil),
			output:      fixtureModel(utils.Ptr(false)),
			region:      testRegion,
			expected:    fixtureModel(utils.Ptr(false)),
			isValid:     true,
		},
		{
			description: "true - only in model set",
			input:       fixtureApplicationLoadBalancer(nil),
			output:      fixtureModel(utils.Ptr(true)),
			region:      testRegion,
			expected:    fixtureModel(utils.Ptr(false)),
			isValid:     true,
		},
		{
			description: "valid empty",
			input:       &albSdk.LoadBalancer{},
			output: &Model{
				ProjectId: types.StringValue(projectID),
				Name:      types.StringValue(lbName),
			},
			region: testRegion,
			expected: fixtureModelNull(func(m *Model) {
				m.Id = types.StringValue(strings.Join([]string{projectID, region, lbName}, ","))
				m.ProjectId = types.StringValue(projectID)
				m.Name = types.StringValue(lbName)
				m.Region = types.StringValue(region)
				m.DisableSecurityGroupAssignment = types.BoolValue(false)
			}),
			isValid: true,
		},
		{
			description: "mapTargets no response",
			input: fixtureApplicationLoadBalancer(nil, func(m *albSdk.LoadBalancer) {
				m.TargetPools = &[]albSdk.TargetPool{
					{ // empty target pool
						ActiveHealthCheck: nil,
						Name:              nil,
						TargetPort:        nil,
						Targets:           nil,
						TlsConfig:         nil,
					},
				}
			}),
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region: testRegion,
			expected: fixtureModel(utils.Ptr(false), func(m *Model) {
				m.TargetPools = types.ListValueMust(
					types.ObjectType{AttrTypes: targetPoolTypes},
					[]attr.Value{
						types.ObjectValueMust(
							targetPoolTypes,
							map[string]attr.Value{
								"name":                types.StringNull(),
								"target_port":         types.Int64Null(),
								"targets":             types.SetNull(types.ObjectType{AttrTypes: targetTypes}),
								"tls_config":          types.ObjectNull(tlsConfigTypes),
								"active_health_check": types.ObjectNull(activeHealthCheckTypes),
							},
						),
					},
				)
			}),
			isValid: true,
		},
		{
			description: "mapHttpHealthChecks no response",
			input: fixtureApplicationLoadBalancer(nil, func(m *albSdk.LoadBalancer) {
				m.TargetPools = &[]albSdk.TargetPool{
					{
						Name:       utils.Ptr(targetPoolName),
						TargetPort: utils.Ptr(int64(80)),
						Targets: &[]albSdk.Target{
							{
								DisplayName: utils.Ptr("test-backend-server"),
								Ip:          utils.Ptr("192.168.0.218"),
							},
						},
						ActiveHealthCheck: &albSdk.ActiveHealthCheck{
							HealthyThreshold:   utils.Ptr(int64(1)),
							UnhealthyThreshold: utils.Ptr(int64(5)),
							Interval:           utils.Ptr("2s"),
							IntervalJitter:     utils.Ptr("3s"),
							Timeout:            utils.Ptr("4s"),
						},
					},
				}
			}),
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region: testRegion,
			expected: fixtureModel(utils.Ptr(false), func(m *Model) {
				m.TargetPools = types.ListValueMust(
					types.ObjectType{AttrTypes: targetPoolTypes},
					[]attr.Value{
						types.ObjectValueMust(
							targetPoolTypes,
							map[string]attr.Value{
								"name":        types.StringValue(targetPoolName),
								"target_port": types.Int64Value(80),
								"targets": types.SetValueMust(
									types.ObjectType{AttrTypes: targetTypes},
									[]attr.Value{
										types.ObjectValueMust(
											targetTypes,
											map[string]attr.Value{
												"display_name": types.StringValue("test-backend-server"),
												"ip":           types.StringValue("192.168.0.218"),
											},
										),
									},
								),
								"tls_config": types.ObjectNull(tlsConfigTypes),
								"active_health_check": types.ObjectValueMust(
									activeHealthCheckTypes,
									map[string]attr.Value{
										"healthy_threshold":   types.Int64Value(1),
										"interval":            types.StringValue("2s"),
										"interval_jitter":     types.StringValue("3s"),
										"timeout":             types.StringValue("4s"),
										"unhealthy_threshold": types.Int64Value(5),
										"http_health_checks":  types.ObjectNull(httpHealthChecksTypes),
									},
								),
							},
						),
					},
				)
			}),
			isValid: true,
		},
		{
			description: "mapOptions no response",
			input: fixtureApplicationLoadBalancer(nil, func(m *albSdk.LoadBalancer) {
				m.Options = &albSdk.LoadBalancerOptions{
					AccessControl:      nil,
					Observability:      nil,
					EphemeralAddress:   nil,
					PrivateNetworkOnly: nil,
				}
			}),
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region: testRegion,
			expected: fixtureModel(utils.Ptr(false), func(m *Model) {
				m.Options = types.ObjectNull(optionsTypes)
			}),
			isValid: true,
		},
		{
			description: "mapCertificates no response",
			input: fixtureApplicationLoadBalancer(nil, func(m *albSdk.LoadBalancer) {
				m.Listeners = &[]albSdk.Listener{
					{
						Name:     utils.Ptr("http-80"),
						Port:     utils.Ptr(int64(80)),
						Protocol: utils.Ptr(albSdk.ListenerProtocol("PROTOCOL_HTTP")),
						Http: &albSdk.ProtocolOptionsHTTP{
							Hosts: &[]albSdk.HostConfig{
								{
									Host: utils.Ptr("*"),
									Rules: &[]albSdk.Rule{
										{
											TargetPool: utils.Ptr(targetPoolName),
											WebSocket:  nil,
											Path: &albSdk.Path{
												Prefix: utils.Ptr("/"),
											},
											Headers: &[]albSdk.HttpHeader{
												{Name: utils.Ptr("a-header"), ExactMatch: utils.Ptr("value")},
											},
											QueryParameters: &[]albSdk.QueryParameter{
												{Name: utils.Ptr("a_query_parameter"), ExactMatch: utils.Ptr("value")},
											},
											CookiePersistence: &albSdk.CookiePersistence{
												Name: utils.Ptr("cookie_name"),
												Ttl:  utils.Ptr("3s"),
											},
										},
									},
								},
							},
						},
						Https: &albSdk.ProtocolOptionsHTTPS{
							CertificateConfig: nil,
						},
						WafConfigName: utils.Ptr("my-waf-config"),
					},
				}
			}),
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region: testRegion,
			expected: fixtureModel(utils.Ptr(false), func(m *Model) {
				m.Listeners = types.ListValueMust(
					types.ObjectType{AttrTypes: listenerTypes},
					[]attr.Value{
						types.ObjectValueMust(
							listenerTypes,
							map[string]attr.Value{
								"name":            types.StringValue("http-80"),
								"port":            types.Int64Value(80),
								"protocol":        types.StringValue("PROTOCOL_HTTP"),
								"waf_config_name": types.StringValue("my-waf-config"),
								"http": types.ObjectValueMust(
									httpTypes,
									map[string]attr.Value{
										"hosts": types.ListValueMust(
											types.ObjectType{AttrTypes: hostConfigTypes},
											[]attr.Value{types.ObjectValueMust(
												hostConfigTypes,
												map[string]attr.Value{
													"host": types.StringValue("*"),
													"rules": types.ListValueMust(
														types.ObjectType{AttrTypes: ruleTypes},
														[]attr.Value{types.ObjectValueMust(
															ruleTypes,
															map[string]attr.Value{
																"target_pool": types.StringValue(targetPoolName),
																"web_socket":  types.BoolValue(false),
																"path": types.ObjectValueMust(pathTypes,
																	map[string]attr.Value{
																		"exact_match": types.StringNull(),
																		"prefix":      types.StringValue("/"),
																	},
																),
																"headers": types.SetValueMust(
																	types.ObjectType{AttrTypes: headersTypes},
																	[]attr.Value{types.ObjectValueMust(
																		headersTypes,
																		map[string]attr.Value{
																			"name":        types.StringValue("a-header"),
																			"exact_match": types.StringValue("value"),
																		}),
																	},
																),
																"query_parameters": types.SetValueMust(
																	types.ObjectType{AttrTypes: queryParameterTypes},
																	[]attr.Value{types.ObjectValueMust(
																		queryParameterTypes,
																		map[string]attr.Value{
																			"name":        types.StringValue("a_query_parameter"),
																			"exact_match": types.StringValue("value"),
																		}),
																	},
																),
																"cookie_persistence": types.ObjectValueMust(
																	cookiePersistenceTypes,
																	map[string]attr.Value{
																		"name": types.StringValue("cookie_name"),
																		"ttl":  types.StringValue("3s"),
																	},
																),
															},
														),
														}),
												},
											),
											}),
									},
								),
								"https": types.ObjectValueMust(
									httpsTypes,
									map[string]attr.Value{
										"certificate_config": types.ObjectNull(certificateConfigTypes),
									},
								),
							},
						),
					},
				)
			}),
			isValid: true,
		},
		{
			description: "mapHttps no response",
			input: fixtureApplicationLoadBalancer(nil, func(m *albSdk.LoadBalancer) {
				m.Listeners = &[]albSdk.Listener{
					{
						Name:     utils.Ptr("http-80"),
						Port:     utils.Ptr(int64(80)),
						Protocol: utils.Ptr(albSdk.ListenerProtocol("PROTOCOL_HTTP")),
						Http: &albSdk.ProtocolOptionsHTTP{
							Hosts: &[]albSdk.HostConfig{
								{
									Host: utils.Ptr("*"),
									Rules: &[]albSdk.Rule{
										{
											TargetPool: utils.Ptr(targetPoolName),
											WebSocket:  nil,
											Path: &albSdk.Path{
												Prefix: utils.Ptr("/"),
											},
											Headers: &[]albSdk.HttpHeader{
												{Name: utils.Ptr("a-header"), ExactMatch: utils.Ptr("value")},
											},
											QueryParameters: &[]albSdk.QueryParameter{
												{Name: utils.Ptr("a_query_parameter"), ExactMatch: utils.Ptr("value")},
											},
											CookiePersistence: &albSdk.CookiePersistence{
												Name: utils.Ptr("cookie_name"),
												Ttl:  utils.Ptr("3s"),
											},
										},
									},
								},
							},
						},
						Https:         nil,
						WafConfigName: utils.Ptr("my-waf-config"),
					},
				}
			}),
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region: testRegion,
			expected: fixtureModel(utils.Ptr(false), func(m *Model) {
				m.Listeners = types.ListValueMust(
					types.ObjectType{AttrTypes: listenerTypes},
					[]attr.Value{
						types.ObjectValueMust(
							listenerTypes,
							map[string]attr.Value{
								"name":            types.StringValue("http-80"),
								"port":            types.Int64Value(80),
								"protocol":        types.StringValue("PROTOCOL_HTTP"),
								"waf_config_name": types.StringValue("my-waf-config"),
								"http": types.ObjectValueMust(
									httpTypes,
									map[string]attr.Value{
										"hosts": types.ListValueMust(
											types.ObjectType{AttrTypes: hostConfigTypes},
											[]attr.Value{types.ObjectValueMust(
												hostConfigTypes,
												map[string]attr.Value{
													"host": types.StringValue("*"),
													"rules": types.ListValueMust(
														types.ObjectType{AttrTypes: ruleTypes},
														[]attr.Value{types.ObjectValueMust(
															ruleTypes,
															map[string]attr.Value{
																"target_pool": types.StringValue(targetPoolName),
																"web_socket":  types.BoolValue(false),
																"path": types.ObjectValueMust(pathTypes,
																	map[string]attr.Value{
																		"exact_match": types.StringNull(),
																		"prefix":      types.StringValue("/"),
																	},
																),
																"headers": types.SetValueMust(
																	types.ObjectType{AttrTypes: headersTypes},
																	[]attr.Value{types.ObjectValueMust(
																		headersTypes,
																		map[string]attr.Value{
																			"name":        types.StringValue("a-header"),
																			"exact_match": types.StringValue("value"),
																		}),
																	},
																),
																"query_parameters": types.SetValueMust(
																	types.ObjectType{AttrTypes: queryParameterTypes},
																	[]attr.Value{types.ObjectValueMust(
																		queryParameterTypes,
																		map[string]attr.Value{
																			"name":        types.StringValue("a_query_parameter"),
																			"exact_match": types.StringValue("value"),
																		}),
																	},
																),
																"cookie_persistence": types.ObjectValueMust(
																	cookiePersistenceTypes,
																	map[string]attr.Value{
																		"name": types.StringValue("cookie_name"),
																		"ttl":  types.StringValue("3s"),
																	},
																),
															},
														),
														}),
												},
											),
											}),
									},
								),
								"https": types.ObjectNull(httpsTypes),
							},
						),
					},
				)
			}),
			isValid: true,
		},
		{
			description: "mapRules contents no response",
			input: fixtureApplicationLoadBalancer(nil, func(m *albSdk.LoadBalancer) {
				m.Listeners = &[]albSdk.Listener{
					{
						Name:     utils.Ptr("http-80"),
						Port:     utils.Ptr(int64(80)),
						Protocol: utils.Ptr(albSdk.ListenerProtocol("PROTOCOL_HTTP")),
						Http: &albSdk.ProtocolOptionsHTTP{
							Hosts: &[]albSdk.HostConfig{
								{
									Host: utils.Ptr("*"),
									Rules: &[]albSdk.Rule{
										{
											TargetPool:        utils.Ptr(targetPoolName),
											WebSocket:         nil,
											Path:              nil,
											Headers:           nil,
											QueryParameters:   nil,
											CookiePersistence: nil,
										},
									},
								},
							},
						},
						Https: &albSdk.ProtocolOptionsHTTPS{
							CertificateConfig: utils.Ptr(albSdk.CertificateConfig{
								CertificateIds: &[]string{
									credentialsRef,
								},
							}),
						},
						WafConfigName: utils.Ptr("my-waf-config"),
					},
				}
			}),
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region: testRegion,
			expected: fixtureModel(utils.Ptr(false), func(m *Model) {
				m.Listeners = types.ListValueMust(
					types.ObjectType{AttrTypes: listenerTypes},
					[]attr.Value{
						types.ObjectValueMust(
							listenerTypes,
							map[string]attr.Value{
								"name":            types.StringValue("http-80"),
								"port":            types.Int64Value(80),
								"protocol":        types.StringValue("PROTOCOL_HTTP"),
								"waf_config_name": types.StringValue("my-waf-config"),
								"http": types.ObjectValueMust(
									httpTypes,
									map[string]attr.Value{
										"hosts": types.ListValueMust(
											types.ObjectType{AttrTypes: hostConfigTypes},
											[]attr.Value{types.ObjectValueMust(
												hostConfigTypes,
												map[string]attr.Value{
													"host": types.StringValue("*"),
													"rules": types.ListValueMust(
														types.ObjectType{AttrTypes: ruleTypes},
														[]attr.Value{types.ObjectValueMust(
															ruleTypes,
															map[string]attr.Value{
																"target_pool":        types.StringValue(targetPoolName),
																"web_socket":         types.BoolValue(false),
																"path":               types.ObjectNull(pathTypes),
																"headers":            types.SetNull(types.ObjectType{AttrTypes: headersTypes}),
																"query_parameters":   types.SetNull(types.ObjectType{AttrTypes: queryParameterTypes}),
																"cookie_persistence": types.ObjectNull(cookiePersistenceTypes),
															},
														),
														}),
												},
											),
											}),
									},
								),
								"https": types.ObjectValueMust(
									httpsTypes,
									map[string]attr.Value{
										"certificate_config": types.ObjectValueMust(
											certificateConfigTypes,
											map[string]attr.Value{
												"certificate_ids": types.SetValueMust(
													types.StringType,
													[]attr.Value{
														types.StringValue(credentialsRef),
													},
												),
											},
										),
									},
								),
							},
						),
					},
				)
			}),
			isValid: true,
		},
		{
			description: "mapRules no response",
			input: fixtureApplicationLoadBalancer(nil, func(m *albSdk.LoadBalancer) {
				m.Listeners = &[]albSdk.Listener{
					{
						Name:     utils.Ptr("http-80"),
						Port:     utils.Ptr(int64(80)),
						Protocol: utils.Ptr(albSdk.ListenerProtocol("PROTOCOL_HTTP")),
						Http: &albSdk.ProtocolOptionsHTTP{
							Hosts: &[]albSdk.HostConfig{
								{
									Host:  utils.Ptr("*"),
									Rules: nil,
								},
							},
						},
						Https: &albSdk.ProtocolOptionsHTTPS{
							CertificateConfig: utils.Ptr(albSdk.CertificateConfig{
								CertificateIds: &[]string{
									credentialsRef,
								},
							}),
						},
						WafConfigName: utils.Ptr("my-waf-config"),
					},
				}
			}),
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region: testRegion,
			expected: fixtureModel(utils.Ptr(false), func(m *Model) {
				m.Listeners = types.ListValueMust(
					types.ObjectType{AttrTypes: listenerTypes},
					[]attr.Value{
						types.ObjectValueMust(
							listenerTypes,
							map[string]attr.Value{
								"name":            types.StringValue("http-80"),
								"port":            types.Int64Value(80),
								"protocol":        types.StringValue("PROTOCOL_HTTP"),
								"waf_config_name": types.StringValue("my-waf-config"),
								"http": types.ObjectValueMust(
									httpTypes,
									map[string]attr.Value{
										"hosts": types.ListValueMust(
											types.ObjectType{AttrTypes: hostConfigTypes},
											[]attr.Value{types.ObjectValueMust(
												hostConfigTypes,
												map[string]attr.Value{
													"host":  types.StringValue("*"),
													"rules": types.ListNull(types.ObjectType{AttrTypes: ruleTypes}),
												}),
											},
										),
									},
								),
								"https": types.ObjectValueMust(
									httpsTypes,
									map[string]attr.Value{
										"certificate_config": types.ObjectValueMust(
											certificateConfigTypes,
											map[string]attr.Value{
												"certificate_ids": types.SetValueMust(
													types.StringType,
													[]attr.Value{
														types.StringValue(credentialsRef),
													},
												),
											},
										),
									},
								),
							},
						),
					},
				)
			}),
			isValid: true,
		},
		{
			description: "mapHosts no response",
			input: fixtureApplicationLoadBalancer(nil, func(m *albSdk.LoadBalancer) {
				m.Listeners = &[]albSdk.Listener{
					{
						Name:     utils.Ptr("http-80"),
						Port:     utils.Ptr(int64(80)),
						Protocol: utils.Ptr(albSdk.ListenerProtocol("PROTOCOL_HTTP")),
						Http: &albSdk.ProtocolOptionsHTTP{
							Hosts: nil,
						},
						Https: &albSdk.ProtocolOptionsHTTPS{
							CertificateConfig: utils.Ptr(albSdk.CertificateConfig{
								CertificateIds: &[]string{
									credentialsRef,
								},
							}),
						},
						WafConfigName: utils.Ptr("my-waf-config"),
					},
				}
			}),
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region: testRegion,
			expected: fixtureModel(utils.Ptr(false), func(m *Model) {
				m.Listeners = types.ListValueMust(
					types.ObjectType{AttrTypes: listenerTypes},
					[]attr.Value{
						types.ObjectValueMust(
							listenerTypes,
							map[string]attr.Value{
								"name":            types.StringValue("http-80"),
								"port":            types.Int64Value(80),
								"protocol":        types.StringValue("PROTOCOL_HTTP"),
								"waf_config_name": types.StringValue("my-waf-config"),
								"http": types.ObjectValueMust(
									httpTypes,
									map[string]attr.Value{
										"hosts": types.ListNull(types.ObjectType{AttrTypes: hostConfigTypes}),
									},
								),
								"https": types.ObjectValueMust(
									httpsTypes,
									map[string]attr.Value{
										"certificate_config": types.ObjectValueMust(
											certificateConfigTypes,
											map[string]attr.Value{
												"certificate_ids": types.SetValueMust(
													types.StringType,
													[]attr.Value{
														types.StringValue(credentialsRef),
													},
												),
											},
										),
									},
								),
							},
						),
					},
				)
			}),
			isValid: true,
		},
		{
			description: "mapHttp no response",
			input: fixtureApplicationLoadBalancer(nil, func(m *albSdk.LoadBalancer) {
				m.Listeners = &[]albSdk.Listener{
					{
						Name:     utils.Ptr("http-80"),
						Port:     utils.Ptr(int64(80)),
						Protocol: utils.Ptr(albSdk.ListenerProtocol("PROTOCOL_HTTP")),
						Http:     nil,
						Https: &albSdk.ProtocolOptionsHTTPS{
							CertificateConfig: utils.Ptr(albSdk.CertificateConfig{
								CertificateIds: &[]string{
									credentialsRef,
								},
							}),
						},
						WafConfigName: utils.Ptr("my-waf-config"),
					},
				}
			}),
			output: &Model{
				ProjectId: types.StringValue(projectID),
			},
			region: testRegion,
			expected: fixtureModel(utils.Ptr(false), func(m *Model) {
				m.Listeners = types.ListValueMust(
					types.ObjectType{AttrTypes: listenerTypes},
					[]attr.Value{
						types.ObjectValueMust(
							listenerTypes,
							map[string]attr.Value{
								"name":            types.StringValue("http-80"),
								"port":            types.Int64Value(80),
								"protocol":        types.StringValue("PROTOCOL_HTTP"),
								"waf_config_name": types.StringValue("my-waf-config"),
								"http":            types.ObjectNull(httpTypes),
								"https": types.ObjectValueMust(
									httpsTypes,
									map[string]attr.Value{
										"certificate_config": types.ObjectValueMust(
											certificateConfigTypes,
											map[string]attr.Value{
												"certificate_ids": types.SetValueMust(
													types.StringType,
													[]attr.Value{
														types.StringValue(credentialsRef),
													},
												),
											},
										),
									},
								),
							},
						),
					},
				)
			}),
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, tt.output, tt.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func Test_toExternalAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    *Model
		expected albSdk.UpdateLoadBalancerPayloadGetExternalAddressAttributeType
		isValid  bool
	}{
		{
			name: "valid",
			input: &Model{
				ExternalAddress: types.StringValue(externalAddress),
			},
			expected: utils.Ptr(externalAddress),
			isValid:  true,
		},
		{
			name: "valid with option",
			input: &Model{
				ExternalAddress: types.StringValue(externalAddress),
				Options: types.ObjectValueMust(optionsTypes,
					map[string]attr.Value{
						"access_control":       types.ObjectNull(accessControlTypes),
						"observability":        types.ObjectNull(observabilityTypes),
						"private_network_only": types.BoolNull(),
						"ephemeral_address":    types.BoolNull(),
					}),
			},
			expected: utils.Ptr(externalAddress),
			isValid:  true,
		},
		{
			name: "invalid with option and both true",
			input: &Model{
				ExternalAddress: types.StringValue(externalAddress),
				Options: types.ObjectValueMust(optionsTypes,
					map[string]attr.Value{
						"access_control":       types.ObjectNull(accessControlTypes),
						"observability":        types.ObjectNull(observabilityTypes),
						"private_network_only": types.BoolValue(true),
						"ephemeral_address":    types.BoolValue(true),
					}),
			},
			expected: nil,
			isValid:  false,
		},
		{
			name: "valid with option and both false",
			input: &Model{
				ExternalAddress: types.StringValue(externalAddress),
				Options: types.ObjectValueMust(optionsTypes,
					map[string]attr.Value{
						"access_control":       types.ObjectNull(accessControlTypes),
						"observability":        types.ObjectNull(observabilityTypes),
						"private_network_only": types.BoolValue(false),
						"ephemeral_address":    types.BoolValue(false),
					}),
			},
			expected: utils.Ptr(externalAddress),
			isValid:  true,
		},
		{
			name: "valid with option and ephemeral address",
			input: &Model{
				ExternalAddress: types.StringValue(externalAddress),
				Options: types.ObjectValueMust(optionsTypes,
					map[string]attr.Value{
						"access_control":       types.ObjectNull(accessControlTypes),
						"observability":        types.ObjectNull(observabilityTypes),
						"private_network_only": types.BoolNull(),
						"ephemeral_address":    types.BoolValue(true),
					}),
			},
			expected: nil,
			isValid:  true,
		},
		{
			name: "valid with option and private network only",
			input: &Model{
				ExternalAddress: types.StringValue(externalAddress),
				Options: types.ObjectValueMust(optionsTypes,
					map[string]attr.Value{
						"access_control":       types.ObjectNull(accessControlTypes),
						"observability":        types.ObjectNull(observabilityTypes),
						"private_network_only": types.BoolValue(true),
						"ephemeral_address":    types.BoolNull(),
					}),
			},
			expected: nil,
			isValid:  true,
		},
		{
			name: "valid with option and null false",
			input: &Model{
				ExternalAddress: types.StringValue(externalAddress),
				Options: types.ObjectValueMust(optionsTypes,
					map[string]attr.Value{
						"access_control":       types.ObjectNull(accessControlTypes),
						"observability":        types.ObjectNull(observabilityTypes),
						"private_network_only": types.BoolNull(),
						"ephemeral_address":    types.BoolValue(false),
					}),
			},
			expected: utils.Ptr(externalAddress),
			isValid:  true,
		},
		{
			name: "valid with option and false null",
			input: &Model{
				ExternalAddress: types.StringValue(externalAddress),
				Options: types.ObjectValueMust(optionsTypes,
					map[string]attr.Value{
						"access_control":       types.ObjectNull(accessControlTypes),
						"observability":        types.ObjectNull(observabilityTypes),
						"private_network_only": types.BoolValue(false),
						"ephemeral_address":    types.BoolNull(),
					}),
			},
			expected: utils.Ptr(externalAddress),
			isValid:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toExternalAddress(context.Background(), tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(got, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func Test_toPathPayload(t *testing.T) {
	tests := []struct {
		name     string
		input    *rule
		expected albSdk.RuleGetPathAttributeType
		isValid  bool
	}{
		{
			name: "valid prefix",
			input: &rule{
				Path: types.ObjectValueMust(pathTypes,
					map[string]attr.Value{
						"exact_match": types.StringNull(),
						"prefix":      types.StringValue("/"),
					}),
			},
			expected: &albSdk.Path{
				ExactMatch: nil,
				Prefix:     utils.Ptr("/"),
			},
			isValid: true,
		},
		{
			name: "valid exact",
			input: &rule{
				Path: types.ObjectValueMust(pathTypes,
					map[string]attr.Value{
						"exact_match": types.StringValue("exact-match"),
						"prefix":      types.StringNull(),
					}),
			},
			expected: &albSdk.Path{
				ExactMatch: utils.Ptr("exact-match"),
				Prefix:     nil,
			},
			isValid: true,
		},
		{
			name: "valid none set",
			input: &rule{
				Path: types.ObjectValueMust(pathTypes,
					map[string]attr.Value{
						"exact_match": types.StringNull(),
						"prefix":      types.StringNull(),
					}),
			},
			expected: nil,
			isValid:  false,
		},
		{
			name: "valid both set",
			input: &rule{
				Path: types.ObjectValueMust(pathTypes,
					map[string]attr.Value{
						"exact_match": types.StringValue("exact-match"),
						"prefix":      types.StringValue("/"),
					}),
			},
			expected: nil,
			isValid:  false,
		},
		{
			name:     "input path nil",
			input:    &rule{},
			expected: nil,
			isValid:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toPathPayload(context.Background(), tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(got, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
