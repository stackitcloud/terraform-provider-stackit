package sqlserverflex

import (
	"context"
	"fmt"
	"math"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sqlserverflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
)

type sqlserverflexClient interface {
	GetFlavorsRequestExecute(ctx context.Context, projectId, region string, page, size *int64, sort *sqlserverflex.FlavorSort) (*sqlserverflex.GetFlavorsResponse, error)
}

func mapFields(ctx context.Context, resp *sqlserverflex.GetInstanceResponse, model *Model, storage *storageModel, encryption *encryptionModel, network *networkModel, region string) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	instance := resp

	var instanceId string
	if model.InstanceId.ValueString() != "" {
		instanceId = model.InstanceId.ValueString()
	} else if instance.Id != nil {
		instanceId = *instance.Id
	} else {
		return fmt.Errorf("instance id not present")
	}

	var storageValues map[string]attr.Value
	if instance.Storage == nil {
		storageValues = map[string]attr.Value{
			"class": storage.Class,
			"size":  storage.Size,
		}
	} else {
		storageValues = map[string]attr.Value{
			"class": types.StringValue(*instance.Storage.Class),
			"size":  types.Int64PointerValue(instance.Storage.Size),
		}
	}
	storageObject, diags := types.ObjectValue(storageTypes, storageValues)
	if diags.HasError() {
		return fmt.Errorf("creating storage: %w", core.DiagsToError(diags))
	}

	var encryptionValues map[string]attr.Value
	if instance.Encryption == nil {
		encryptionValues = map[string]attr.Value{
			"keyring_id":      encryption.KeyRingId,
			"key_id":          encryption.KeyId,
			"key_version":     encryption.KeyVersion,
			"service_account": encryption.ServiceAccount,
		}
	} else {
		encryptionValues = map[string]attr.Value{
			"keyring_id":      types.StringValue(*instance.Encryption.KekKeyRingId),
			"key_id":          types.StringValue(*instance.Encryption.KekKeyId),
			"key_version":     types.StringValue(*instance.Encryption.KekKeyVersion),
			"service_account": types.StringValue(*instance.Encryption.ServiceAccount),
		}
	}
	encryptionObject, diags := types.ObjectValue(encryptionTypes, encryptionValues)
	if diags.HasError() {
		return fmt.Errorf("creating encryption: %w", core.DiagsToError(diags))
	}

	var networkValues map[string]attr.Value
	if instance.Network == nil {
		networkValues = map[string]attr.Value{
			"acl":              network.ACL,
			"access_scope":     network.AccessScope,
			"instance_address": network.InstanceAddress,
			"router_address":   network.RouterAddress,
		}
	} else {
		aclList, diags := types.ListValueFrom(ctx, types.StringType, *instance.Network.Acl)
		if diags.HasError() {
			return fmt.Errorf("creating network (acl list): %w", core.DiagsToError(diags))
		}

		var routerAddress string
		if instance.Network.RouterAddress != nil {
			routerAddress = *instance.Network.RouterAddress
			diags.AddWarning("field missing while mapping fields", "router_address was empty in API response")
		}
		if instance.Network.InstanceAddress == nil {
			return fmt.Errorf("creating network: no instance address returned")
		}
		networkValues = map[string]attr.Value{
			"acl":              aclList,
			"access_scope":     types.StringValue(string(*instance.Network.AccessScope)),
			"instance_address": types.StringValue(*instance.Network.InstanceAddress),
			"router_address":   types.StringValue(routerAddress),
		}
	}
	networkObject, diags := types.ObjectValue(networkTypes, networkValues)
	if diags.HasError() {
		return fmt.Errorf("creating network: %w", core.DiagsToError(diags))
	}

	simplifiedModelBackupSchedule := utils.SimplifyBackupSchedule(model.BackupSchedule.ValueString())
	// If the value returned by the API is different from the one in the model after simplification,
	// we update the model so that it causes an error in Terraform
	if simplifiedModelBackupSchedule != types.StringPointerValue(instance.BackupSchedule).ValueString() {
		model.BackupSchedule = types.StringPointerValue(instance.BackupSchedule)
	}

	if instance.Replicas == nil {
		return fmt.Errorf("instance has no replicas set")
	}

	if instance.RetentionDays == nil {
		return fmt.Errorf("instance has no retention days set")
	}

	if instance.Version == nil {
		return fmt.Errorf("instance has no version set")
	}

	if instance.Edition == nil {
		return fmt.Errorf("instance has no edition set")
	}

	if instance.Status == nil {
		return fmt.Errorf("instance has no status set")
	}

	if instance.IsDeletable == nil {
		return fmt.Errorf("instance has no IsDeletable set")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, instanceId)
	model.InstanceId = types.StringValue(instanceId)
	model.Name = types.StringPointerValue(instance.Name)
	model.FlavorId = types.StringPointerValue(instance.FlavorId)
	model.Replicas = types.Int64Value(int64(*instance.Replicas))
	model.Storage = storageObject
	model.Version = types.StringValue(string(*instance.Version))
	model.Edition = types.StringValue(string(*instance.Edition))
	model.Region = types.StringValue(region)
	model.Encryption = encryptionObject
	model.Network = networkObject
	model.RetentionDays = types.Int64Value(*instance.RetentionDays)
	model.Status = types.StringValue(string(*instance.Status))
	model.IsDeletable = types.BoolValue(*instance.IsDeletable)
	return nil
}

