package postgresflexalpha

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	postgresflexalphadatasource "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/postgresflexalpha/instance/datasources_gen"
	postgresflexalpharesource "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/postgresflexalpha/instance/resources_gen"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
)

func mapGetInstanceResponseToModel(ctx context.Context, m *postgresflexalpharesource.InstanceModel, resp *postgresflex.GetInstanceResponse) error {
	m.BackupSchedule = types.StringValue(resp.GetBackupSchedule())
	// need to leave out encryption, as the GetInstance endpoint does not provide it
	//m.Encryption = postgresflexalpharesource.NewEncryptionValueMust(
	//	m.Encryption.AttributeTypes(ctx),
	//	map[string]attr.Value{
	//		"kek_key_id":      types.StringValue(resp.Encryption.GetKekKeyId()),
	//		"kek_key_ring_id": types.StringValue(resp.Encryption.GetKekKeyRingId()),
	//		"kek_key_version": types.StringValue(resp.Encryption.GetKekKeyVersion()),
	//		"service_account": types.StringValue(resp.Encryption.GetServiceAccount()),
	//	},
	//)
	m.FlavorId = types.StringValue(resp.GetFlavorId())
	if m.Id.IsNull() || m.Id.IsUnknown() {
		m.Id = utils.BuildInternalTerraformId(m.ProjectId.ValueString(), m.Region.ValueString(), m.InstanceId.ValueString())
	}
	m.InstanceId = types.StringPointerValue(resp.Id)
	m.IsDeletable = types.BoolValue(resp.GetIsDeletable())
	m.Name = types.StringValue(resp.GetName())

	netAcl, diags := types.ListValueFrom(ctx, types.StringType, resp.Network.GetAcl())
	if diags.HasError() {
		return fmt.Errorf("failed converting network acl from response")
	}

	net, diags := postgresflexalpharesource.NewNetworkValue(
		postgresflexalpharesource.NetworkValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"access_scope":     types.StringValue(string(resp.Network.GetAccessScope())),
			"acl":              netAcl,
			"instance_address": types.StringValue(resp.Network.GetInstanceAddress()),
			"router_address":   types.StringValue(resp.Network.GetRouterAddress()),
		},
	)
	if diags.HasError() {
		return fmt.Errorf("failed converting network from response")
	}

	m.Network = net
	m.Replicas = types.Int64Value(int64(resp.GetReplicas()))
	m.RetentionDays = types.Int64Value(resp.GetRetentionDays())
	m.Status = types.StringValue(string(resp.GetStatus()))

	storage, diags := postgresflexalpharesource.NewStorageValue(
		postgresflexalpharesource.StorageValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"performance_class": types.StringValue(resp.Storage.GetPerformanceClass()),
			"size":              types.Int64Value(resp.Storage.GetSize()),
		},
	)
	if diags.HasError() {
		return fmt.Errorf("failed converting storage from response")
	}
	m.Storage = storage
	m.Version = types.StringValue(resp.GetVersion())
	return nil
}

func mapGetDataInstanceResponseToModel(ctx context.Context, m *postgresflexalphadatasource.InstanceModel, resp *postgresflex.GetInstanceResponse) error {
	m.BackupSchedule = types.StringValue(resp.GetBackupSchedule())
	//m.Encryption = postgresflexalpharesource.EncryptionValue{
	//	KekKeyId:       types.StringValue(resp.Encryption.GetKekKeyId()),
	//	KekKeyRingId:   types.StringValue(resp.Encryption.GetKekKeyRingId()),
	//	KekKeyVersion:  types.StringValue(resp.Encryption.GetKekKeyVersion()),
	//	ServiceAccount: types.StringValue(resp.Encryption.GetServiceAccount()),
	//}
	m.FlavorId = types.StringValue(resp.GetFlavorId())
	m.Id = utils.BuildInternalTerraformId(m.ProjectId.ValueString(), m.Region.ValueString(), m.InstanceId.ValueString())
	m.InstanceId = types.StringPointerValue(resp.Id)
	m.IsDeletable = types.BoolValue(resp.GetIsDeletable())
	m.Name = types.StringValue(resp.GetName())
	netAcl, diags := types.ListValueFrom(ctx, types.StringType, resp.Network.GetAcl())
	if diags.HasError() {
		return fmt.Errorf("failed converting network acl from response")
	}

	net, diags := postgresflexalphadatasource.NewNetworkValue(
		postgresflexalphadatasource.NetworkValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"access_scope":     types.StringValue(string(resp.Network.GetAccessScope())),
			"acl":              netAcl,
			"instance_address": types.StringValue(resp.Network.GetInstanceAddress()),
			"router_address":   types.StringValue(resp.Network.GetRouterAddress()),
		},
	)
	if diags.HasError() {
		return fmt.Errorf("failed converting network from response")
	}

	m.Network = net
	m.Replicas = types.Int64Value(int64(resp.GetReplicas()))
	m.RetentionDays = types.Int64Value(resp.GetRetentionDays())
	m.Status = types.StringValue(string(resp.GetStatus()))
	storage, diags := postgresflexalphadatasource.NewStorageValue(
		postgresflexalphadatasource.StorageValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"performance_class": types.StringValue(resp.Storage.GetPerformanceClass()),
			"size":              types.Int64Value(resp.Storage.GetSize()),
		},
	)
	if diags.HasError() {
		return fmt.Errorf("failed converting storage from response")
	}
	m.Storage = storage
	m.Version = types.StringValue(resp.GetVersion())
	return nil
}
