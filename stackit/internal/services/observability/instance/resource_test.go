package observability

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	observabilitySdk "github.com/stackitcloud/stackit-sdk-go/services/observability/v1api"
)

func fixtureEmailConfigsModel() basetypes.ListValue {
	return types.ListValueMust(types.ObjectType{AttrTypes: emailConfigsTypes}, []attr.Value{
		types.ObjectValueMust(emailConfigsTypes, map[string]attr.Value{
			"auth_identity": types.StringValue("identity"),
			"auth_password": types.StringValue("password"),
			"auth_username": types.StringValue("username"),
			"from":          types.StringValue("notification@example.com"),
			"send_resolved": types.BoolValue(true),
			"smart_host":    types.StringValue("smtp.example.com"),
			"to":            types.StringValue("me@example.com"),
		}),
	})
}

func fixtureOpsGenieConfigsModel() basetypes.ListValue {
	return types.ListValueMust(types.ObjectType{AttrTypes: opsgenieConfigsTypes}, []attr.Value{
		types.ObjectValueMust(opsgenieConfigsTypes, map[string]attr.Value{
			"api_key":       types.StringValue("key"),
			"tags":          types.StringValue("tag"),
			"api_url":       types.StringValue("ops.example.com"),
			"priority":      types.StringValue("P3"),
			"send_resolved": types.BoolValue(true),
		}),
	})
}

func fixtureWebHooksConfigsModel() basetypes.ListValue {
	return types.ListValueMust(types.ObjectType{AttrTypes: webHooksConfigsTypes}, []attr.Value{
		types.ObjectValueMust(webHooksConfigsTypes, map[string]attr.Value{
			"url":           types.StringValue("http://example.com"),
			"ms_teams":      types.BoolValue(true),
			"google_chat":   types.BoolValue(true),
			"send_resolved": types.BoolValue(true),
		}),
	})
}

func fixtureReceiverModel(emailConfigs, opsGenieConfigs, webHooksConfigs basetypes.ListValue) basetypes.ObjectValue {
	return types.ObjectValueMust(receiversTypes, map[string]attr.Value{
		"name":             types.StringValue("name"),
		"email_configs":    emailConfigs,
		"opsgenie_configs": opsGenieConfigs,
		"webhooks_configs": webHooksConfigs,
	})
}

func fixtureRouteModel() basetypes.ObjectValue {
	return types.ObjectValueMust(mainRouteTypes, map[string]attr.Value{
		"continue": types.BoolValue(false),
		"group_by": types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("label1"),
			types.StringValue("label2"),
		}),
		"group_interval":  types.StringValue("1m"),
		"group_wait":      types.StringValue("1m"),
		"receiver":        types.StringValue("name"),
		"repeat_interval": types.StringValue("1m"),
		// "routes":          types.ListNull(getRouteListType()),
		"routes": types.ListValueMust(getRouteListType(), []attr.Value{
			types.ObjectValueMust(getRouteListType().AttrTypes, map[string]attr.Value{
				"continue": types.BoolValue(false),
				"group_by": types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("label1"),
					types.StringValue("label2"),
				}),
				"group_interval": types.StringValue("1m"),
				"group_wait":     types.StringValue("1m"),
				"match":          types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				"match_regex":    types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
				"matchers": types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("matcher1"),
					types.StringValue("matcher2"),
				}),
				"receiver":        types.StringValue("name"),
				"repeat_interval": types.StringValue("1m"),
			}),
		}),
	})
}

func fixtureNullRouteModel() basetypes.ObjectValue {
	return types.ObjectValueMust(mainRouteTypes, map[string]attr.Value{
		"continue":        types.BoolNull(),
		"group_by":        types.ListNull(types.StringType),
		"group_interval":  types.StringNull(),
		"group_wait":      types.StringNull(),
		"receiver":        types.StringValue(""),
		"repeat_interval": types.StringNull(),
		"routes":          types.ListNull(getRouteListType()),
	})
}

func fixtureGlobalConfigModel() basetypes.ObjectValue {
	return types.ObjectValueMust(globalConfigurationTypes, map[string]attr.Value{
		"opsgenie_api_key":   types.StringValue("key"),
		"opsgenie_api_url":   types.StringValue("ops.example.com"),
		"resolve_timeout":    types.StringValue("1m"),
		"smtp_auth_identity": types.StringValue("identity"),
		"smtp_auth_username": types.StringValue("username"),
		"smtp_auth_password": types.StringValue("password"),
		"smtp_from":          types.StringValue("me@example.com"),
		"smtp_smart_host":    types.StringValue("smtp.example.com:25"),
	})
}

func fixtureNullGlobalConfigModel() basetypes.ObjectValue {
	return types.ObjectValueMust(globalConfigurationTypes, map[string]attr.Value{
		"opsgenie_api_key":   types.StringNull(),
		"opsgenie_api_url":   types.StringNull(),
		"resolve_timeout":    types.StringNull(),
		"smtp_auth_identity": types.StringNull(),
		"smtp_auth_username": types.StringNull(),
		"smtp_auth_password": types.StringNull(),
		"smtp_from":          types.StringNull(),
		"smtp_smart_host":    types.StringNull(),
	})
}

func fixtureEmailConfigsPayload() observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerEmailConfigsInner {
	return observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerEmailConfigsInner{
		AuthIdentity: new("identity"),
		AuthPassword: new("password"),
		AuthUsername: new("username"),
		From:         new("notification@example.com"),
		SendResolved: new(true),
		Smarthost:    new("smtp.example.com"),
		To:           new("me@example.com"),
	}
}

func fixtureOpsGenieConfigsPayload() observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerOpsgenieConfigsInner {
	return observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerOpsgenieConfigsInner{
		ApiKey:       new("key"),
		Tags:         new("tag"),
		ApiUrl:       new("ops.example.com"),
		Priority:     new("P3"),
		SendResolved: new(true),
	}
}

func fixtureWebHooksConfigsPayload() observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerWebHookConfigsInner {
	return observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerWebHookConfigsInner{
		Url:          new("http://example.com"),
		MsTeams:      new(true),
		GoogleChat:   new(true),
		SendResolved: new(true),
	}
}

func fixtureReceiverPayload(emailConfigs []observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerEmailConfigsInner, opsGenieConfigs []observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerOpsgenieConfigsInner, webHooksConfigs []observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerWebHookConfigsInner) observabilitySdk.UpdateAlertConfigsPayloadReceiversInner {
	return observabilitySdk.UpdateAlertConfigsPayloadReceiversInner{
		EmailConfigs:    emailConfigs,
		Name:            "name",
		OpsgenieConfigs: opsGenieConfigs,
		WebHookConfigs:  webHooksConfigs,
	}
}

