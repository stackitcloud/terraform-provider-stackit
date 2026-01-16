package postgresflexalpha

import (
	"context"
	"fmt"
	"math"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
)

func mapFields(
	ctx context.Context,
	resp *postgresflex.GetInstanceResponse,
	model *Model,
	storage *storageModel,
	encryption *encryptionModel,
	network *networkModel,
	region string,
) error {
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

		networkValues = map[string]attr.Value{
			"acl":              aclList,
			"access_scope":     types.StringPointerValue((*string)(instance.Network.AccessScope)),
			"instance_address": types.StringPointerValue(instance.Network.InstanceAddress),
			"router_address":   types.StringPointerValue(instance.Network.RouterAddress),
		}
	}
	networkObject, diags := types.ObjectValue(networkTypes, networkValues)
	if diags.HasError() {
		return fmt.Errorf("creating network: %w", core.DiagsToError(diags))
	}

	var storageValues map[string]attr.Value
	if instance.Storage == nil {
		storageValues = map[string]attr.Value{
			"class": storage.Class,
			"size":  storage.Size,
		}
	} else {
		storageValues = map[string]attr.Value{
			"class": types.StringValue(*instance.Storage.PerformanceClass),
			"size":  types.Int64PointerValue(instance.Storage.Size),
		}
	}
	storageObject, diags := types.ObjectValue(storageTypes, storageValues)
	if diags.HasError() {
		return fmt.Errorf("creating storage: %w", core.DiagsToError(diags))
	}

	if instance.Replicas == nil {
		diags.AddError("error mapping fields", "replicas is nil")
		return fmt.Errorf("replicas is nil")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, instanceId)
	model.InstanceId = types.StringValue(instanceId)
	model.Name = types.StringPointerValue(instance.Name)
	model.BackupSchedule = types.StringPointerValue(instance.BackupSchedule)
	model.FlavorId = types.StringPointerValue(instance.FlavorId)
	model.Replicas = types.Int64Value(int64(*instance.Replicas))
	model.Storage = storageObject
	model.Version = types.StringPointerValue(instance.Version)
	model.Region = types.StringValue(region)
	model.Encryption = encryptionObject
	model.Network = networkObject
	return nil
}

func toCreatePayload(
	model *Model,
	storage *storageModel,
	enc *encryptionModel,
	net *networkModel,
) (*postgresflex.CreateInstanceRequestPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}

	var replVal int32
	if !model.Replicas.IsNull() && !model.Replicas.IsUnknown() {
		if model.Replicas.ValueInt64() > math.MaxInt32 {
			return nil, fmt.Errorf("replica count too big: %d", model.Replicas.ValueInt64())
		}
		replVal = int32(model.Replicas.ValueInt64()) // nolint:gosec // check is performed above
	}

	storagePayload := &postgresflex.CreateInstanceRequestPayloadGetStorageArgType{
		PerformanceClass: conversion.StringValueToPointer(storage.Class),
		Size:             conversion.Int64ValueToPointer(storage.Size),
	}

	encryptionPayload := &postgresflex.CreateInstanceRequestPayloadGetEncryptionArgType{}
	if enc != nil {
		encryptionPayload.KekKeyId = conversion.StringValueToPointer(enc.KeyId)
		encryptionPayload.KekKeyVersion = conversion.StringValueToPointer(enc.KeyVersion)
		encryptionPayload.KekKeyRingId = conversion.StringValueToPointer(enc.KeyRingId)
		encryptionPayload.ServiceAccount = conversion.StringValueToPointer(enc.ServiceAccount)
	}

	var aclElements []string
	if net != nil && !net.ACL.IsNull() && !net.ACL.IsUnknown() {
		aclElements = make([]string, 0, len(net.ACL.Elements()))
		diags := net.ACL.ElementsAs(context.TODO(), &aclElements, false)
		if diags.HasError() {
			return nil, fmt.Errorf("creating network: %w", core.DiagsToError(diags))
		}
	}

	if len(aclElements) < 1 {
		return nil, fmt.Errorf("no acl elements found")
	}

	networkPayload := &postgresflex.CreateInstanceRequestPayloadGetNetworkArgType{}
	if net != nil {
		networkPayload = &postgresflex.CreateInstanceRequestPayloadGetNetworkArgType{
			AccessScope: postgresflex.InstanceNetworkGetAccessScopeAttributeType(conversion.StringValueToPointer(net.AccessScope)),
			Acl:         &aclElements,
		}
	}

	return &postgresflex.CreateInstanceRequestPayload{
		Acl:            &aclElements,
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		Encryption:     encryptionPayload,
		FlavorId:       conversion.StringValueToPointer(model.FlavorId),
		Name:           conversion.StringValueToPointer(model.Name),
		Network:        networkPayload,
		Replicas:       postgresflex.CreateInstanceRequestPayloadGetReplicasAttributeType(&replVal),
		RetentionDays:  conversion.Int64ValueToPointer(model.RetentionDays),
		Storage:        storagePayload,
		Version:        conversion.StringValueToPointer(model.Version),
	}, nil
}

func toUpdatePayload(
	model *Model,
	storage *storageModel,
	_ *networkModel,
) (*postgresflex.UpdateInstancePartiallyRequestPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}

	return &postgresflex.UpdateInstancePartiallyRequestPayload{
		// Acl: postgresflexalpha.UpdateInstancePartiallyRequestPayloadGetAclAttributeType{
		//	Items: &acl,
		// },
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       conversion.StringValueToPointer(model.FlavorId),
		Name:           conversion.StringValueToPointer(model.Name),
		// Replicas:       conversion.Int64ValueToPointer(model.Replicas),
		Storage: &postgresflex.StorageUpdate{
			Size: conversion.Int64ValueToPointer(storage.Size),
		},
		Version: conversion.StringValueToPointer(model.Version),
	}, nil
}
