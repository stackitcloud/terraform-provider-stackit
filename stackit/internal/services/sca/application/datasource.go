package sca

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type Model struct {
	Id            types.String `tfsdk:"id"`
	ApplicationId types.String `tfsdk:"application_id"`
	ProjectId     types.String `tfsdk:"project_id"`
	Region        types.String `tfsdk:"region"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	DisplayName   types.String `tfsdk:"display_name"`
	Stopped       types.Bool   `tfsdk:"stopped"`
	Containers    types.List   `tfsdk:"containers"`
	Network       types.Object `tfsdk:"network"`
	Scaling       types.Object `tfsdk:"scaling"`
	Urls          types.List   `tfsdk:"urls"`
	Status        types.String `tfsdk:"status"`
}

// Struct corresponding to ResourceModel.Containers[i]
type containerModel struct {
	Name      types.String `tfsdk:"name"`
	Image     types.String `tfsdk:"image"`
	CPUMillis types.Int32  `tfsdk:"cpu_millis"`
	MemoryMB  types.Int32  `tfsdk:"memory_mb"`
	Command   types.List   `tfsdk:"command"`
	Args      types.List   `tfsdk:"args"`
	Env       types.Object `tfsdk:"env"`
}

// Types corresponding to containerModel
var containerTypes = map[string]attr.Type{
	"name":       basetypes.StringType{},
	"image":      basetypes.StringType{},
	"cpu_millis": basetypes.Int32Type{},
	"memory_mb":  basetypes.Int32Type{},
	"command":    basetypes.ListType{ElemType: types.StringType},
	"args":       basetypes.ListType{ElemType: types.StringType},
	"env":        basetypes.ObjectType{AttrTypes: envTypes},
}

// Struct corresponding to ResourceModel.Containers[i].Env
type envModel struct {
	FromValue     types.Map `tfsdk:"from_value"`
	FromSecretRef types.Map `tfsdk:"from_secret_ref"`
}

var envTypes = map[string]attr.Type{
	"from_value":      basetypes.MapType{ElemType: types.StringType},
	"from_secret_ref": basetypes.MapType{ElemType: types.StringType},
}

// Struct corresponding to ResourceModel.Network
type networkModel struct {
	Public types.Object `tfsdk:"public"`
	// Internal feature to be implemented
}

// Types corresponding to networkModel
var networkTypes = map[string]attr.Type{
	"public": basetypes.ObjectType{AttrTypes: publicNetworkTypes},
}

// Struct corresponding to Model.Network.Public
type publicNetworkModel struct {
	Enabled types.Bool  `tfsdk:"enabled"`
	Port    types.Int32 `tfsdk:"port"`
	ACL     types.List  `tfsdk:"acl"`
}

// Types corresponding to publicNetworkModel
var publicNetworkTypes = map[string]attr.Type{
	"enabled": basetypes.BoolType{},
	"port":    basetypes.Int32Type{},
	"acl":     basetypes.ListType{ElemType: types.StringType},
}

// Struct corresponding to ResourceModel.Scaling
type scalingModel struct {
	Auto   types.Object `tfsdk:"auto"`   // SingleNested
	Manual types.Object `tfsdk:"manual"` // SingleNested
}

// Types corresponding to scalingModel
var scalingTypes = map[string]attr.Type{
	"auto":   basetypes.ObjectType{AttrTypes: autoScalingTypes},
	"manual": basetypes.ObjectType{AttrTypes: manualScalingTypes},
}

// Struct corresponding to Model.Scaling.Manual
type manualScalingModel struct {
	Instances types.Int32 `tfsdk:"instances"`
}

// Types corresponding to manualScalingModel
var manualScalingTypes = map[string]attr.Type{
	"instances": basetypes.Int32Type{},
}

// Struct corresponding to Model.Scaling.Auto
type autoScalingModel struct {
	MinInstances     types.Int32  `tfsdk:"min_instances"`
	MaxInstances     types.Int32  `tfsdk:"max_instances"`
	AllowScaleToZero types.Bool   `tfsdk:"allow_scale_to_zero"`
	HTTPRule         types.Object `tfsdk:"http_rule"`
	NativeRules      types.List   `tfsdk:"native_rules"`
}

var autoScalingTypes = map[string]attr.Type{
	"min_instances":       basetypes.Int32Type{},
	"max_instances":       basetypes.Int32Type{},
	"allow_scale_to_zero": basetypes.BoolType{},
	"http_rule":           basetypes.ObjectType{AttrTypes: httpRuleTypes},
	"native_rules":        basetypes.ListType{ElemType: basetypes.ObjectType{AttrTypes: nativeRuleTypes}},
}

// Struct corresponding to Model.Scaling.Auto.HTTPRule
type httpRuleModel struct {
	Name        types.String `tfsdk:"name"`
	Concurrency types.Int32  `tfsdk:"concurrency"`
	Rps         types.Int32  `tfsdk:"rps"`
}

var httpRuleTypes = map[string]attr.Type{
	"name":        basetypes.StringType{},
	"concurrency": basetypes.Int32Type{},
	"rps":         basetypes.Int32Type{},
}

// Struct corresponding to Model.Scaling.Auto.NativeRules[i]
type nativeRuleModel struct {
	Name           types.String `tfsdk:"name"`
	Trigger        types.String `tfsdk:"trigger"`
	Parameters     types.Map    `tfsdk:"parameters"`
	SecretsMapping types.Map    `tfsdk:"secrets_mapping"`
}

var nativeRuleTypes = map[string]attr.Type{
	"name":            basetypes.StringType{},
	"trigger":         basetypes.StringType{},
	"parameters":      basetypes.MapType{ElemType: types.StringType},
	"secrets_mapping": basetypes.MapType{ElemType: types.StringType},
}