func fixtureRoutePayload() observabilitySdk.UpdateAlertConfigsPayloadRoute {
	return observabilitySdk.UpdateAlertConfigsPayloadRoute{
		Continue:       new(false),
		GroupBy:        []string{"label1", "label2"},
		GroupInterval:  new("1m"),
		GroupWait:      new("1m"),
		Receiver:       "name",
		RepeatInterval: new("1m"),
		Routes: []observabilitySdk.UpdateAlertConfigsPayloadRouteRoutesInner{
			{
				Continue:       new(false),
				GroupBy:        []string{"label1", "label2"},
				GroupInterval:  new("1m"),
				GroupWait:      new("1m"),
				Match:          map[string]any{"key": "value"},
				MatchRe:        map[string]any{"key": "value"},
				Matchers:       []string{"matcher1", "matcher2"},
				Receiver:       new("name"),
				RepeatInterval: new("1m"),
			},
		},
	}
}

func fixtureGlobalConfigPayload() *observabilitySdk.UpdateAlertConfigsPayloadGlobal {
	return &observabilitySdk.UpdateAlertConfigsPayloadGlobal{
		OpsgenieApiKey:   new("key"),
		OpsgenieApiUrl:   new("ops.example.com"),
		ResolveTimeout:   new("1m"),
		SmtpAuthIdentity: new("identity"),
		SmtpAuthUsername: new("username"),
		SmtpAuthPassword: new("password"),
		SmtpFrom:         new("me@example.com"),
		SmtpSmarthost:    new("smtp.example.com:25"),
	}
}

func fixtureReceiverResponse(emailConfigs []observabilitySdk.EmailConfig, opsGenieConfigs []observabilitySdk.OpsgenieConfig, webhookConfigs []observabilitySdk.WebHook) observabilitySdk.Receivers {
	return observabilitySdk.Receivers{
		Name:            "name",
		EmailConfigs:    emailConfigs,
		OpsgenieConfigs: opsGenieConfigs,
		WebHookConfigs:  webhookConfigs,
	}
}

func fixtureEmailConfigsResponse() observabilitySdk.EmailConfig {
	return observabilitySdk.EmailConfig{
		AuthIdentity: new("identity"),
		AuthPassword: new("password"),
		AuthUsername: new("username"),
		From:         new("notification@example.com"),
		SendResolved: new(true),
		Smarthost:    new("smtp.example.com"),
		To:           "me@example.com",
	}
}

func fixtureOpsGenieConfigsResponse() observabilitySdk.OpsgenieConfig {
	return observabilitySdk.OpsgenieConfig{
		ApiKey:       new("key"),
		Tags:         new("tag"),
		ApiUrl:       new("ops.example.com"),
		Priority:     new("P3"),
		SendResolved: new(true),
	}
}

func fixtureWebHooksConfigsResponse() observabilitySdk.WebHook {
	return observabilitySdk.WebHook{
		Url:          "http://example.com",
		MsTeams:      new(true),
		GoogleChat:   new(true),
		SendResolved: new(true),
	}
}

func fixtureRouteResponse() observabilitySdk.Route {
	return observabilitySdk.Route{
		Continue:       new(false),
		GroupBy:        []string{"label1", "label2"},
		GroupInterval:  new("1m"),
		GroupWait:      new("1m"),
		Match:          &map[string]string{"key": "value"},
		MatchRe:        &map[string]string{"key": "value"},
		Matchers:       []string{"matcher1", "matcher2"},
		Receiver:       "name",
		RepeatInterval: new("1m"),
		Routes: []observabilitySdk.RouteSerializer{
			{
				Continue:       new(false),
				GroupBy:        []string{"label1", "label2"},
				GroupInterval:  new("1m"),
				GroupWait:      new("1m"),
				Match:          &map[string]string{"key": "value"},
				MatchRe:        &map[string]string{"key": "value"},
				Matchers:       []string{"matcher1", "matcher2"},
				Receiver:       "name",
				RepeatInterval: new("1m"),
			},
		},
	}
}

func fixtureGlobalConfigResponse() *observabilitySdk.Global {
	return &observabilitySdk.Global{
		OpsgenieApiKey:   new("key"),
		OpsgenieApiUrl:   new("ops.example.com"),
		ResolveTimeout:   new("1m"),
		SmtpAuthIdentity: new("identity"),
		SmtpAuthUsername: new("username"),
		SmtpAuthPassword: new("password"),
		SmtpFrom:         new("me@example.com"),
		SmtpSmarthost:    new("smtp.example.com:25"),
	}
}

