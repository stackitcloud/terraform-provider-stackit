package postgresflexalpha

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
)

type postgresflexClient interface {
	GetFlavorsRequestExecute(ctx context.Context, projectId string, region string) (*postgresflex.GetFlavorsResponse, error)
}

func mapFields(ctx context.Context, resp *postgresflex.GetInstanceResponse, model *Model, flavor *flavorModel, storage *storageModel, network *networkModel, region string) error {
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

	/*

		var aclList basetypes.ListValue
		var diags diag.Diagnostics
		if instance.Acl == nil {
			aclList = types.ListNull(types.StringType)
		} else {
			respACL := *instance.Acl
			modelACL, err := utils.ListValuetoStringSlice(model.ACL)
			if err != nil {
				return err
			}

			reconciledACL := utils.ReconcileStringSlices(modelACL, respACL)

			aclList, diags = types.ListValueFrom(ctx, types.StringType, reconciledACL)
			if diags.HasError() {
				return fmt.Errorf("mapping ACL: %w", core.DiagsToError(diags))
			}
		}

	*/

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

	var flavorValues map[string]attr.Value
	if instance.FlavorId == nil {
		flavorValues = map[string]attr.Value{
			"id":          flavor.Id,
			"description": flavor.Description,
			"cpu":         flavor.CPU,
			"ram":         flavor.RAM,
		}
	} else {
		// TODO @mhenselin
		// flavorValues = map[string]attr.Value{
		//	"id":          types.StringValue(*instance.FlavorId),
		//	"description": types.StringValue(*instance.FlavorId.Description),
		//	"cpu":         types.Int64PointerValue(instance.FlavorId.Cpu),
		//	"ram":         types.Int64PointerValue(instance.FlavorId.Memory),
		// }
	}
	flavorObject, diags := types.ObjectValue(flavorTypes, flavorValues)
	if diags.HasError() {
		return fmt.Errorf("creating flavor: %w", core.DiagsToError(diags))
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

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), region, instanceId)
	model.InstanceId = types.StringValue(instanceId)
	model.Name = types.StringPointerValue(instance.Name)
	// model.ACL = aclList
	model.Network = networkObject
	model.BackupSchedule = types.StringPointerValue(instance.BackupSchedule)
	model.Flavor = flavorObject
	// TODO - verify working
	model.Replicas = types.Int64Value(int64(*instance.Replicas))
	model.Storage = storageObject
	model.Version = types.StringPointerValue(instance.Version)
	model.Region = types.StringValue(region)
	//model.Encryption = types.ObjectValue()
	//model.Network = networkModel
	return nil
}

func toCreatePayload(model *Model, acl []string, flavor *flavorModel, storage *storageModel, enc *encryptionModel, net *networkModel) (*postgresflex.CreateInstanceRequestPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if acl == nil {
		return nil, fmt.Errorf("nil acl")
	}
	if flavor == nil {
		return nil, fmt.Errorf("nil flavor")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}

	replVal := int32(model.Replicas.ValueInt64())
	return &postgresflex.CreateInstanceRequestPayload{
		// TODO - verify working
		//Acl: &[]string{
		//	strings.Join(acl, ","),
		//},
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       conversion.StringValueToPointer(flavor.Id),
		Name:           conversion.StringValueToPointer(model.Name),
		// TODO - verify working
		Replicas: postgresflex.CreateInstanceRequestPayloadGetReplicasAttributeType(&replVal),
		// TODO - verify working
		Storage: postgresflex.CreateInstanceRequestPayloadGetStorageAttributeType(&postgresflex.Storage{
			PerformanceClass: conversion.StringValueToPointer(storage.Class),
			Size:             conversion.Int64ValueToPointer(storage.Size),
		}),
		Version: conversion.StringValueToPointer(model.Version),
		// TODO - verify working
		Encryption: postgresflex.CreateInstanceRequestPayloadGetEncryptionAttributeType(
			&postgresflex.InstanceEncryption{
				KekKeyId:       conversion.StringValueToPointer(enc.KeyId), // model.Encryption.Attributes(),
				KekKeyRingId:   conversion.StringValueToPointer(enc.KeyRingId),
				KekKeyVersion:  conversion.StringValueToPointer(enc.KeyVersion),
				ServiceAccount: conversion.StringValueToPointer(enc.ServiceAccount),
			},
		),
		Network: &postgresflex.InstanceNetwork{
			AccessScope: postgresflex.InstanceNetworkGetAccessScopeAttributeType(
				conversion.StringValueToPointer(net.AccessScope),
			),
		},
	}, nil
}

func toUpdatePayload(model *Model, acl []string, flavor *flavorModel, storage *storageModel) (*postgresflex.UpdateInstancePartiallyRequestPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if acl == nil {
		return nil, fmt.Errorf("nil acl")
	}
	if flavor == nil {
		return nil, fmt.Errorf("nil flavor")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}

	return &postgresflex.UpdateInstancePartiallyRequestPayload{
		//Acl: postgresflexalpha.UpdateInstancePartiallyRequestPayloadGetAclAttributeType{
		//	Items: &acl,
		//},
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       conversion.StringValueToPointer(flavor.Id),
		Name:           conversion.StringValueToPointer(model.Name),
		//Replicas:       conversion.Int64ValueToPointer(model.Replicas),
		Storage: &postgresflex.StorageUpdate{
			Size: conversion.Int64ValueToPointer(storage.Size),
		},
		Version: conversion.StringValueToPointer(model.Version),
	}, nil
}

func loadFlavorId(ctx context.Context, client postgresflexClient, model *Model, flavor *flavorModel) error {
	if model == nil {
		return fmt.Errorf("nil model")
	}
	if flavor == nil {
		return fmt.Errorf("nil flavor")
	}
	cpu := conversion.Int64ValueToPointer(flavor.CPU)
	if cpu == nil {
		return fmt.Errorf("nil CPU")
	}
	ram := conversion.Int64ValueToPointer(flavor.RAM)
	if ram == nil {
		return fmt.Errorf("nil RAM")
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()
	res, err := client.GetFlavorsRequestExecute(ctx, projectId, region)
	if err != nil {
		return fmt.Errorf("listing postgresflex flavors: %w", err)
	}

	avl := ""
	if res.Flavors == nil {
		return fmt.Errorf("finding flavors for project %s", projectId)
	}
	for _, f := range *res.Flavors {
		if f.Id == nil || f.Cpu == nil || f.Memory == nil {
			continue
		}
		if *f.Cpu == *cpu && *f.Memory == *ram {
			flavor.Id = types.StringValue(*f.Id)
			flavor.Description = types.StringValue(*f.Description)
			break
		}
		avl = fmt.Sprintf("%s\n- %d CPU, %d GB RAM", avl, *f.Cpu, *f.Memory)
	}
	if flavor.Id.ValueString() == "" {
		return fmt.Errorf("couldn't find flavor, available specs are:%s", avl)
	}

	return nil
}
