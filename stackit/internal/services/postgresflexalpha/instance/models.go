package postgresflexalpha

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	InstanceId     types.String `tfsdk:"instance_id"`
	ProjectId      types.String `tfsdk:"project_id"`
	Name           types.String `tfsdk:"name"`
	BackupSchedule types.String `tfsdk:"backup_schedule"`
	Flavor         types.Object `tfsdk:"flavor"`
	Replicas       types.Int64  `tfsdk:"replicas"`
	Storage        types.Object `tfsdk:"storage"`
	Version        types.String `tfsdk:"version"`
	Region         types.String `tfsdk:"region"`
	Encryption     types.Object `tfsdk:"encryption"`
	Network        types.Object `tfsdk:"network"`
}

type encryptionModel struct {
	KeyRingId      types.String `tfsdk:"keyring_id"`
	KeyId          types.String `tfsdk:"key_id"`
	KeyVersion     types.String `tfsdk:"key_version"`
	ServiceAccount types.String `tfsdk:"service_account"`
}

var encryptionTypes = map[string]attr.Type{
	"keyring_id":      basetypes.StringType{},
	"key_id":          basetypes.StringType{},
	"key_version":     basetypes.StringType{},
	"service_account": basetypes.StringType{},
}

type networkModel struct {
	ACL             types.List   `tfsdk:"acl"`
	AccessScope     types.String `tfsdk:"access_scope"`
	InstanceAddress types.String `tfsdk:"instance_address"`
	RouterAddress   types.String `tfsdk:"router_address"`
}

var networkTypes = map[string]attr.Type{
	"acl":              basetypes.ListType{ElemType: types.StringType},
	"access_scope":     basetypes.StringType{},
	"instance_address": basetypes.StringType{},
	"router_address":   basetypes.StringType{},
}

// Struct corresponding to Model.Flavor
type flavorModel struct {
	Id          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	CPU         types.Int64  `tfsdk:"cpu"`
	RAM         types.Int64  `tfsdk:"ram"`
	NodeType    types.String `tfsdk:"node_type"`
}

// Types corresponding to flavorModel
var flavorTypes = map[string]attr.Type{
	"id":          basetypes.StringType{},
	"description": basetypes.StringType{},
	"cpu":         basetypes.Int64Type{},
	"ram":         basetypes.Int64Type{},
	"node_type":   basetypes.StringType{},
}

// Struct corresponding to Model.Storage
type storageModel struct {
	Class types.String `tfsdk:"class"`
	Size  types.Int64  `tfsdk:"size"`
}

// Types corresponding to storageModel
var storageTypes = map[string]attr.Type{
	"class": basetypes.StringType{},
	"size":  basetypes.Int64Type{},
}