func toCreatePayload(model *Model, storage *storageModel, encryption *encryptionModel, network *networkModel) (*sqlserverflex.CreateInstanceRequestPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	storagePayload := &sqlserverflex.CreateInstanceRequestPayloadGetStorageArgType{}
	if storage != nil {
		storagePayload.Class = conversion.StringValueToPointer(storage.Class)
		storagePayload.Size = conversion.Int64ValueToPointer(storage.Size)
	}

	encryptionPayload := &sqlserverflex.CreateInstanceRequestPayloadGetEncryptionArgType{}
	if encryption != nil {
		encryptionPayload.KekKeyId = conversion.StringValueToPointer(encryption.KeyId)
		encryptionPayload.KekKeyVersion = conversion.StringValueToPointer(encryption.KeyVersion)
		encryptionPayload.KekKeyRingId = conversion.StringValueToPointer(encryption.KeyRingId)
		encryptionPayload.ServiceAccount = conversion.StringValueToPointer(encryption.ServiceAccount)
	}

	var aclElements []string
	if network != nil && !network.ACL.IsNull() && !network.ACL.IsUnknown() {
		aclElements = make([]string, 0, len(network.ACL.Elements()))
		diags := network.ACL.ElementsAs(context.TODO(), &aclElements, false)
		if diags.HasError() {
			return nil, fmt.Errorf("creating network: %w", core.DiagsToError(diags))
		}
	}

	networkPayload := &sqlserverflex.CreateInstanceRequestPayloadGetNetworkArgType{}
	if network != nil {
		networkPayload.AccessScope = sqlserverflex.CreateInstanceRequestPayloadNetworkGetAccessScopeAttributeType(conversion.StringValueToPointer(network.AccessScope))
		networkPayload.Acl = &aclElements
	}

	return &sqlserverflex.CreateInstanceRequestPayload{
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		Encryption:     encryptionPayload,
		FlavorId:       conversion.StringValueToPointer(model.FlavorId),
		Name:           conversion.StringValueToPointer(model.Name),
		Network:        networkPayload,
		RetentionDays:  conversion.Int64ValueToPointer(model.RetentionDays),
		Storage:        storagePayload,
		Version:        sqlserverflex.CreateInstanceRequestPayloadGetVersionAttributeType(conversion.StringValueToPointer(model.Version)),
	}, nil
}

func toUpdatePartiallyPayload(model *Model, storage *storageModel, network *networkModel) (*sqlserverflex.UpdateInstancePartiallyRequestPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	storagePayload := &sqlserverflex.UpdateInstanceRequestPayloadGetStorageArgType{}
	if storage != nil {
		storagePayload.Size = conversion.Int64ValueToPointer(storage.Size)
	}

	var aclElements []string
	if network != nil && !network.ACL.IsNull() && !network.ACL.IsUnknown() {
		aclElements = make([]string, 0, len(network.ACL.Elements()))
		diags := network.ACL.ElementsAs(context.TODO(), &aclElements, false)
		if diags.HasError() {
			return nil, fmt.Errorf("creating network: %w", core.DiagsToError(diags))
		}
	}

	networkPayload := &sqlserverflex.UpdateInstancePartiallyRequestPayloadGetNetworkArgType{}
	if network != nil {
		networkPayload.AccessScope = sqlserverflex.UpdateInstancePartiallyRequestPayloadNetworkGetAccessScopeAttributeType(conversion.StringValueToPointer(network.AccessScope))
		networkPayload.Acl = &aclElements
	}

	if model.Replicas.ValueInt64() > math.MaxInt32 {
		return nil, fmt.Errorf("replica count too big: %d", model.Replicas.ValueInt64())
	}
	replCount := int32(model.Replicas.ValueInt64()) // nolint:gosec // check is performed above
	return &sqlserverflex.UpdateInstancePartiallyRequestPayload{
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       conversion.StringValueToPointer(model.FlavorId),
		Name:           conversion.StringValueToPointer(model.Name),
		Network:        networkPayload,
		Replicas:       sqlserverflex.UpdateInstancePartiallyRequestPayloadGetReplicasAttributeType(&replCount),
		RetentionDays:  conversion.Int64ValueToPointer(model.RetentionDays),
		Storage:        storagePayload,
		Version:        sqlserverflex.UpdateInstancePartiallyRequestPayloadGetVersionAttributeType(conversion.StringValueToPointer(model.Version)),
	}, nil
}

func toUpdatePayload(model *Model, storage *storageModel, network *networkModel) (*sqlserverflex.UpdateInstanceRequestPayload, error) {
	return &sqlserverflex.UpdateInstanceRequestPayload{
		BackupSchedule: nil,
		FlavorId:       nil,
		Name:           nil,
		Network:        nil,
		Replicas:       nil,
		RetentionDays:  nil,
		Storage:        nil,
		Version:        nil,
	}, nil
}