func fixtureRouteAttributeSchema(route *schema.ListNestedAttribute, isDatasource bool) map[string]schema.Attribute {
	attributeMap := map[string]schema.Attribute{
		"continue": schema.BoolAttribute{
			Description: routeDescriptions["continue"],
			Optional:    !isDatasource,
			Computed:    true,
		},
		"group_by": schema.ListAttribute{
			Description: routeDescriptions["group_by"],
			Optional:    !isDatasource,
			Computed:    isDatasource,
			ElementType: types.StringType,
		},
		"group_interval": schema.StringAttribute{
			Description: routeDescriptions["group_interval"],
			Optional:    !isDatasource,
			Computed:    true,
		},
		"group_wait": schema.StringAttribute{
			Description: routeDescriptions["group_wait"],
			Optional:    !isDatasource,
			Computed:    true,
		},
		"match": schema.MapAttribute{
			Description:        routeDescriptions["match"],
			DeprecationMessage: "Use `matchers` in the `routes` instead.",
			Optional:           !isDatasource,
			Computed:           isDatasource,
			ElementType:        types.StringType,
		},
		"match_regex": schema.MapAttribute{
			Description:        routeDescriptions["match_regex"],
			DeprecationMessage: "Use `matchers` in the `routes` instead.",
			Optional:           !isDatasource,
			Computed:           isDatasource,
			ElementType:        types.StringType,
		},
		"matchers": schema.ListAttribute{
			Description: routeDescriptions["matchers"],
			Optional:    !isDatasource,
			Computed:    isDatasource,
			ElementType: types.StringType,
		},
		"receiver": schema.StringAttribute{
			Description: routeDescriptions["receiver"],
			Required:    !isDatasource,
			Computed:    isDatasource,
		},
		"repeat_interval": schema.StringAttribute{
			Description: routeDescriptions["repeat_interval"],
			Optional:    !isDatasource,
			Computed:    true,
		},
	}
	if route != nil {
		attributeMap["routes"] = *route
	}
	return attributeMap
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description             string
		instanceResp            *observabilitySdk.GetInstanceResponse
		listACLResp             *observabilitySdk.ListACLResponse
		getMetricsRetentionResp *observabilitySdk.GetMetricsStorageRetentionResponse
		getLogsRetentionResp    *observabilitySdk.LogsConfigResponse
		getTracesRetentionResp  *observabilitySdk.TracesConfigResponse
		expected                Model
		isValid                 bool
	}{
		{
			"default_ok",
			&observabilitySdk.GetInstanceResponse{
				Id: "iid",
			},
			&observabilitySdk.ListACLResponse{},
			&observabilitySdk.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "7d",
			},
			&observabilitySdk.LogsConfigResponse{Config: observabilitySdk.LogsConfig{Retention: "168h"}},
			&observabilitySdk.TracesConfigResponse{Config: observabilitySdk.TraceConfig{Retention: "168h"}},
			Model{
				Id:                                 types.StringValue("pid,iid"),
				ProjectId:                          types.StringValue("pid"),
				InstanceId:                         types.StringValue("iid"),
				PlanId:                             types.StringValue(""),
				PlanName:                           types.StringValue(""),
				Name:                               types.StringNull(),
				Parameters:                         types.MapNull(types.StringType),
				ACL:                                types.SetNull(types.StringType),
				TracesRetentionDays:                types.Int64Value(7),
				LogsRetentionDays:                  types.Int64Value(7),
				MetricsRetentionDays:               types.Int32Value(60),
				MetricsRetentionDays1hDownsampling: types.Int32Value(30),
				MetricsRetentionDays5mDownsampling: types.Int32Value(7),
				DashboardURL:                       types.StringValue(""),
				GrafanaURL:                         types.StringValue(""),
				GrafanaPublicReadAccess:            types.BoolValue(false),
				GrafanaAdminEnabled:                types.BoolValue(false),
				MetricsURL:                         types.StringValue(""),
				MetricsPushURL:                     types.StringValue(""),
				TargetsURL:                         types.StringValue(""),
				AlertingURL:                        types.StringValue(""),
				LogsURL:                            types.StringValue(""),
				LogsPushURL:                        types.StringValue(""),
				JaegerTracesURL:                    types.StringValue(""),
				JaegerUIURL:                        types.StringValue(""),
				OtlpGRPCTracesURL:                  types.StringValue(""),
				OtlpHTTPLogsURL:                    types.StringValue(""),
				OtlpHTTPTracesURL:                  types.StringValue(""),
				OtlpTracesURL:                      types.StringValue(""),
				ZipkinSpansURL:                     types.StringValue(""),
			},
			true,
		},
		{
			"values_ok",
			&observabilitySdk.GetInstanceResponse{
				Id:         "iid",
				Name:       new("name"),
				PlanName:   "plan1",
				PlanId:     "planId",
				Parameters: &map[string]string{"key": "value"},
				Instance: observabilitySdk.InstanceSensitiveData{
					MetricsRetentionTimeRaw: int32(60),
					MetricsRetentionTime1h:  int32(30),
					MetricsRetentionTime5m:  int32(7),
					OtlpTracesUrl:           "otlp_traces",
					OtlpGrpcTracesUrl:       "otlp_grpc_traces",
					OtlpHttpTracesUrl:       "otlp_http_traces",
					OtlpHttpLogsUrl:         "otlp_http_logs",
				},
			},
			&observabilitySdk.ListACLResponse{
				Acl: []string{
					"1.1.1.1/32",
				},
				Message: "message",
			},
			&observabilitySdk.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "7d",
			},
			&observabilitySdk.LogsConfigResponse{Config: observabilitySdk.LogsConfig{Retention: "168h"}},
			&observabilitySdk.TracesConfigResponse{Config: observabilitySdk.TraceConfig{Retention: "168h"}},
			Model{
				Id:         types.StringValue("pid,iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				InstanceId: types.StringValue("iid"),
				PlanId:     types.StringValue("planId"),
				PlanName:   types.StringValue("plan1"),
				Parameters: toTerraformStringMapMust(context.Background(), map[string]string{"key": "value"}),
				ACL: types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("1.1.1.1/32"),
				}),
				TracesRetentionDays:                types.Int64Value(7),
				LogsRetentionDays:                  types.Int64Value(7),
				MetricsRetentionDays:               types.Int32Value(60),
				MetricsRetentionDays1hDownsampling: types.Int32Value(30),
				MetricsRetentionDays5mDownsampling: types.Int32Value(7),
				OtlpTracesURL:                      types.StringValue("otlp_traces"),
				OtlpGRPCTracesURL:                  types.StringValue("otlp_grpc_traces"),
				OtlpHTTPTracesURL:                  types.StringValue("otlp_http_traces"),
				OtlpHTTPLogsURL:                    types.StringValue("otlp_http_logs"),
				DashboardURL:                       types.StringValue(""),
				GrafanaURL:                         types.StringValue(""),
				GrafanaPublicReadAccess:            types.BoolValue(false),
				GrafanaAdminEnabled:                types.BoolValue(false),
				MetricsURL:                         types.StringValue(""),
				MetricsPushURL:                     types.StringValue(""),
				TargetsURL:                         types.StringValue(""),
				AlertingURL:                        types.StringValue(""),
				LogsURL:                            types.StringValue(""),
				LogsPushURL:                        types.StringValue(""),
				JaegerTracesURL:                    types.StringValue(""),
				JaegerUIURL:                        types.StringValue(""),
				ZipkinSpansURL:                     types.StringValue(""),
			},
			true,
		},
		{
			"values_ok_multiple_acls",
			&observabilitySdk.GetInstanceResponse{
				Id:         "iid",
				Name:       new("name"),
				PlanName:   "plan1",
				PlanId:     "planId",
				Parameters: &map[string]string{"key": "value"},
			},
			&observabilitySdk.ListACLResponse{
				Acl: []string{
					"1.1.1.1/32",
					"8.8.8.8/32",
				},
				Message: "message",
			},
			&observabilitySdk.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "7d",
			},
			&observabilitySdk.LogsConfigResponse{Config: observabilitySdk.LogsConfig{Retention: "168h"}},
			&observabilitySdk.TracesConfigResponse{Config: observabilitySdk.TraceConfig{Retention: "168h"}},
			Model{
				Id:         types.StringValue("pid,iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				InstanceId: types.StringValue("iid"),
				PlanId:     types.StringValue("planId"),
				PlanName:   types.StringValue("plan1"),
				Parameters: toTerraformStringMapMust(context.Background(), map[string]string{"key": "value"}),
				ACL: types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("1.1.1.1/32"),
					types.StringValue("8.8.8.8/32"),
				}),
				TracesRetentionDays:                types.Int64Value(7),
				LogsRetentionDays:                  types.Int64Value(7),
				MetricsRetentionDays:               types.Int32Value(60),
				MetricsRetentionDays1hDownsampling: types.Int32Value(30),
				MetricsRetentionDays5mDownsampling: types.Int32Value(7),
				DashboardURL:                       types.StringValue(""),
				GrafanaURL:                         types.StringValue(""),
				GrafanaPublicReadAccess:            types.BoolValue(false),
				GrafanaAdminEnabled:                types.BoolValue(false),
				MetricsURL:                         types.StringValue(""),
				MetricsPushURL:                     types.StringValue(""),
				TargetsURL:                         types.StringValue(""),
				AlertingURL:                        types.StringValue(""),
				LogsURL:                            types.StringValue(""),
				LogsPushURL:                        types.StringValue(""),
				JaegerTracesURL:                    types.StringValue(""),
				JaegerUIURL:                        types.StringValue(""),
				OtlpGRPCTracesURL:                  types.StringValue(""),
				OtlpHTTPLogsURL:                    types.StringValue(""),
				OtlpHTTPTracesURL:                  types.StringValue(""),
				OtlpTracesURL:                      types.StringValue(""),
				ZipkinSpansURL:                     types.StringValue(""),
			},
			true,
		},
		{
			"nullable_fields_ok",
			&observabilitySdk.GetInstanceResponse{
				Id:   "iid",
				Name: nil,
			},
			&observabilitySdk.ListACLResponse{
				Acl:     []string{},
				Message: "",
			},
			&observabilitySdk.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "7d",
			},
			&observabilitySdk.LogsConfigResponse{Config: observabilitySdk.LogsConfig{Retention: "168h"}},
			&observabilitySdk.TracesConfigResponse{Config: observabilitySdk.TraceConfig{Retention: "168h"}},
			Model{
				Id:                                 types.StringValue("pid,iid"),
				ProjectId:                          types.StringValue("pid"),
				InstanceId:                         types.StringValue("iid"),
				PlanId:                             types.StringValue(""),
				PlanName:                           types.StringValue(""),
				Name:                               types.StringNull(),
				Parameters:                         types.MapNull(types.StringType),
				ACL:                                types.SetNull(types.StringType),
				TracesRetentionDays:                types.Int64Value(7),
				LogsRetentionDays:                  types.Int64Value(7),
				MetricsRetentionDays:               types.Int32Value(60),
				MetricsRetentionDays1hDownsampling: types.Int32Value(30),
				MetricsRetentionDays5mDownsampling: types.Int32Value(7),
				DashboardURL:                       types.StringValue(""),
				GrafanaURL:                         types.StringValue(""),
				GrafanaPublicReadAccess:            types.BoolValue(false),
				GrafanaAdminEnabled:                types.BoolValue(false),
				MetricsURL:                         types.StringValue(""),
				MetricsPushURL:                     types.StringValue(""),
				TargetsURL:                         types.StringValue(""),
				AlertingURL:                        types.StringValue(""),
				LogsURL:                            types.StringValue(""),
				LogsPushURL:                        types.StringValue(""),
				JaegerTracesURL:                    types.StringValue(""),
				JaegerUIURL:                        types.StringValue(""),
				OtlpGRPCTracesURL:                  types.StringValue(""),
				OtlpHTTPLogsURL:                    types.StringValue(""),
				OtlpHTTPTracesURL:                  types.StringValue(""),
				OtlpTracesURL:                      types.StringValue(""),
				ZipkinSpansURL:                     types.StringValue(""),
			},
			true,
		},
		{
			"response_nil_fail",
			nil,
			nil,
			nil,
			nil,
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&observabilitySdk.GetInstanceResponse{},
			nil,
			nil,
			nil,
			nil,
			Model{},
			false,
		},
		{
			"empty metrics retention",
			&observabilitySdk.GetInstanceResponse{
				Id:   "iid",
				Name: nil,
			},
			&observabilitySdk.ListACLResponse{
				Acl:     []string{},
				Message: "",
			},
			&observabilitySdk.GetMetricsStorageRetentionResponse{},
			&observabilitySdk.LogsConfigResponse{},
			&observabilitySdk.TracesConfigResponse{},
			Model{},
			false,
		},
		{
			"nil metrics retention",
			&observabilitySdk.GetInstanceResponse{
				Id:   "iid",
				Name: nil,
			},
			&observabilitySdk.ListACLResponse{
				Acl:     []string{},
				Message: "",
			},
			nil,
			nil,
			nil,
			Model{},
			false,
		},
		{
			"update metrics retention",
			&observabilitySdk.GetInstanceResponse{
				Id:         "iid",
				Name:       new("name"),
				PlanName:   "plan1",
				PlanId:     "planId",
				Parameters: &map[string]string{"key": "value"},
				Instance: observabilitySdk.InstanceSensitiveData{
					MetricsRetentionTimeRaw: int32(30),
					MetricsRetentionTime1h:  int32(15),
					MetricsRetentionTime5m:  int32(10),
				},
			},
			&observabilitySdk.ListACLResponse{
				Acl: []string{
					"1.1.1.1/32",
				},
				Message: "message",
			},
			&observabilitySdk.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "7d",
			},
			&observabilitySdk.LogsConfigResponse{Config: observabilitySdk.LogsConfig{Retention: "480h"}},
			&observabilitySdk.TracesConfigResponse{Config: observabilitySdk.TraceConfig{Retention: "720h"}},
			Model{
				Id:         types.StringValue("pid,iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				InstanceId: types.StringValue("iid"),
				PlanId:     types.StringValue("planId"),
				PlanName:   types.StringValue("plan1"),
				Parameters: toTerraformStringMapMust(context.Background(), map[string]string{"key": "value"}),
				ACL: types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("1.1.1.1/32"),
				}),
				LogsRetentionDays:                  types.Int64Value(20),
				TracesRetentionDays:                types.Int64Value(30),
				MetricsRetentionDays:               types.Int32Value(60),
				MetricsRetentionDays1hDownsampling: types.Int32Value(30),
				MetricsRetentionDays5mDownsampling: types.Int32Value(7),
				DashboardURL:                       types.StringValue(""),
				GrafanaURL:                         types.StringValue(""),
				GrafanaPublicReadAccess:            types.BoolValue(false),
				GrafanaAdminEnabled:                types.BoolValue(false),
				MetricsURL:                         types.StringValue(""),
				MetricsPushURL:                     types.StringValue(""),
				TargetsURL:                         types.StringValue(""),
				AlertingURL:                        types.StringValue(""),
				LogsURL:                            types.StringValue(""),
				LogsPushURL:                        types.StringValue(""),
				JaegerTracesURL:                    types.StringValue(""),
				JaegerUIURL:                        types.StringValue(""),
				OtlpGRPCTracesURL:                  types.StringValue(""),
				OtlpHTTPLogsURL:                    types.StringValue(""),
				OtlpHTTPTracesURL:                  types.StringValue(""),
				OtlpTracesURL:                      types.StringValue(""),
				ZipkinSpansURL:                     types.StringValue(""),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId: tt.expected.ProjectId,
				ACL:       types.SetNull(types.StringType),
			}
			err := mapFields(context.Background(), tt.instanceResp, state)
			aclErr := mapACLField(tt.listACLResp, state)
			metricsErr := mapMetricsRetentionField(tt.getMetricsRetentionResp, state)
			logsErr := mapLogsRetentionField(tt.getLogsRetentionResp, state)
			tracesErr := mapTracesRetentionField(tt.getTracesRetentionResp, state)
			if !tt.isValid && err == nil && aclErr == nil && metricsErr == nil && logsErr == nil && tracesErr == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && (err != nil || aclErr != nil || metricsErr != nil || logsErr != nil || tracesErr != nil) {
				t.Fatalf("Should not have failed: %v, aclErr %v, metricsErr %v, logsErr %v, tracesErr %v", err, aclErr, metricsErr, logsErr, tracesErr)
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

func TestMapAlertConfigField(t *testing.T) {
	tests := []struct {
		description     string
		alertConfigResp *observabilitySdk.GetAlertConfigsResponse
		expected        Model
		isValid         bool
	}{
		{
			description: "basic_ok",
			alertConfigResp: &observabilitySdk.GetAlertConfigsResponse{
				Data: observabilitySdk.Alert{
					Receivers: []observabilitySdk.Receivers{
						fixtureReceiverResponse(
							[]observabilitySdk.EmailConfig{
								fixtureEmailConfigsResponse(),
							},
							[]observabilitySdk.OpsgenieConfig{
								fixtureOpsGenieConfigsResponse(),
							},
							[]observabilitySdk.WebHook{
								fixtureWebHooksConfigsResponse(),
							},
						),
					},
					Route:  fixtureRouteResponse(),
					Global: fixtureGlobalConfigResponse(),
				},
			},
			expected: Model{
				ACL:        types.SetNull(types.StringType),
				Parameters: types.MapNull(types.StringType),
				AlertConfig: types.ObjectValueMust(alertConfigTypes, map[string]attr.Value{
					"receivers": types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
						fixtureReceiverModel(
							fixtureEmailConfigsModel(),
							fixtureOpsGenieConfigsModel(),
							fixtureWebHooksConfigsModel(),
						),
					}),
					"route":  fixtureRouteModel(),
					"global": fixtureGlobalConfigModel(),
				}),
			},
			isValid: true,
		},
		{
			description: "receivers only emailconfigs",
			alertConfigResp: &observabilitySdk.GetAlertConfigsResponse{
				Data: observabilitySdk.Alert{
					Receivers: []observabilitySdk.Receivers{
						fixtureReceiverResponse(
							[]observabilitySdk.EmailConfig{
								fixtureEmailConfigsResponse(),
							},
							nil,
							nil,
						),
					},
					Route: fixtureRouteResponse(),
				},
			},
			expected: Model{
				ACL:        types.SetNull(types.StringType),
				Parameters: types.MapNull(types.StringType),
				AlertConfig: types.ObjectValueMust(alertConfigTypes, map[string]attr.Value{
					"receivers": types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
						fixtureReceiverModel(
							fixtureEmailConfigsModel(),
							types.ListNull(types.ObjectType{AttrTypes: opsgenieConfigsTypes}),
							types.ListNull(types.ObjectType{AttrTypes: webHooksConfigsTypes}),
						),
					}),
					"route":  fixtureRouteModel(),
					"global": types.ObjectNull(globalConfigurationTypes),
				}),
			},
			isValid: true,
		},
		{
			description: "receivers only opsgenieconfigs",
			alertConfigResp: &observabilitySdk.GetAlertConfigsResponse{
				Data: observabilitySdk.Alert{
					Receivers: []observabilitySdk.Receivers{
						fixtureReceiverResponse(
							nil,
							[]observabilitySdk.OpsgenieConfig{
								fixtureOpsGenieConfigsResponse(),
							},
							nil,
						),
					},
					Route: fixtureRouteResponse(),
				},
			},
			expected: Model{
				ACL:        types.SetNull(types.StringType),
				Parameters: types.MapNull(types.StringType),
				AlertConfig: types.ObjectValueMust(alertConfigTypes, map[string]attr.Value{
					"receivers": types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
						fixtureReceiverModel(
							types.ListNull(types.ObjectType{AttrTypes: emailConfigsTypes}),
							fixtureOpsGenieConfigsModel(),
							types.ListNull(types.ObjectType{AttrTypes: webHooksConfigsTypes}),
						),
					}),
					"route":  fixtureRouteModel(),
					"global": types.ObjectNull(globalConfigurationTypes),
				}),
			},
			isValid: true,
		},
		{
			description: "receivers only webhooksconfigs",
			alertConfigResp: &observabilitySdk.GetAlertConfigsResponse{
				Data: observabilitySdk.Alert{
					Receivers: []observabilitySdk.Receivers{
						fixtureReceiverResponse(
							nil,
							nil,
							[]observabilitySdk.WebHook{
								fixtureWebHooksConfigsResponse(),
							},
						),
					},
					Route: fixtureRouteResponse(),
				},
			},
			expected: Model{
				ACL:        types.SetNull(types.StringType),
				Parameters: types.MapNull(types.StringType),
				AlertConfig: types.ObjectValueMust(alertConfigTypes, map[string]attr.Value{
					"receivers": types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
						fixtureReceiverModel(
							types.ListNull(types.ObjectType{AttrTypes: emailConfigsTypes}),
							types.ListNull(types.ObjectType{AttrTypes: opsgenieConfigsTypes}),
							fixtureWebHooksConfigsModel(),
						),
					}),
					"route":  fixtureRouteModel(),
					"global": types.ObjectNull(globalConfigurationTypes),
				}),
			},
			isValid: true,
		},
		{
			description: "no receivers, no routes",
			alertConfigResp: &observabilitySdk.GetAlertConfigsResponse{
				Data: observabilitySdk.Alert{
					Receivers: []observabilitySdk.Receivers{},
					Route:     observabilitySdk.Route{},
				},
			},
			expected: Model{
				ACL:        types.SetNull(types.StringType),
				Parameters: types.MapNull(types.StringType),
				AlertConfig: types.ObjectValueMust(alertConfigTypes, map[string]attr.Value{
					"receivers": types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{}),
					"route":     fixtureNullRouteModel(),
					"global":    types.ObjectNull(globalConfigurationTypes),
				}),
			},
			isValid: true,
		},
		{
			description: "no receivers, default routes",
			alertConfigResp: &observabilitySdk.GetAlertConfigsResponse{
				Data: observabilitySdk.Alert{
					Receivers: []observabilitySdk.Receivers{},
					Route:     fixtureRouteResponse(),
				},
			},
			expected: Model{
				ACL:        types.SetNull(types.StringType),
				Parameters: types.MapNull(types.StringType),
				AlertConfig: types.ObjectValueMust(alertConfigTypes, map[string]attr.Value{
					"receivers": types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{}),
					"route":     fixtureRouteModel(),
					"global":    types.ObjectNull(globalConfigurationTypes),
				}),
			},
			isValid: true,
		},
		{
			description: "default receivers, no routes",
			alertConfigResp: &observabilitySdk.GetAlertConfigsResponse{
				Data: observabilitySdk.Alert{
					Receivers: []observabilitySdk.Receivers{
						fixtureReceiverResponse(
							[]observabilitySdk.EmailConfig{
								fixtureEmailConfigsResponse(),
							},
							[]observabilitySdk.OpsgenieConfig{
								fixtureOpsGenieConfigsResponse(),
							},
							[]observabilitySdk.WebHook{
								fixtureWebHooksConfigsResponse(),
							},
						),
					},
					Route: observabilitySdk.Route{},
				},
			},
			expected: Model{
				ACL:        types.SetNull(types.StringType),
				Parameters: types.MapNull(types.StringType),
				AlertConfig: types.ObjectValueMust(alertConfigTypes, map[string]attr.Value{
					"receivers": types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
						fixtureReceiverModel(
							fixtureEmailConfigsModel(),
							fixtureOpsGenieConfigsModel(),
							fixtureWebHooksConfigsModel(),
						),
					}),
					"route":  fixtureNullRouteModel(),
					"global": types.ObjectNull(globalConfigurationTypes),
				}),
			},
			isValid: true,
		},
		{
			description: "nil receivers",
			alertConfigResp: &observabilitySdk.GetAlertConfigsResponse{
				Data: observabilitySdk.Alert{
					Receivers: nil,
					Route:     fixtureRouteResponse(),
				},
			},
			expected: Model{
				ACL:        types.SetNull(types.StringType),
				Parameters: types.MapNull(types.StringType),
				AlertConfig: types.ObjectValueMust(alertConfigTypes, map[string]attr.Value{
					"receivers": types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{}),
					"route":     fixtureRouteModel(),
					"global":    types.ObjectNull(globalConfigurationTypes),
				}),
			},
			isValid: true,
		},
		{
			description: "nil route",
			alertConfigResp: &observabilitySdk.GetAlertConfigsResponse{
				Data: observabilitySdk.Alert{
					Receivers: []observabilitySdk.Receivers{
						fixtureReceiverResponse(
							[]observabilitySdk.EmailConfig{
								fixtureEmailConfigsResponse(),
							},
							[]observabilitySdk.OpsgenieConfig{
								fixtureOpsGenieConfigsResponse(),
							},
							[]observabilitySdk.WebHook{
								fixtureWebHooksConfigsResponse(),
							},
						),
					},
					Route: observabilitySdk.Route{},
				},
			},
			expected: Model{
				ACL:        types.SetNull(types.StringType),
				Parameters: types.MapNull(types.StringType),
				AlertConfig: types.ObjectValueMust(alertConfigTypes, map[string]attr.Value{
					"receivers": types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
						fixtureReceiverModel(
							fixtureEmailConfigsModel(),
							fixtureOpsGenieConfigsModel(),
							fixtureWebHooksConfigsModel(),
						),
					}),
					"route":  fixtureNullRouteModel(),
					"global": types.ObjectNull(globalConfigurationTypes),
				}),
			},
			isValid: true,
		},
		{
			description: "empty global options",
			alertConfigResp: &observabilitySdk.GetAlertConfigsResponse{
				Data: observabilitySdk.Alert{
					Receivers: []observabilitySdk.Receivers{
						fixtureReceiverResponse(
							[]observabilitySdk.EmailConfig{
								fixtureEmailConfigsResponse(),
							},
							[]observabilitySdk.OpsgenieConfig{
								fixtureOpsGenieConfigsResponse(),
							},
							[]observabilitySdk.WebHook{
								fixtureWebHooksConfigsResponse(),
							},
						),
					},
					Route:  fixtureRouteResponse(),
					Global: &observabilitySdk.Global{},
				},
			},
			expected: Model{
				ACL:        types.SetNull(types.StringType),
				Parameters: types.MapNull(types.StringType),
				AlertConfig: types.ObjectValueMust(alertConfigTypes, map[string]attr.Value{
					"receivers": types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
						fixtureReceiverModel(
							fixtureEmailConfigsModel(),
							fixtureOpsGenieConfigsModel(),
							fixtureWebHooksConfigsModel(),
						),
					}),
					"route":  fixtureRouteModel(),
					"global": fixtureNullGlobalConfigModel(),
				}),
			},
			isValid: true,
		},
		{
			description:     "nil resp",
			alertConfigResp: nil,
			expected: Model{
				ACL:         types.SetNull(types.StringType),
				Parameters:  types.MapNull(types.StringType),
				AlertConfig: types.ObjectNull(receiversTypes),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				ACL:        types.SetNull(types.StringType),
				Parameters: types.MapNull(types.StringType),
			}
			err := mapAlertConfigField(context.Background(), tt.alertConfigResp, state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}

			if tt.isValid {
				diff := cmp.Diff(state.AlertConfig, tt.expected.AlertConfig)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description         string
		input               *Model
		grafanaAdminEnabled bool
		expected            *observabilitySdk.CreateInstancePayload
		isValid             bool
	}{
		{
			"basic_ok",
			&Model{
				GrafanaAdminEnabled: types.BoolValue(true),
				PlanId:              types.StringValue("planId"),
			},
			true,
			&observabilitySdk.CreateInstancePayload{
				GrafanaAdminEnabled: new(true),
				Name:                nil,
				PlanId:              "planId",
				Parameter:           map[string]any{},
			},
			true,
		},
		{
			"ok",
			&Model{
				GrafanaAdminEnabled: types.BoolValue(false),
				Name:                types.StringValue("Name"),
				PlanId:              types.StringValue("planId"),
				Parameters:          makeTestMap(t),
			},
			true,
			&observabilitySdk.CreateInstancePayload{
				GrafanaAdminEnabled: new(false),
				Name:                new("Name"),
				PlanId:              "planId",
				Parameter:           map[string]any{"key": `"value"`},
			},
			true,
		},
		{
			"plan does not support grafana",
			&Model{
				Name:       types.StringValue("Name"),
				PlanId:     types.StringValue("planId"),
				Parameters: makeTestMap(t),
			},
			false,
			&observabilitySdk.CreateInstancePayload{
				Name:      new("Name"),
				PlanId:    "planId",
				Parameter: map[string]any{"key": `"value"`},
			},
			true,
		},
		{
			"nil_model",
			nil,
			true,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input, tt.grafanaAdminEnabled)
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

func TestToPayloadUpdate(t *testing.T) {
	tests := []struct {
		description         string
		input               *Model
		grafanaAdminEnabled bool
		expected            *observabilitySdk.UpdateInstancePayload
		isValid             bool
	}{
		{
			"basic_ok",
			&Model{
				GrafanaAdminEnabled: types.BoolValue(true),
				PlanId:              types.StringValue("planId"),
			},
			true,
			&observabilitySdk.UpdateInstancePayload{
				GrafanaAdminEnabled: new(true),
				Name:                nil,
				PlanId:              new("planId"),
				Parameter:           map[string]any{},
			},
			true,
		},
		{
			"ok",
			&Model{
				GrafanaAdminEnabled: types.BoolValue(false),
				Name:                types.StringValue("Name"),
				PlanId:              types.StringValue("planId"),
				Parameters:          makeTestMap(t),
			},
			true,
			&observabilitySdk.UpdateInstancePayload{
				GrafanaAdminEnabled: new(false),
				Name:                new("Name"),
				PlanId:              new("planId"),
				Parameter:           map[string]any{"key": `"value"`},
			},
			true,
		},
		{
			"plan does not support grafana",
			&Model{
				Name:       types.StringValue("Name"),
				PlanId:     types.StringValue("planId"),
				Parameters: makeTestMap(t),
			},
			false,
			&observabilitySdk.UpdateInstancePayload{
				Name:      new("Name"),
				PlanId:    new("planId"),
				Parameter: map[string]any{"key": `"value"`},
			},
			true,
		},
		{
			"nil_model",
			nil,
			true,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input, tt.grafanaAdminEnabled)
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

func TestToUpdateMetricsStorageRetentionPayload(t *testing.T) {
	tests := []struct {
		description      string
		retentionDaysRaw *int32
		retentionDays1h  *int32
		retentionDays5m  *int32
		getMetricsResp   *observabilitySdk.GetMetricsStorageRetentionResponse
		expected         *observabilitySdk.UpdateMetricsStorageRetentionPayload
		isValid          bool
	}{
		{
			"basic_ok",
			new(int32(120)),
			new(int32(60)),
			new(int32(14)),
			&observabilitySdk.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "7d",
			},
			&observabilitySdk.UpdateMetricsStorageRetentionPayload{
				MetricsRetentionTimeRaw: "120d",
				MetricsRetentionTime1h:  "60d",
				MetricsRetentionTime5m:  "14d",
			},
			true,
		},
		{
			"only_raw_given",
			new(int32(120)),
			nil,
			nil,
			&observabilitySdk.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "7d",
			},
			&observabilitySdk.UpdateMetricsStorageRetentionPayload{
				MetricsRetentionTimeRaw: "120d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "7d",
			},
			true,
		},
		{
			"only_1h_given",
			nil,
			new(int32(60)),
			nil,
			&observabilitySdk.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "7d",
			},
			&observabilitySdk.UpdateMetricsStorageRetentionPayload{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "60d",
				MetricsRetentionTime5m:  "7d",
			},
			true,
		},
		{
			"only_5m_given",
			nil,
			nil,
			new(int32(14)),
			&observabilitySdk.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "7d",
			},
			&observabilitySdk.UpdateMetricsStorageRetentionPayload{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "14d",
			},
			true,
		},
		{
			"none_given",
			nil,
			nil,
			nil,
			&observabilitySdk.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "7d",
			},
			&observabilitySdk.UpdateMetricsStorageRetentionPayload{
				MetricsRetentionTimeRaw: "60d",
				MetricsRetentionTime1h:  "30d",
				MetricsRetentionTime5m:  "7d",
			},
			true,
		},
		{
			"nil_response",
			nil,
			nil,
			nil,
			nil,
			nil,
			false,
		},
		{
			"empty_response",
			nil,
			nil,
			nil,
			&observabilitySdk.GetMetricsStorageRetentionResponse{},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdateMetricsStorageRetentionPayload(tt.retentionDaysRaw, tt.retentionDays5m, tt.retentionDays1h, tt.getMetricsResp)
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

func TestToUpdateAlertConfigPayload(t *testing.T) {
	tests := []struct {
		description string
		input       alertConfigModel
		expected    *observabilitySdk.UpdateAlertConfigsPayload
		isValid     bool
	}{
		{
			description: "base",
			input: alertConfigModel{
				Receivers: types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
					fixtureReceiverModel(
						fixtureEmailConfigsModel(),
						fixtureOpsGenieConfigsModel(),
						fixtureWebHooksConfigsModel(),
					),
				}),
				Route:               fixtureRouteModel(),
				GlobalConfiguration: fixtureGlobalConfigModel(),
			},
			expected: &observabilitySdk.UpdateAlertConfigsPayload{
				Receivers: []observabilitySdk.UpdateAlertConfigsPayloadReceiversInner{
					fixtureReceiverPayload(
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerEmailConfigsInner{fixtureEmailConfigsPayload()},
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerOpsgenieConfigsInner{fixtureOpsGenieConfigsPayload()},
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerWebHookConfigsInner{fixtureWebHooksConfigsPayload()},
					),
				},
				Route:  fixtureRoutePayload(),
				Global: fixtureGlobalConfigPayload(),
			},
			isValid: true,
		},
		{
			description: "receivers only emailconfigs",
			input: alertConfigModel{
				Receivers: types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
					fixtureReceiverModel(
						fixtureEmailConfigsModel(),
						types.ListNull(types.ObjectType{AttrTypes: opsgenieConfigsTypes}),
						types.ListNull(types.ObjectType{AttrTypes: webHooksConfigsTypes}),
					),
				}),
				Route: fixtureRouteModel(),
			},
			expected: &observabilitySdk.UpdateAlertConfigsPayload{
				Receivers: []observabilitySdk.UpdateAlertConfigsPayloadReceiversInner{
					fixtureReceiverPayload(
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerEmailConfigsInner{fixtureEmailConfigsPayload()},
						nil,
						nil,
					),
				},
				Route: fixtureRoutePayload(),
			},
			isValid: true,
		},
		{
			description: "receivers only opsgenieconfigs",
			input: alertConfigModel{
				Receivers: types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
					fixtureReceiverModel(
						types.ListNull(types.ObjectType{AttrTypes: emailConfigsTypes}),
						fixtureOpsGenieConfigsModel(),
						types.ListNull(types.ObjectType{AttrTypes: webHooksConfigsTypes}),
					),
				}),
				Route: fixtureRouteModel(),
			},
			expected: &observabilitySdk.UpdateAlertConfigsPayload{
				Receivers: []observabilitySdk.UpdateAlertConfigsPayloadReceiversInner{
					fixtureReceiverPayload(
						nil,
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerOpsgenieConfigsInner{fixtureOpsGenieConfigsPayload()},
						nil,
					),
				},
				Route: fixtureRoutePayload(),
			},
			isValid: true,
		},
		{
			description: "multiple receivers",
			input: alertConfigModel{
				Receivers: types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
					fixtureReceiverModel(
						fixtureEmailConfigsModel(),
						fixtureOpsGenieConfigsModel(),
						fixtureWebHooksConfigsModel(),
					),
					fixtureReceiverModel(
						fixtureEmailConfigsModel(),
						fixtureOpsGenieConfigsModel(),
						fixtureWebHooksConfigsModel(),
					),
				}),
				Route: fixtureRouteModel(),
			},
			expected: &observabilitySdk.UpdateAlertConfigsPayload{
				Receivers: []observabilitySdk.UpdateAlertConfigsPayloadReceiversInner{
					fixtureReceiverPayload(
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerEmailConfigsInner{fixtureEmailConfigsPayload()},
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerOpsgenieConfigsInner{fixtureOpsGenieConfigsPayload()},
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerWebHookConfigsInner{fixtureWebHooksConfigsPayload()},
					),
					fixtureReceiverPayload(
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerEmailConfigsInner{fixtureEmailConfigsPayload()},
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerOpsgenieConfigsInner{fixtureOpsGenieConfigsPayload()},
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerWebHookConfigsInner{fixtureWebHooksConfigsPayload()},
					),
				},
				Route: fixtureRoutePayload(),
			},
			isValid: true,
		},
		{
			description: "empty global options",
			input: alertConfigModel{
				Receivers: types.ListValueMust(types.ObjectType{AttrTypes: receiversTypes}, []attr.Value{
					fixtureReceiverModel(
						fixtureEmailConfigsModel(),
						fixtureOpsGenieConfigsModel(),
						fixtureWebHooksConfigsModel(),
					),
				}),
				Route:               fixtureRouteModel(),
				GlobalConfiguration: fixtureNullGlobalConfigModel(),
			},
			expected: &observabilitySdk.UpdateAlertConfigsPayload{
				Receivers: []observabilitySdk.UpdateAlertConfigsPayloadReceiversInner{
					fixtureReceiverPayload(
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerEmailConfigsInner{fixtureEmailConfigsPayload()},
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerOpsgenieConfigsInner{fixtureOpsGenieConfigsPayload()},
						[]observabilitySdk.UpdateAlertConfigsPayloadReceiversInnerWebHookConfigsInner{fixtureWebHooksConfigsPayload()},
					),
				},
				Route:  fixtureRoutePayload(),
				Global: &observabilitySdk.UpdateAlertConfigsPayloadGlobal{},
			},
			isValid: true,
		},
		{
			description: "empty alert config",
			input:       alertConfigModel{},
			isValid:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdateAlertConfigPayload(context.Background(), &tt.input)
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

func TestGetRouteNestedObjectAux(t *testing.T) {
	tests := []struct {
		description    string
		startingLevel  int
		recursionLimit int
		isDatasource   bool
		expected       schema.ListNestedAttribute
	}{
		{
			"no recursion, resource",
			1,
			1,
			false,
			schema.ListNestedAttribute{
				Description: routeDescriptions["routes"],
				Optional:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: fixtureRouteAttributeSchema(nil, false),
				},
			},
		},
		{
			"recursion 1, resource",
			1,
			2,
			false,
			schema.ListNestedAttribute{
				Description: routeDescriptions["routes"],
				Optional:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: fixtureRouteAttributeSchema(
						&schema.ListNestedAttribute{
							Description: routeDescriptions["routes"],
							Optional:    true,
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: fixtureRouteAttributeSchema(nil, false),
							},
						},
						false,
					),
				},
			},
		},
		{
			"no recursion,datasource",
			1,
			1,
			true,
			schema.ListNestedAttribute{
				Description: routeDescriptions["routes"],
				Computed:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: fixtureRouteAttributeSchema(nil, true),
				},
			},
		},
		{
			"recursion 1, datasource",
			1,
			2,
			true,
			schema.ListNestedAttribute{
				Description: routeDescriptions["routes"],
				Computed:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: fixtureRouteAttributeSchema(
						&schema.ListNestedAttribute{
							Description: routeDescriptions["routes"],
							Computed:    true,
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: fixtureRouteAttributeSchema(nil, true),
							},
						},
						true,
					),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output := getRouteNestedObjectAux(tt.isDatasource, tt.startingLevel, tt.recursionLimit)
			diff := cmp.Diff(output, tt.expected)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}

