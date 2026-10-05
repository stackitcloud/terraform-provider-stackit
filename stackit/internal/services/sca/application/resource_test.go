package sca

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"

	scaSdk "github.com/stackitcloud/stackit-sdk-go/services/sca/v1alphaapi"
)

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		name     string
		input    *ResourceModel
		expected *scaSdk.CreateApplicationPayload
		isValid  bool
	}{
		{
			"simple_values",
			&ResourceModel{
				Model: Model{
					DisplayName: types.StringValue("test-app"),
					Stopped:     types.BoolValue(true),
					Containers: types.ListValueMust(
						types.ObjectType{AttrTypes: containerTypes},
						[]attr.Value{
							types.ObjectValueMust(
								containerTypes,
								map[string]attr.Value{
									"name":       types.StringValue("app"),
									"image":      types.StringValue("nginx"),
									"cpu_millis": types.Int32Value(500),
									"memory_mb":  types.Int32Value(256),
									"command":    types.ListNull(types.StringType),
									"args":       types.ListNull(types.StringType),
									"env":        types.ObjectNull(envTypes),
								},
							),
						},
					),
					Network: types.ObjectValueMust(
						networkTypes,
						map[string]attr.Value{
							"public": types.ObjectValueMust(
								publicNetworkTypes,
								map[string]attr.Value{
									"enabled": types.BoolValue(true),
									"port":    types.Int32Value(8080),
									"acl":     types.ListNull(types.StringType),
								},
							),
						},
					),
					Scaling: types.ObjectValueMust(
						scalingTypes,
						map[string]attr.Value{
							"manual": types.ObjectValueMust(
								manualScalingTypes,
								map[string]attr.Value{
									"instances": types.Int32Value(2),
								},
							),
							"auto": types.ObjectNull(autoScalingTypes),
						},
					),
				},
			},
			&scaSdk.CreateApplicationPayload{
				DisplayName: "test-app",
				Stopped:     conversion.BoolValueToPointer(types.BoolValue(true)),
				Containers: []scaSdk.Container{
					{
						Name:   "app",
						Image:  "nginx",
						Cpu:    conversion.Int32ValueToPointer(types.Int32Value(500)),
						Memory: conversion.Int32ValueToPointer(types.Int32Value(256)),
					},
				},
				Network: scaSdk.Network{
					PublicIngress: true,
					Port:          conversion.Int32ValueToPointer(types.Int32Value(8080)),
				},
				Scaling: scaSdk.Scaling{
					Type: scaSdk.SCALINGTYPE_SCALING_TYPE_MANUAL,
					ManualScaling: &scaSdk.ManualScaling{
						Instances: 2,
					},
				},
			},
			true,
		},
		{
			"missing_required_fields",
			&ResourceModel{
				Model: Model{
					Containers: types.ListNull(types.ObjectType{AttrTypes: containerTypes}),
				},
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		name     string
		input    *ResourceModel
		expected *scaSdk.UpdateApplicationPayload
		isValid  bool
	}{
		{
			"simple_values",
			&ResourceModel{
				Model: Model{
					Stopped: types.BoolValue(false),
					Containers: types.ListValueMust(
						types.ObjectType{AttrTypes: containerTypes},
						[]attr.Value{
							types.ObjectValueMust(
								containerTypes,
								map[string]attr.Value{
									"name":       types.StringValue("app"),
									"image":      types.StringValue("nginx"),
									"cpu_millis": types.Int32Value(1000),
									"memory_mb":  types.Int32Value(1024),
									"command":    types.ListNull(types.StringType),
									"args":       types.ListNull(types.StringType),
									"env":        types.ObjectNull(envTypes),
								},
							),
						},
					),
					Network: types.ObjectNull(networkTypes),
					Scaling: types.ObjectValueMust(
						scalingTypes,
						map[string]attr.Value{
							"manual": types.ObjectValueMust(
								manualScalingTypes,
								map[string]attr.Value{
									"instances": types.Int32Value(3),
								},
							),
							"auto": types.ObjectNull(autoScalingTypes),
						},
					),
				},
			},
			&scaSdk.UpdateApplicationPayload{
				Stopped: conversion.BoolValueToPointer(types.BoolValue(false)),
				Containers: []scaSdk.Container{
					{
						Name:   "app",
						Image:  "nginx",
						Cpu:    conversion.Int32ValueToPointer(types.Int32Value(1000)),
						Memory: conversion.Int32ValueToPointer(types.Int32Value(1024)),
					},
				},
				Scaling: &scaSdk.Scaling{
					Type: scaSdk.SCALINGTYPE_SCALING_TYPE_MANUAL,
					ManualScaling: &scaSdk.ManualScaling{
						Instances: 3,
					},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
	tests := []struct {
		name     string
		input    *scaSdk.Application
		region   string
		expected ResourceModel
		isValid  bool
	}{
		{
			"simple_values",
			&scaSdk.Application{
				Id:          new("app-123"),
				DisplayName: "test-app",
				Stopped:     new(false),
				Containers: []scaSdk.Container{
					{
						Name:   "app",
						Image:  "nginx",
						Cpu:    conversion.Int32ValueToPointer(types.Int32Value(500)),
						Memory: conversion.Int32ValueToPointer(types.Int32Value(256)),
					},
				},
				Network: scaSdk.Network{
					PublicIngress: true,
					Port:          conversion.Int32ValueToPointer(types.Int32Value(8080)),
					IngressAcl:    []string{"1.1.1.1/32"},
				},
				Scaling: scaSdk.Scaling{
					Type: scaSdk.SCALINGTYPE_SCALING_TYPE_MANUAL,
					ManualScaling: &scaSdk.ManualScaling{
						Instances: 2,
					},
				},
				RuntimeStatus: &scaSdk.RuntimeStatus{
					CurrentStatus: new(scaSdk.CURRENTSTATUS_CURRENT_STATUS_RUNNING),
					Urls:          []string{"https://test-app.com"},
				},
			},
			"eu01",
			ResourceModel{
				Model: Model{
					Id:            types.StringValue("pid,eu01,env-id,app-123"),
					ProjectId:     types.StringValue("pid"),
					EnvironmentId: types.StringValue("env-id"),
					ApplicationId: types.StringValue("app-123"),
					DisplayName:   types.StringValue("test-app"),
					Stopped:       types.BoolValue(false),
					Containers: types.ListValueMust(
						types.ObjectType{AttrTypes: containerTypes},
						[]attr.Value{
							types.ObjectValueMust(
								containerTypes,
								map[string]attr.Value{
									"name":       types.StringValue("app"),
									"image":      types.StringValue("nginx"),
									"cpu_millis": types.Int32Value(500),
									"memory_mb":  types.Int32Value(256),
									"command":    types.ListNull(types.StringType),
									"args":       types.ListNull(types.StringType),
									"env":        types.ObjectNull(envTypes),
								},
							),
						},
					),
					Network: types.ObjectValueMust(
						networkTypes,
						map[string]attr.Value{
							"public": types.ObjectValueMust(
								publicNetworkTypes,
								map[string]attr.Value{
									"enabled": types.BoolValue(true),
									"port":    types.Int32Value(8080),
									"acl": types.ListValueMust(types.StringType, []attr.Value{
										types.StringValue("1.1.1.1/32"),
									}),
								},
							),
						},
					),
					Scaling: types.ObjectValueMust(
						scalingTypes,
						map[string]attr.Value{
							"manual": types.ObjectValueMust(
								manualScalingTypes,
								map[string]attr.Value{
									"instances": types.Int32Value(2),
								},
							),
							"auto": types.ObjectNull(autoScalingTypes),
						},
					),
					Status: types.StringValue("CURRENT_STATUS_RUNNING"),
					Urls: types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("https://test-app.com"),
					}),
				},
			},
			true,
		},
		{
			"nil_response",
			nil,
			"eu01",
			ResourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			state := &ResourceModel{
				Model: Model{
					ProjectId:     tt.expected.ProjectId,
					EnvironmentId: tt.expected.EnvironmentId,
				},
			}
			if !tt.isValid {
				state.ProjectId = types.StringValue("pid")
				state.EnvironmentId = types.StringValue("env-id")
			}

			err := mapFields(context.Background(), tt.input, state, tt.region)
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

func TestMapContainers(t *testing.T) {
	tests := []struct {
		name     string
		input    []scaSdk.Container
		expected *ResourceModel
		isValid  bool
	}{
		{
			"complex_containers_with_envs_and_missing_args",
			[]scaSdk.Container{
				{
					Name:   "app1",
					Image:  "nginx",
					Cpu:    conversion.Int32ValueToPointer(types.Int32Value(100)),
					Memory: conversion.Int32ValueToPointer(types.Int32Value(256)),
					EnvironmentVariables: []scaSdk.EnvVar{
						{
							Key:    "PLAINTEXT_VAR",
							Value:  "foo",
							Origin: new(scaSdk.ENVVARTYPE_ENV_FROM_SOURCE_TYPE_MANUAL),
						},
						{
							Key:    "SECRET_VAR",
							Value:  "bar_secret",
							Origin: new(scaSdk.ENVVARTYPE_ENV_FROM_SOURCE_TYPE_SECRET),
						},
						{
							Key:    "UNKNOWN_VAR",
							Value:  "ignored",
							Origin: new(scaSdk.EnvVarType("UNKNOWN_TYPE")), // Should trigger tflog.Warn but not fail
						},
					},
				},
			},
			&ResourceModel{
				Model: Model{
					Containers: types.ListValueMust(types.ObjectType{AttrTypes: containerTypes}, []attr.Value{
						types.ObjectValueMust(containerTypes, map[string]attr.Value{
							"name":       types.StringValue("app1"),
							"image":      types.StringValue("nginx"),
							"cpu_millis": types.Int32Value(100),
							"memory_mb":  types.Int32Value(256),
							"command":    types.ListNull(types.StringType),
							"args":       types.ListNull(types.StringType),
							"env": types.ObjectValueMust(envTypes, map[string]attr.Value{
								"from_value": types.MapValueMust(types.StringType, map[string]attr.Value{
									"PLAINTEXT_VAR": types.StringValue("foo"),
								}),
								"from_secret_ref": types.MapValueMust(types.StringType, map[string]attr.Value{
									"SECRET_VAR": types.StringValue("bar_secret"),
								}),
							}),
						}),
					}),
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &ResourceModel{}
			err := mapContainers(context.Background(), tt.input, m)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(m.Containers, tt.expected.Containers)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapNetwork(t *testing.T) {
	tests := []struct {
		name     string
		input    scaSdk.Network
		expected *ResourceModel
		isValid  bool
	}{
		{
			"disabled_public_network",
			scaSdk.Network{
				PublicIngress: false,
			},
			&ResourceModel{
				Model: Model{
					Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"public": types.ObjectNull(publicNetworkTypes),
					}),
				},
			},
			true,
		},
		{
			"enabled_public_network",
			scaSdk.Network{
				PublicIngress: true,
				Port:          conversion.Int32ValueToPointer(types.Int32Value(8080)),
				IngressAcl:    []string{"0.0.0.0/0"},
			},
			&ResourceModel{
				Model: Model{
					Network: types.ObjectValueMust(networkTypes, map[string]attr.Value{
						"public": types.ObjectValueMust(publicNetworkTypes, map[string]attr.Value{
							"enabled": types.BoolValue(true),
							"port":    types.Int32Value(8080),
							"acl": types.ListValueMust(types.StringType, []attr.Value{
								types.StringValue("0.0.0.0/0"),
							}),
						}),
					}),
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &ResourceModel{}
			err := mapNetwork(context.Background(), tt.input, m)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(m.Network, tt.expected.Network)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapScaling(t *testing.T) {
	tests := []struct {
		name     string
		input    scaSdk.Scaling
		expected *ResourceModel
		isValid  bool
	}{
		{
			"manual_scaling",
			scaSdk.Scaling{
				Type: scaSdk.SCALINGTYPE_SCALING_TYPE_MANUAL,
				ManualScaling: &scaSdk.ManualScaling{
					Instances: 2,
				},
			},
			&ResourceModel{
				Model: Model{
					Scaling: types.ObjectValueMust(scalingTypes, map[string]attr.Value{
						"manual": types.ObjectValueMust(manualScalingTypes, map[string]attr.Value{
							"instances": types.Int32Value(2),
						}),
						"auto": types.ObjectNull(autoScalingTypes),
					}),
				},
			},
			true,
		},
		{
			"auto_scaling_with_http_and_native_rules",
			scaSdk.Scaling{
				Type: scaSdk.SCALINGTYPE_SCALING_TYPE_AUTO,
				AutoScaling: &scaSdk.AutoScaling{
					MinInstances:     1,
					MaxInstances:     5,
					AllowScaleToZero: conversion.BoolValueToPointer(types.BoolValue(true)),
					Rules: []scaSdk.ScaleRule{
						{
							Name: "http-rule",
							Type: scaSdk.RULETYPE_RULE_TYPE_HTTP,
							HttpRule: &scaSdk.HttpScaleRule{
								Concurrency: conversion.Int32ValueToPointer(types.Int32Value(10)),
								Rps:         conversion.Int32ValueToPointer(types.Int32Value(50)),
							},
						},
						{
							Name: "keda-rule",
							Type: scaSdk.RULETYPE_RULE_TYPE_CUSTOM,
							CustomRule: &scaSdk.CustomScaleRule{
								Type: "rabbitmq",
								Parameters: []scaSdk.CustomRuleParameter{
									{Name: "queueName", Value: "tasks"},
								},
								SecretsMapping: []scaSdk.CustomRuleSecretMapping{
									{Parameter: "host", Secret: "rabbitmq-host"},
								},
							},
						},
					},
				},
			},
			&ResourceModel{
				Model: Model{
					Scaling: types.ObjectValueMust(scalingTypes, map[string]attr.Value{
						"manual": types.ObjectNull(manualScalingTypes),
						"auto": types.ObjectValueMust(autoScalingTypes, map[string]attr.Value{
							"min_instances":       types.Int32Value(1),
							"max_instances":       types.Int32Value(5),
							"allow_scale_to_zero": types.BoolValue(true),
							"http_rule": types.ObjectValueMust(httpRuleTypes, map[string]attr.Value{
								"name":        types.StringValue("http-rule"),
								"concurrency": types.Int32Value(10),
								"rps":         types.Int32Value(50),
							}),
							"native_rules": types.ListValueMust(types.ObjectType{AttrTypes: nativeRuleTypes}, []attr.Value{
								types.ObjectValueMust(nativeRuleTypes, map[string]attr.Value{
									"name":    types.StringValue("keda-rule"),
									"trigger": types.StringValue("rabbitmq"),
									"parameters": types.MapValueMust(types.StringType, map[string]attr.Value{
										"queueName": types.StringValue("tasks"),
									}),
									"secrets_mapping": types.MapValueMust(types.StringType, map[string]attr.Value{
										"host": types.StringValue("rabbitmq-host"),
									}),
								}),
							}),
						}),
					}),
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &ResourceModel{}
			err := mapScaling(context.Background(), tt.input, m)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(m.Scaling, tt.expected.Scaling)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapRuntimeStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    *scaSdk.RuntimeStatus
		expected *ResourceModel
		isValid  bool
	}{
		{
			"nil_status",
			nil,
			&ResourceModel{
				Model: Model{
					Status: types.StringNull(),
					Urls:   types.ListNull(types.StringType),
				},
			},
			true,
		},
		{
			"populated_status",
			&scaSdk.RuntimeStatus{
				CurrentStatus: new(scaSdk.CURRENTSTATUS_CURRENT_STATUS_PROGRESSING),
				Urls:          []string{"https://foo.bar"},
			},
			&ResourceModel{
				Model: Model{
					Status: types.StringValue(string(scaSdk.CURRENTSTATUS_CURRENT_STATUS_PROGRESSING)),
					Urls: types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("https://foo.bar"),
					}),
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &ResourceModel{
				Model: Model{
					Status: types.StringNull(),
					Urls:   types.ListNull(types.StringType),
				},
			}
			err := mapRuntimeStatus(context.Background(), tt.input, m)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(m.Status, tt.expected.Status)
				if diff != "" {
					t.Fatalf("Status does not match: %s", diff)
				}
				diff = cmp.Diff(m.Urls, tt.expected.Urls)
				if diff != "" {
					t.Fatalf("URLs do not match: %s", diff)
				}
			}
		})
	}
}

func TestToAPIContainers(t *testing.T) {
	tests := []struct {
		name     string
		input    types.List
		expected []scaSdk.Container
		isValid  bool
	}{
		{
			"null_list",
			types.ListNull(types.ObjectType{AttrTypes: containerTypes}),
			nil,
			false,
		},
		{
			"valid_container_with_args",
			types.ListValueMust(types.ObjectType{AttrTypes: containerTypes}, []attr.Value{
				types.ObjectValueMust(containerTypes, map[string]attr.Value{
					"name":       types.StringValue("app"),
					"image":      types.StringValue("nginx"),
					"cpu_millis": types.Int32Value(1000),
					"memory_mb":  types.Int32Value(512),
					"command": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("run"),
					}),
					"args": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("--verbose"),
					}),
					"env": types.ObjectNull(envTypes),
				}),
			}),
			[]scaSdk.Container{
				{
					Name:    "app",
					Image:   "nginx",
					Cpu:     conversion.Int32ValueToPointer(types.Int32Value(1000)),
					Memory:  conversion.Int32ValueToPointer(types.Int32Value(512)),
					Command: []string{"run"},
					Args:    []string{"--verbose"},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := toAPIContainers(context.Background(), tt.input)
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

func TestToAPIEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    types.Object
		expected []scaSdk.EnvVar
		isValid  bool
	}{
		{
			"null_object",
			types.ObjectNull(envTypes),
			nil,
			true,
		},
		{
			"valid_env_mapping",
			types.ObjectValueMust(envTypes, map[string]attr.Value{
				"from_value": types.MapValueMust(types.StringType, map[string]attr.Value{
					"VAR_1": types.StringValue("value1"),
				}),
				"from_secret_ref": types.MapValueMust(types.StringType, map[string]attr.Value{
					"VAR_2": types.StringValue("secret1"),
				}),
			}),
			[]scaSdk.EnvVar{
				{
					Key:    "VAR_1",
					Value:  "value1",
					Origin: new(scaSdk.ENVVARTYPE_ENV_FROM_SOURCE_TYPE_MANUAL),
				},
				{
					Key:    "VAR_2",
					Value:  "secret1",
					Origin: new(scaSdk.ENVVARTYPE_ENV_FROM_SOURCE_TYPE_SECRET),
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, err := toAPIEnv(context.Background(), tt.input)
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

func TestToAPINetwork(t *testing.T) {
	tests := []struct {
		name     string
		input    types.Object
		expected *scaSdk.Network
		isValid  bool
	}{
		{
			"null_object",
			types.ObjectNull(networkTypes),
			nil,
			true,
		},
		{
			"valid_network",
			types.ObjectValueMust(networkTypes, map[string]attr.Value{
				"public": types.ObjectValueMust(publicNetworkTypes, map[string]attr.Value{
					"enabled": types.BoolValue(true),
					"port":    types.Int32Value(8080),
					"acl": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("1.1.1.1/32"),
					}),
				}),
			}),
			&scaSdk.Network{
				PublicIngress: true,
				Port:          conversion.Int32ValueToPointer(types.Int32Value(8080)),
				IngressAcl:    []string{"1.1.1.1/32"},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := toAPINetwork(context.Background(), tt.input)
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

func TestToAPIScaling(t *testing.T) {
	tests := []struct {
		name     string
		input    types.Object
		expected *scaSdk.Scaling
		isValid  bool
	}{
		{
			"null_object",
			types.ObjectNull(scalingTypes),
			nil,
			true,
		},
		{
			"manual_scaling",
			types.ObjectValueMust(scalingTypes, map[string]attr.Value{
				"manual": types.ObjectValueMust(manualScalingTypes, map[string]attr.Value{
					"instances": types.Int32Value(3),
				}),
				"auto": types.ObjectNull(autoScalingTypes),
			}),
			&scaSdk.Scaling{
				Type: scaSdk.SCALINGTYPE_SCALING_TYPE_MANUAL,
				ManualScaling: &scaSdk.ManualScaling{
					Instances: 3,
				},
			},
			true,
		},
		{
			"auto_scaling_with_http_rule",
			types.ObjectValueMust(scalingTypes, map[string]attr.Value{
				"manual": types.ObjectNull(manualScalingTypes),
				"auto": types.ObjectValueMust(autoScalingTypes, map[string]attr.Value{
					"min_instances":       types.Int32Value(1),
					"max_instances":       types.Int32Value(5),
					"allow_scale_to_zero": types.BoolValue(true),
					"http_rule": types.ObjectValueMust(httpRuleTypes, map[string]attr.Value{
						"name":        types.StringValue("http-scaler"),
						"concurrency": types.Int32Value(10),
						"rps":         types.Int32Value(100),
					}),
					"native_rules": types.ListNull(types.ObjectType{AttrTypes: nativeRuleTypes}),
				}),
			}),
			&scaSdk.Scaling{
				Type: scaSdk.SCALINGTYPE_SCALING_TYPE_AUTO,
				AutoScaling: &scaSdk.AutoScaling{
					MinInstances:     1,
					MaxInstances:     5,
					AllowScaleToZero: conversion.BoolValueToPointer(types.BoolValue(true)),
					Rules: []scaSdk.ScaleRule{
						{
							Name: "http-scaler",
							Type: scaSdk.RULETYPE_RULE_TYPE_HTTP,
							HttpRule: &scaSdk.HttpScaleRule{
								Concurrency: conversion.Int32ValueToPointer(types.Int32Value(10)),
								Rps:         conversion.Int32ValueToPointer(types.Int32Value(100)),
							},
						},
					},
				},
			},
			true,
		},
		{
			"auto_scaling_with_native_rules",
			types.ObjectValueMust(scalingTypes, map[string]attr.Value{
				"manual": types.ObjectNull(manualScalingTypes),
				"auto": types.ObjectValueMust(autoScalingTypes, map[string]attr.Value{
					"min_instances":       types.Int32Value(2),
					"max_instances":       types.Int32Value(10),
					"allow_scale_to_zero": types.BoolNull(),
					"http_rule":           types.ObjectNull(httpRuleTypes),
					"native_rules": types.ListValueMust(types.ObjectType{AttrTypes: nativeRuleTypes}, []attr.Value{
						types.ObjectValueMust(nativeRuleTypes, map[string]attr.Value{
							"name":    types.StringValue("rabbitmq-trigger"),
							"trigger": types.StringValue("rabbitmq"),
							"parameters": types.MapValueMust(types.StringType, map[string]attr.Value{
								"queueName": types.StringValue("tasks"),
							}),
							"secrets_mapping": types.MapValueMust(types.StringType, map[string]attr.Value{
								"host": types.StringValue("rabbitmq-host"),
							}),
						}),
					}),
				}),
			}),
			&scaSdk.Scaling{
				Type: scaSdk.SCALINGTYPE_SCALING_TYPE_AUTO,
				AutoScaling: &scaSdk.AutoScaling{
					MinInstances: 2,
					MaxInstances: 10,
					Rules: []scaSdk.ScaleRule{
						{
							Name: "rabbitmq-trigger",
							Type: scaSdk.RULETYPE_RULE_TYPE_CUSTOM,
							CustomRule: &scaSdk.CustomScaleRule{
								Type: "rabbitmq",
								Parameters: []scaSdk.CustomRuleParameter{
									{Name: "queueName", Value: "tasks"},
								},
								SecretsMapping: []scaSdk.CustomRuleSecretMapping{
									{Parameter: "host", Secret: "rabbitmq-host"},
								},
							},
						},
					},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, err := toAPIScaling(context.Background(), tt.input)
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

func TestIsPublicIngressEnabled(t *testing.T) {
	tests := []struct {
		name     string
		input    types.Object
		expected bool
	}{
		{
			"null_network",
			types.ObjectNull(networkTypes),
			false,
		},
		{
			"unknown_network",
			types.ObjectUnknown(networkTypes),
			false,
		},
		{
			"network_with_null_public",
			types.ObjectValueMust(networkTypes, map[string]attr.Value{
				"public": types.ObjectNull(publicNetworkTypes),
			}),
			false,
		},
		{
			"network_with_public_disabled",
			types.ObjectValueMust(networkTypes, map[string]attr.Value{
				"public": types.ObjectValueMust(publicNetworkTypes, map[string]attr.Value{
					"enabled": types.BoolValue(false),
					"port":    types.Int32Null(),
					"acl":     types.ListNull(types.StringType),
				}),
			}),
			false,
		},
		{
			"network_with_public_enabled",
			types.ObjectValueMust(networkTypes, map[string]attr.Value{
				"public": types.ObjectValueMust(publicNetworkTypes, map[string]attr.Value{
					"enabled": types.BoolValue(true),
					"port":    types.Int32Value(8080),
					"acl":     types.ListNull(types.StringType),
				}),
			}),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diags diag.Diagnostics
			result := isPublicIngressEnabled(context.Background(), tt.input, &diags)

			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags.Errors())
			}
			if result != tt.expected {
				t.Fatalf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
