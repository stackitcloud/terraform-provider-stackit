package observability

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
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
		"group_by":        types.ListNull(types.StringType),
		"group_interval":  types.StringNull(),
		"group_wait":      types.StringNull(),
		"receiver":        types.StringNull(),
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

func fixtureEmailConfigsPayload() observability.CreateAlertConfigReceiverPayloadEmailConfigsInner {
	return observability.CreateAlertConfigReceiverPayloadEmailConfigsInner{
		AuthIdentity: utils.Ptr("identity"),
		AuthPassword: utils.Ptr("password"),
		AuthUsername: utils.Ptr("username"),
		From:         utils.Ptr("notification@example.com"),
		SendResolved: utils.Ptr(true),
		Smarthost:    utils.Ptr("smtp.example.com"),
		To:           utils.Ptr("me@example.com"),
	}
}

func fixtureOpsGenieConfigsPayload() observability.CreateAlertConfigReceiverPayloadOpsgenieConfigsInner {
	return observability.CreateAlertConfigReceiverPayloadOpsgenieConfigsInner{
		ApiKey:       utils.Ptr("key"),
		Tags:         utils.Ptr("tag"),
		ApiUrl:       utils.Ptr("ops.example.com"),
		Priority:     utils.Ptr("P3"),
		SendResolved: utils.Ptr(true),
	}
}

func fixtureWebHooksConfigsPayload() observability.CreateAlertConfigReceiverPayloadWebHookConfigsInner {
	return observability.CreateAlertConfigReceiverPayloadWebHookConfigsInner{
		Url:          utils.Ptr("http://example.com"),
		MsTeams:      utils.Ptr(true),
		GoogleChat:   utils.Ptr(true),
		SendResolved: utils.Ptr(true),
	}
}

func fixtureReceiverPayload(emailConfigs *[]observability.CreateAlertConfigReceiverPayloadEmailConfigsInner, opsGenieConfigs *[]observability.CreateAlertConfigReceiverPayloadOpsgenieConfigsInner, webHooksConfigs *[]observability.CreateAlertConfigReceiverPayloadWebHookConfigsInner) observability.UpdateAlertConfigsPayloadReceiversInner {
	return observability.UpdateAlertConfigsPayloadReceiversInner{
		EmailConfigs:    emailConfigs,
		Name:            utils.Ptr("name"),
		OpsgenieConfigs: opsGenieConfigs,
		WebHookConfigs:  webHooksConfigs,
	}
}

func fixtureRoutePayload() *observability.UpdateAlertConfigsPayloadRoute {
	return &observability.UpdateAlertConfigsPayloadRoute{
		Continue:       nil,
		GroupBy:        utils.Ptr([]string{"label1", "label2"}),
		GroupInterval:  utils.Ptr("1m"),
		GroupWait:      utils.Ptr("1m"),
		Receiver:       utils.Ptr("name"),
		RepeatInterval: utils.Ptr("1m"),
		Routes: &[]observability.UpdateAlertConfigsPayloadRouteRoutesInner{
			{
				Continue:       utils.Ptr(false),
				GroupBy:        utils.Ptr([]string{"label1", "label2"}),
				GroupInterval:  utils.Ptr("1m"),
				GroupWait:      utils.Ptr("1m"),
				Match:          &map[string]interface{}{"key": "value"},
				MatchRe:        &map[string]interface{}{"key": "value"},
				Matchers:       &[]string{"matcher1", "matcher2"},
				Receiver:       utils.Ptr("name"),
				RepeatInterval: utils.Ptr("1m"),
			},
		},
	}
}

func fixtureGlobalConfigPayload() *observability.UpdateAlertConfigsPayloadGlobal {
	return &observability.UpdateAlertConfigsPayloadGlobal{
		OpsgenieApiKey:   utils.Ptr("key"),
		OpsgenieApiUrl:   utils.Ptr("ops.example.com"),
		ResolveTimeout:   utils.Ptr("1m"),
		SmtpAuthIdentity: utils.Ptr("identity"),
		SmtpAuthUsername: utils.Ptr("username"),
		SmtpAuthPassword: utils.Ptr("password"),
		SmtpFrom:         utils.Ptr("me@example.com"),
		SmtpSmarthost:    utils.Ptr("smtp.example.com:25"),
	}
}