func TestGetRouteListTypeAux(t *testing.T) {
	tests := []struct {
		description    string
		startingLevel  int
		recursionLimit int
		expected       types.ObjectType
	}{
		{
			"no recursion",
			1,
			1,
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"continue":        types.BoolType,
					"group_by":        types.ListType{ElemType: types.StringType},
					"group_interval":  types.StringType,
					"group_wait":      types.StringType,
					"match":           types.MapType{ElemType: types.StringType},
					"match_regex":     types.MapType{ElemType: types.StringType},
					"matchers":        types.ListType{ElemType: types.StringType},
					"receiver":        types.StringType,
					"repeat_interval": types.StringType,
				},
			},
		},
		{
			"recursion 1",
			1,
			2,
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"continue":        types.BoolType,
					"group_by":        types.ListType{ElemType: types.StringType},
					"group_interval":  types.StringType,
					"group_wait":      types.StringType,
					"match":           types.MapType{ElemType: types.StringType},
					"match_regex":     types.MapType{ElemType: types.StringType},
					"matchers":        types.ListType{ElemType: types.StringType},
					"receiver":        types.StringType,
					"repeat_interval": types.StringType,
					"routes": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"continue":        types.BoolType,
						"group_by":        types.ListType{ElemType: types.StringType},
						"group_interval":  types.StringType,
						"group_wait":      types.StringType,
						"match":           types.MapType{ElemType: types.StringType},
						"match_regex":     types.MapType{ElemType: types.StringType},
						"matchers":        types.ListType{ElemType: types.StringType},
						"receiver":        types.StringType,
						"repeat_interval": types.StringType,
					}}},
				},
			},
		},
		{
			"recursion 2",
			2,
			2,
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"continue":        types.BoolType,
					"group_by":        types.ListType{ElemType: types.StringType},
					"group_interval":  types.StringType,
					"group_wait":      types.StringType,
					"match":           types.MapType{ElemType: types.StringType},
					"match_regex":     types.MapType{ElemType: types.StringType},
					"matchers":        types.ListType{ElemType: types.StringType},
					"receiver":        types.StringType,
					"repeat_interval": types.StringType,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output := getRouteListTypeAux(tt.startingLevel, tt.recursionLimit)
			diff := cmp.Diff(output, tt.expected)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}

func makeTestMap(t *testing.T) basetypes.MapValue {
	p := make(map[string]attr.Value, 1)
	p["key"] = types.StringValue("value")
	params, diag := types.MapValueFrom(context.Background(), types.StringType, p)
	if diag.HasError() {
		t.Fail()
	}
	return params
}

// ToTerraformStringMapMust Silently ignores the error
func toTerraformStringMapMust(ctx context.Context, m map[string]string) basetypes.MapValue {
	labels := make(map[string]attr.Value, len(m))
	for l, v := range m {
		stringValue := types.StringValue(v)
		labels[l] = stringValue
	}
	res, diags := types.MapValueFrom(ctx, types.StringType, m)
	if diags.HasError() {
		return types.MapNull(types.StringType)
	}
	return res
}
