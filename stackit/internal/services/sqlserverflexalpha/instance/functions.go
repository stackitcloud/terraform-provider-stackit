package sqlserverflex

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	sqlserverflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
)

type sqlserverflexClient interface {
	GetFlavorsRequestExecute(ctx context.Context, projectId, region string, page, size *int64, sort *sqlserverflex.FlavorSort) (*sqlserverflex.GetFlavorsResponse, error)
}

func mapFields(ctx context.Context, resp *sqlserverflex.GetInstanceResponse, model *Model, flavor *flavorModel, storage *storageModel, encryption *encryptionModel, network *networkModel, region string) error {
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

	var flavorValues map[string]attr.Value
	if instance.FlavorId == nil {
		return fmt.Errorf("instance has no flavor id")
	}
	if *instance.FlavorId != flavor.Id.ValueString() {
		return fmt.Errorf("instance has different flavor id %s - %s", *instance.FlavorId, flavor.Id.ValueString())
	}

	flavorValues = map[string]attr.Value{
		"id":          flavor.Id,
		"description": flavor.Description,
		"cpu":         flavor.CPU,
		"ram":         flavor.RAM,
		"node_type":   flavor.NodeType,
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
	model.Flavor = flavorObject
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

	if model.Flavor.IsNull() || model.Flavor.IsUnknown() {
		return nil, fmt.Errorf("nil flavor")
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

	networkPayload := &sqlserverflex.CreateInstanceRequestPayloadGetNetworkArgType{}
	if network != nil {
		networkPayload.AccessScope = sqlserverflex.CreateInstanceRequestPayloadNetworkGetAccessScopeAttributeType(conversion.StringValueToPointer(network.AccessScope))
	}

	flavorId := ""
	if !(model.Flavor.IsNull() || model.Flavor.IsUnknown()) {
		modelValues := model.Flavor.Attributes()
		if _, ok := modelValues["id"]; !ok {
			return nil, fmt.Errorf("flavor has not yet been created")
		}
		flavorId = strings.Trim(modelValues["id"].String(), "\"")
	}

	var aclElements []string
	if network != nil && !(network.ACL.IsNull() || network.ACL.IsUnknown()) {
		aclElements = make([]string, 0, len(network.ACL.Elements()))
		diags := network.ACL.ElementsAs(context.TODO(), &aclElements, false)
		if diags.HasError() {
			return nil, fmt.Errorf("creating network: %w", core.DiagsToError(diags))
		}
	}

	return &sqlserverflex.CreateInstanceRequestPayload{
		Acl:            &aclElements,
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       &flavorId,
		Name:           conversion.StringValueToPointer(model.Name),
		Storage:        storagePayload,
		Version:        sqlserverflex.CreateInstanceRequestPayloadGetVersionAttributeType(conversion.StringValueToPointer(model.Version)),
		Encryption:     encryptionPayload,
		RetentionDays:  conversion.Int64ValueToPointer(model.RetentionDays),
		Network:        networkPayload,
	}, nil
}

func toUpdatePayload(model *Model, storage *storageModel, network *networkModel) (*sqlserverflex.UpdateInstancePartiallyRequestPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if model.Flavor.IsNull() || model.Flavor.IsUnknown() {
		return nil, fmt.Errorf("nil flavor")
	}
	var flavorMdl flavorModel
	diag := model.Flavor.As(context.Background(), &flavorMdl, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: false,
	})
	if diag.HasError() {
		return nil, fmt.Errorf("flavor conversion error: %v", diag.Errors())
	}

	storagePayload := &sqlserverflex.UpdateInstanceRequestPayloadGetStorageArgType{}
	if storage != nil {
		storagePayload.Size = conversion.Int64ValueToPointer(storage.Size)
	}

	var aclElements []string
	if network != nil && !(network.ACL.IsNull() || network.ACL.IsUnknown()) {
		aclElements = make([]string, 0, len(network.ACL.Elements()))
		diags := network.ACL.ElementsAs(context.TODO(), &aclElements, false)
		if diags.HasError() {
			return nil, fmt.Errorf("creating network: %w", core.DiagsToError(diags))
		}
	}

	// TODO - implement network.ACL as soon as it becomes available
	replCount := int32(model.Replicas.ValueInt64())
	flavorId := flavorMdl.Id.ValueString()
	return &sqlserverflex.UpdateInstancePartiallyRequestPayload{
		Acl:            &aclElements,
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       &flavorId,
		Name:           conversion.StringValueToPointer(model.Name),
		Replicas:       sqlserverflex.UpdateInstancePartiallyRequestPayloadGetReplicasAttributeType(&replCount),
		RetentionDays:  conversion.Int64ValueToPointer(model.RetentionDays),
		Storage:        storagePayload,
		Version:        sqlserverflex.UpdateInstancePartiallyRequestPayloadGetVersionAttributeType(conversion.StringValueToPointer(model.Version)),
	}, nil
}

func getAllFlavors(ctx context.Context, client sqlserverflexClient, projectId, region string) ([]sqlserverflex.ListFlavors, error) {
	if projectId == "" || region == "" {
		return nil, fmt.Errorf("listing sqlserverflex flavors: projectId and region are required")
	}
	var flavorList []sqlserverflex.ListFlavors

	page := int64(1)
	size := int64(10)
	for {
		sort := sqlserverflex.FLAVORSORT_INDEX_ASC
		res, err := client.GetFlavorsRequestExecute(ctx, projectId, region, &page, &size, &sort)
		if err != nil {
			return nil, fmt.Errorf("listing sqlserverflex flavors: %w", err)
		}
		if res.Flavors == nil {
			return nil, fmt.Errorf("finding flavors for project %s", projectId)
		}
		pagination := res.GetPagination()
		flavorList = append(flavorList, *res.Flavors...)

		if *pagination.TotalRows == int64(len(flavorList)) {
			break
		}
		page++
	}
	return flavorList, nil
}

func loadFlavorId(ctx context.Context, client sqlserverflexClient, model *Model, flavor *flavorModel, storage *storageModel) error {
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
	nodeType := conversion.StringValueToPointer(flavor.NodeType)
	if nodeType == nil {
		return fmt.Errorf("nil NodeType")
	}
	storageClass := conversion.StringValueToPointer(storage.Class)
	if storageClass == nil {
		return fmt.Errorf("nil StorageClass")
	}
	storageSize := conversion.Int64ValueToPointer(storage.Size)
	if storageSize == nil {
		return fmt.Errorf("nil StorageSize")
	}

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()

	flavorList, err := getAllFlavors(ctx, client, projectId, region)
	if err != nil {
		return err
	}

	avl := ""
	foundFlavorCount := 0
	for _, f := range flavorList {
		if f.Id == nil || f.Cpu == nil || f.Memory == nil {
			continue
		}
		if !strings.EqualFold(*f.NodeType, *nodeType) {
			continue
		}
		if *f.Cpu == *cpu && *f.Memory == *ram {
			var useSc *sqlserverflex.FlavorStorageClassesStorageClass
			for _, sc := range *f.StorageClasses {
				if *sc.Class != *storageClass {
					continue
				}
				if *storageSize < *f.MinGB || *storageSize > *f.MaxGB {
					return fmt.Errorf("storage size %d out of bounds (min: %d - max: %d)", *storageSize, *f.MinGB, *f.MaxGB)
				}
				useSc = &sc
			}
			if useSc == nil {
				return fmt.Errorf("no storage class found for %s", *storageClass)
			}

			flavor.Id = types.StringValue(*f.Id)
			flavor.Description = types.StringValue(*f.Description)
			foundFlavorCount++
		}
		for _, cls := range *f.StorageClasses {
			avl = fmt.Sprintf("%s\n- %d CPU, %d GB RAM, storage %s (min: %d - max: %d)", avl, *f.Cpu, *f.Memory, *cls.Class, *f.MinGB, *f.MaxGB)
		}
	}
	if foundFlavorCount > 1 {
		return fmt.Errorf("multiple flavors found: %d flavors", foundFlavorCount)
	}
	if flavor.Id.ValueString() == "" {
		return fmt.Errorf("couldn't find flavor, available specs are:%s", avl)
	}

	return nil
}

func getFlavorModelById(ctx context.Context, client sqlserverflexClient, model *Model, flavor *flavorModel) error {
	if model == nil {
		return fmt.Errorf("nil model")
	}
	if flavor == nil {
		return fmt.Errorf("nil flavor")
	}
	id := conversion.StringValueToPointer(flavor.Id)
	if id == nil {
		return fmt.Errorf("nil flavor ID")
	}

	flavor.Id = types.StringValue("")

	projectId := model.ProjectId.ValueString()
	region := model.Region.ValueString()

	flavorList, err := getAllFlavors(ctx, client, projectId, region)
	if err != nil {
		return err
	}

	avl := ""
	for _, f := range flavorList {
		if f.Id == nil || f.Cpu == nil || f.Memory == nil {
			continue
		}
		if *f.Id == *id {
			flavor.Id = types.StringValue(*f.Id)
			flavor.Description = types.StringValue(*f.Description)
			flavor.CPU = types.Int64Value(*f.Cpu)
			flavor.RAM = types.Int64Value(*f.Memory)
			flavor.NodeType = types.StringValue(*f.NodeType)
			break
		}
		avl = fmt.Sprintf("%s\n- %d CPU, %d GB RAM", avl, *f.Cpu, *f.Memory)
	}
	if flavor.Id.ValueString() == "" {
		return fmt.Errorf("couldn't find flavor, available specs are: %s", avl)
	}

	return nil
}