func fixtureReceiverResponse(emailConfigs *[]observability.EmailConfig, opsGenieConfigs *[]observability.OpsgenieConfig, webhookConfigs *[]observability.WebHook) observability.Receivers {
	return observability.Receivers{
		Name:            utils.Ptr("name"),
		EmailConfigs:    emailConfigs,
		OpsgenieConfigs: opsGenieConfigs,
		WebHookConfigs:  webhookConfigs,
	}
}

func fixtureEmailConfigsResponse() observability.EmailConfig {
	return observability.EmailConfig{
		AuthIdentity: utils.Ptr("identity"),
		AuthPassword: utils.Ptr("password"),
		AuthUsername: utils.Ptr("username"),
		From:         utils.Ptr("notification@example.com"),
		SendResolved: utils.Ptr(true),
		Smarthost:    utils.Ptr("smtp.example.com"),
		To:           utils.Ptr("me@example.com"),
	}
}

func fixtureOpsGenieConfigsResponse() observability.OpsgenieConfig {
	return observability.OpsgenieConfig{
		ApiKey:       utils.Ptr("key"),
		Tags:         utils.Ptr("tag"),
		ApiUrl:       utils.Ptr("ops.example.com"),
		Priority:     utils.Ptr("P3"),
		SendResolved: utils.Ptr(true),
	}
}

func fixtureWebHooksConfigsResponse() observability.WebHook {
	return observability.WebHook{
		Url:          utils.Ptr("http://example.com"),
		MsTeams:      utils.Ptr(true),
		GoogleChat:   utils.Ptr(true),
		SendResolved: utils.Ptr(true),
	}
}

func fixtureRouteResponse() *observability.Route {
	return &observability.Route{
		Continue:       nil,
		GroupBy:        utils.Ptr([]string{"label1", "label2"}),
		GroupInterval:  utils.Ptr("1m"),
		GroupWait:      utils.Ptr("1m"),
		Match:          &map[string]string{"key": "value"},
		MatchRe:        &map[string]string{"key": "value"},
		Matchers:       &[]string{"matcher1", "matcher2"},
		Receiver:       utils.Ptr("name"),
		RepeatInterval: utils.Ptr("1m"),
		Routes: &[]observability.RouteSerializer{
			{
				Continue:       utils.Ptr(false),
				GroupBy:        utils.Ptr([]string{"label1", "label2"}),
				GroupInterval:  utils.Ptr("1m"),
				GroupWait:      utils.Ptr("1m"),
				Match:          &map[string]string{"key": "value"},
				MatchRe:        &map[string]string{"key": "value"},
				Matchers:       &[]string{"matcher1", "matcher2"},
				Receiver:       utils.Ptr("name"),
				RepeatInterval: utils.Ptr("1m"),
			},
		},
	}
}

func fixtureGlobalConfigResponse() *observability.Global {
	return &observability.Global{
		OpsgenieApiKey:   utils.Ptr("key"),
		OpsgenieApiUrl:   utils.Ptr("ops.example.com"),
		ResolveTimeout:   utils.Ptr("1m"),
		SmtpAuthIdentity: utils.Ptr("identity"),
		SmtpAuthUsername: utils.Ptr("username"),
		SmtpAuthPassword: utils.Ptr("password"),
		SmtpFrom:         utils.Ptr("me@example.com"),
		SmtpSmarthost:    utils.Ptr("smtp.example.com:25"),
	}
}

func fixtureRouteAttributeSchema(route *schema.ListNestedAttribute, isDatasource bool) map[string]schema.Attribute {
	attributeMap := map[string]schema.Attribute{
		"continue": schema.BoolAttribute{
			Description: routeDescriptions["continue"],
			Optional:    !isDatasource,
			Computed:    isDatasource,
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
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"group_wait": schema.StringAttribute{
			Description: routeDescriptions["group_wait"],
			Optional:    !isDatasource,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
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
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
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
		instanceResp            *observability.GetInstanceResponse
		listACLResp             *observability.ListACLResponse
		getMetricsRetentionResp *observability.GetMetricsStorageRetentionResponse
		getLogsRetentionResp    *observability.LogsConfigResponse
		getTracesRetentionResp  *observability.TracesConfigResponse
		expected                Model
		isValid                 bool
	}{
		{
			"default_ok",
			&observability.GetInstanceResponse{
				Id: utils.Ptr("iid"),
			},
			&observability.ListACLResponse{},
			&observability.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			&observability.LogsConfigResponse{Config: &observability.LogsConfig{Retention: utils.Ptr("168h")}},
			&observability.TracesConfigResponse{Config: &observability.TraceConfig{Retention: utils.Ptr("168h")}},
			Model{
				Id:                                 types.StringValue("pid,iid"),
				ProjectId:                          types.StringValue("pid"),
				InstanceId:                         types.StringValue("iid"),
				PlanId:                             types.StringNull(),
				PlanName:                           types.StringNull(),
				Name:                               types.StringNull(),
				Parameters:                         types.MapNull(types.StringType),
				ACL:                                types.SetNull(types.StringType),
				TracesRetentionDays:                types.Int64Value(7),
				LogsRetentionDays:                  types.Int64Value(7),
				MetricsRetentionDays:               types.Int64Value(60),
				MetricsRetentionDays1hDownsampling: types.Int64Value(30),
				MetricsRetentionDays5mDownsampling: types.Int64Value(7),
			},
			true,
		},
		{
			"values_ok",
			&observability.GetInstanceResponse{
				Id:         utils.Ptr("iid"),
				Name:       utils.Ptr("name"),
				PlanName:   utils.Ptr("plan1"),
				PlanId:     utils.Ptr("planId"),
				Parameters: &map[string]string{"key": "value"},
				Instance: &observability.InstanceSensitiveData{
					MetricsRetentionTimeRaw: utils.Ptr(int64(60)),
					MetricsRetentionTime1h:  utils.Ptr(int64(30)),
					MetricsRetentionTime5m:  utils.Ptr(int64(7)),
				},
			},
			&observability.ListACLResponse{
				Acl: &[]string{
					"1.1.1.1/32",
				},
				Message: utils.Ptr("message"),
			},
			&observability.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			&observability.LogsConfigResponse{Config: &observability.LogsConfig{Retention: utils.Ptr("168h")}},
			&observability.TracesConfigResponse{Config: &observability.TraceConfig{Retention: utils.Ptr("168h")}},
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
				MetricsRetentionDays:               types.Int64Value(60),
				MetricsRetentionDays1hDownsampling: types.Int64Value(30),
				MetricsRetentionDays5mDownsampling: types.Int64Value(7),
			},
			true,
		},
		{
			"values_ok_multiple_acls",
			&observability.GetInstanceResponse{
				Id:         utils.Ptr("iid"),
				Name:       utils.Ptr("name"),
				PlanName:   utils.Ptr("plan1"),
				PlanId:     utils.Ptr("planId"),
				Parameters: &map[string]string{"key": "value"},
			},
			&observability.ListACLResponse{
				Acl: &[]string{
					"1.1.1.1/32",
					"8.8.8.8/32",
				},
				Message: utils.Ptr("message"),
			},
			&observability.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			&observability.LogsConfigResponse{Config: &observability.LogsConfig{Retention: utils.Ptr("168h")}},
			&observability.TracesConfigResponse{Config: &observability.TraceConfig{Retention: utils.Ptr("168h")}},
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
				MetricsRetentionDays:               types.Int64Value(60),
				MetricsRetentionDays1hDownsampling: types.Int64Value(30),
				MetricsRetentionDays5mDownsampling: types.Int64Value(7),
			},
			true,
		},
		{
			"nullable_fields_ok",
			&observability.GetInstanceResponse{
				Id:   utils.Ptr("iid"),
				Name: nil,
			},
			&observability.ListACLResponse{
				Acl:     &[]string{},
				Message: nil,
			},
			&observability.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			&observability.LogsConfigResponse{Config: &observability.LogsConfig{Retention: utils.Ptr("168h")}},
			&observability.TracesConfigResponse{Config: &observability.TraceConfig{Retention: utils.Ptr("168h")}},
			Model{
				Id:                                 types.StringValue("pid,iid"),
				ProjectId:                          types.StringValue("pid"),
				InstanceId:                         types.StringValue("iid"),
				PlanId:                             types.StringNull(),
				PlanName:                           types.StringNull(),
				Name:                               types.StringNull(),
				Parameters:                         types.MapNull(types.StringType),
				ACL:                                types.SetNull(types.StringType),
				TracesRetentionDays:                types.Int64Value(7),
				LogsRetentionDays:                  types.Int64Value(7),
				MetricsRetentionDays:               types.Int64Value(60),
				MetricsRetentionDays1hDownsampling: types.Int64Value(30),
				MetricsRetentionDays5mDownsampling: types.Int64Value(7),
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
			&observability.GetInstanceResponse{},
			nil,
			nil,
			nil,
			nil,
			Model{},
			false,
		},
		{
			"empty metrics retention",
			&observability.GetInstanceResponse{
				Id:   utils.Ptr("iid"),
				Name: nil,
			},
			&observability.ListACLResponse{
				Acl:     &[]string{},
				Message: nil,
			},
			&observability.GetMetricsStorageRetentionResponse{},
			&observability.LogsConfigResponse{},
			&observability.TracesConfigResponse{},
			Model{},
			false,
		},
		{
			"nil metrics retention",
			&observability.GetInstanceResponse{
				Id:   utils.Ptr("iid"),
				Name: nil,
			},
			&observability.ListACLResponse{
				Acl:     &[]string{},
				Message: nil,
			},
			nil,
			nil,
			nil,
			Model{},
			false,
		},
		{
			"update metrics retention",
			&observability.GetInstanceResponse{
				Id:         utils.Ptr("iid"),
				Name:       utils.Ptr("name"),
				PlanName:   utils.Ptr("plan1"),
				PlanId:     utils.Ptr("planId"),
				Parameters: &map[string]string{"key": "value"},
				Instance: &observability.InstanceSensitiveData{
					MetricsRetentionTimeRaw: utils.Ptr(int64(30)),
					MetricsRetentionTime1h:  utils.Ptr(int64(15)),
					MetricsRetentionTime5m:  utils.Ptr(int64(10)),
				},
			},
			&observability.ListACLResponse{
				Acl: &[]string{
					"1.1.1.1/32",
				},
				Message: utils.Ptr("message"),
			},
			&observability.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			&observability.LogsConfigResponse{Config: &observability.LogsConfig{Retention: utils.Ptr("480h")}},
			&observability.TracesConfigResponse{Config: &observability.TraceConfig{Retention: utils.Ptr("720h")}},
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
				MetricsRetentionDays:               types.Int64Value(60),
				MetricsRetentionDays1hDownsampling: types.Int64Value(30),
				MetricsRetentionDays5mDownsampling: types.Int64Value(7),
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

func TestMapAlertConfigField(t *testing.T) {
	tests := []struct {
		description     string
		alertConfigResp *observability.GetAlertConfigsResponse
		expected        Model
		isValid         bool
	}{
		{
			description: "basic_ok",
			alertConfigResp: &observability.GetAlertConfigsResponse{
				Data: &observability.Alert{
					Receivers: &[]observability.Receivers{
						fixtureReceiverResponse(
							&[]observability.EmailConfig{
								fixtureEmailConfigsResponse(),
							},
							&[]observability.OpsgenieConfig{
								fixtureOpsGenieConfigsResponse(),
							},
							&[]observability.WebHook{
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
			alertConfigResp: &observability.GetAlertConfigsResponse{
				Data: &observability.Alert{
					Receivers: &[]observability.Receivers{
						fixtureReceiverResponse(
							&[]observability.EmailConfig{
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
			alertConfigResp: &observability.GetAlertConfigsResponse{
				Data: &observability.Alert{
					Receivers: &[]observability.Receivers{
						fixtureReceiverResponse(
							nil,
							&[]observability.OpsgenieConfig{
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
			alertConfigResp: &observability.GetAlertConfigsResponse{
				Data: &observability.Alert{
					Receivers: &[]observability.Receivers{
						fixtureReceiverResponse(
							nil,
							nil,
							&[]observability.WebHook{
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
			alertConfigResp: &observability.GetAlertConfigsResponse{
				Data: &observability.Alert{
					Receivers: &[]observability.Receivers{},
					Route:     &observability.Route{},
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
			alertConfigResp: &observability.GetAlertConfigsResponse{
				Data: &observability.Alert{
					Receivers: &[]observability.Receivers{},
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
			alertConfigResp: &observability.GetAlertConfigsResponse{
				Data: &observability.Alert{
					Receivers: &[]observability.Receivers{
						fixtureReceiverResponse(
							&[]observability.EmailConfig{
								fixtureEmailConfigsResponse(),
							},
							&[]observability.OpsgenieConfig{
								fixtureOpsGenieConfigsResponse(),
							},
							&[]observability.WebHook{
								fixtureWebHooksConfigsResponse(),
							},
						),
					},
					Route: &observability.Route{},
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
			alertConfigResp: &observability.GetAlertConfigsResponse{
				Data: &observability.Alert{
					Receivers: nil,
					Route:     fixtureRouteResponse(),
				},
			},
			expected: Model{
				ACL:        types.SetNull(types.StringType),
				Parameters: types.MapNull(types.StringType),
				AlertConfig: types.ObjectValueMust(alertConfigTypes, map[string]attr.Value{
					"receivers": types.ListNull(types.ObjectType{AttrTypes: receiversTypes}),
					"route":     fixtureRouteModel(),
					"global":    types.ObjectNull(globalConfigurationTypes),
				}),
			},
			isValid: true,
		},
		{
			description: "nil route",
			alertConfigResp: &observability.GetAlertConfigsResponse{
				Data: &observability.Alert{
					Receivers: &[]observability.Receivers{
						fixtureReceiverResponse(
							&[]observability.EmailConfig{
								fixtureEmailConfigsResponse(),
							},
							&[]observability.OpsgenieConfig{
								fixtureOpsGenieConfigsResponse(),
							},
							&[]observability.WebHook{
								fixtureWebHooksConfigsResponse(),
							},
						),
					},
					Route: nil,
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
					"route":  types.ObjectNull(mainRouteTypes),
					"global": types.ObjectNull(globalConfigurationTypes),
				}),
			},
			isValid: true,
		},
		{
			description: "empty global options",
			alertConfigResp: &observability.GetAlertConfigsResponse{
				Data: &observability.Alert{
					Receivers: &[]observability.Receivers{
						fixtureReceiverResponse(
							&[]observability.EmailConfig{
								fixtureEmailConfigsResponse(),
							},
							&[]observability.OpsgenieConfig{
								fixtureOpsGenieConfigsResponse(),
							},
							&[]observability.WebHook{
								fixtureWebHooksConfigsResponse(),
							},
						),
					},
					Route:  fixtureRouteResponse(),
					Global: &observability.Global{},
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
		description string
		input       *Model
		expected    *observability.CreateInstancePayload
		isValid     bool
	}{
		{
			"basic_ok",
			&Model{
				PlanId: types.StringValue("planId"),
			},
			&observability.CreateInstancePayload{
				Name:      nil,
				PlanId:    utils.Ptr("planId"),
				Parameter: &map[string]interface{}{},
			},
			true,
		},
		{
			"ok",
			&Model{
				Name:       types.StringValue("Name"),
				PlanId:     types.StringValue("planId"),
				Parameters: makeTestMap(t),
			},
			&observability.CreateInstancePayload{
				Name:      utils.Ptr("Name"),
				PlanId:    utils.Ptr("planId"),
				Parameter: &map[string]interface{}{"key": `"value"`},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)
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
		description string
		input       *Model
		expected    *observability.UpdateInstancePayload
		isValid     bool
	}{
		{
			"basic_ok",
			&Model{
				PlanId: types.StringValue("planId"),
			},
			&observability.UpdateInstancePayload{
				Name:      nil,
				PlanId:    utils.Ptr("planId"),
				Parameter: &map[string]any{},
			},
			true,
		},
		{
			"ok",
			&Model{
				Name:       types.StringValue("Name"),
				PlanId:     types.StringValue("planId"),
				Parameters: makeTestMap(t),
			},
			&observability.UpdateInstancePayload{
				Name:      utils.Ptr("Name"),
				PlanId:    utils.Ptr("planId"),
				Parameter: &map[string]any{"key": `"value"`},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input)
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
		retentionDaysRaw *int64
		retentionDays1h  *int64
		retentionDays5m  *int64
		getMetricsResp   *observability.GetMetricsStorageRetentionResponse
		expected         *observability.UpdateMetricsStorageRetentionPayload
		isValid          bool
	}{
		{
			"basic_ok",
			utils.Ptr(int64(120)),
			utils.Ptr(int64(60)),
			utils.Ptr(int64(14)),
			&observability.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			&observability.UpdateMetricsStorageRetentionPayload{
				MetricsRetentionTimeRaw: utils.Ptr("120d"),
				MetricsRetentionTime1h:  utils.Ptr("60d"),
				MetricsRetentionTime5m:  utils.Ptr("14d"),
			},
			true,
		},
		{
			"only_raw_given",
			utils.Ptr(int64(120)),
			nil,
			nil,
			&observability.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			&observability.UpdateMetricsStorageRetentionPayload{
				MetricsRetentionTimeRaw: utils.Ptr("120d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			true,
		},
		{
			"only_1h_given",
			nil,
			utils.Ptr(int64(60)),
			nil,
			&observability.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			&observability.UpdateMetricsStorageRetentionPayload{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("60d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			true,
		},
		{
			"only_5m_given",
			nil,
			nil,
			utils.Ptr(int64(14)),
			&observability.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			&observability.UpdateMetricsStorageRetentionPayload{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("14d"),
			},
			true,
		},
		{
			"none_given",
			nil,
			nil,
			nil,
			&observability.GetMetricsStorageRetentionResponse{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
			},
			&observability.UpdateMetricsStorageRetentionPayload{
				MetricsRetentionTimeRaw: utils.Ptr("60d"),
				MetricsRetentionTime1h:  utils.Ptr("30d"),
				MetricsRetentionTime5m:  utils.Ptr("7d"),
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
			&observability.GetMetricsStorageRetentionResponse{},
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
		expected    *observability.UpdateAlertConfigsPayload
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
			expected: &observability.UpdateAlertConfigsPayload{
				Receivers: &[]observability.UpdateAlertConfigsPayloadReceiversInner{
					fixtureReceiverPayload(
						&[]observability.CreateAlertConfigReceiverPayloadEmailConfigsInner{fixtureEmailConfigsPayload()},
						&[]observability.CreateAlertConfigReceiverPayloadOpsgenieConfigsInner{fixtureOpsGenieConfigsPayload()},
						&[]observability.CreateAlertConfigReceiverPayloadWebHookConfigsInner{fixtureWebHooksConfigsPayload()},
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
			expected: &observability.UpdateAlertConfigsPayload{
				Receivers: &[]observability.UpdateAlertConfigsPayloadReceiversInner{
					fixtureReceiverPayload(
						&[]observability.CreateAlertConfigReceiverPayloadEmailConfigsInner{fixtureEmailConfigsPayload()},
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
			expected: &observability.UpdateAlertConfigsPayload{
				Receivers: &[]observability.UpdateAlertConfigsPayloadReceiversInner{
					fixtureReceiverPayload(
						nil,
						&[]observability.CreateAlertConfigReceiverPayloadOpsgenieConfigsInner{fixtureOpsGenieConfigsPayload()},
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
			expected: &observability.UpdateAlertConfigsPayload{
				Receivers: &[]observability.UpdateAlertConfigsPayloadReceiversInner{
					fixtureReceiverPayload(
						&[]observability.CreateAlertConfigReceiverPayloadEmailConfigsInner{fixtureEmailConfigsPayload()},
						&[]observability.CreateAlertConfigReceiverPayloadOpsgenieConfigsInner{fixtureOpsGenieConfigsPayload()},
						&[]observability.CreateAlertConfigReceiverPayloadWebHookConfigsInner{fixtureWebHooksConfigsPayload()},
					),
					fixtureReceiverPayload(
						&[]observability.CreateAlertConfigReceiverPayloadEmailConfigsInner{fixtureEmailConfigsPayload()},
						&[]observability.CreateAlertConfigReceiverPayloadOpsgenieConfigsInner{fixtureOpsGenieConfigsPayload()},
						&[]observability.CreateAlertConfigReceiverPayloadWebHookConfigsInner{fixtureWebHooksConfigsPayload()},
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
			expected: &observability.UpdateAlertConfigsPayload{
				Receivers: &[]observability.UpdateAlertConfigsPayloadReceiversInner{
					fixtureReceiverPayload(
						&[]observability.CreateAlertConfigReceiverPayloadEmailConfigsInner{fixtureEmailConfigsPayload()},
						&[]observability.CreateAlertConfigReceiverPayloadOpsgenieConfigsInner{fixtureOpsGenieConfigsPayload()},
						&[]observability.CreateAlertConfigReceiverPayloadWebHookConfigsInner{fixtureWebHooksConfigsPayload()},
					),
				},
				Route:  fixtureRoutePayload(),
				Global: &observability.UpdateAlertConfigsPayloadGlobal{},
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
