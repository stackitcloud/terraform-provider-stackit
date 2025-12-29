package postgresflexalpha

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"
)

type postgresflexClient interface {
	GetFlavorsRequestExecute(ctx context.Context, projectId string, region string, page, size *int64, sort *postgresflex.FlavorSort) (*postgresflex.GetFlavorsResponse, error)
}

func mapFields(
	ctx context.Context,
	resp *postgresflex.GetInstanceResponse,
	model *Model,
	flavor *flavorModel,
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
		return fmt.Errorf("instance has no flavor id")
	}
	if !flavor.Id.IsUnknown() && !flavor.Id.IsNull() {
		if *instance.FlavorId != flavor.Id.ValueString() {
			return fmt.Errorf("instance has different flavor id %s - %s", *instance.FlavorId, flavor.Id.ValueString())
		}
	}
	if model.Flavor.IsNull() || model.Flavor.IsUnknown() {
		flavorValues = map[string]attr.Value{
			"id":          flavor.Id,
			"description": flavor.Description,
			"cpu":         flavor.CPU,
			"ram":         flavor.RAM,
			"node_type":   flavor.NodeType,
		}
	} else {
		flavorValues = model.Flavor.Attributes()
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
	model.Network = networkObject
	model.BackupSchedule = types.StringPointerValue(instance.BackupSchedule)
	model.Flavor = flavorObject
	// TODO - verify working
	model.Replicas = types.Int64Value(int64(*instance.Replicas))
	model.Storage = storageObject
	model.Version = types.StringPointerValue(instance.Version)
	model.Region = types.StringValue(region)
	model.Encryption = encryptionObject
	model.Network = networkObject
	return nil
}

func toCreatePayload(model *Model, flavor *flavorModel, storage *storageModel, enc *encryptionModel, net *networkModel) (*postgresflex.CreateInstanceRequestPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if flavor == nil {
		return nil, fmt.Errorf("nil flavor")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}

	if model.Replicas.ValueInt64() > math.MaxInt32 {
		return nil, fmt.Errorf("replica count too big: %d", model.Replicas.ValueInt64())
	}
	replVal := int32(model.Replicas.ValueInt64()) // nolint:gosec // check is performed above

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

	networkPayload := &postgresflex.CreateInstanceRequestPayloadGetNetworkArgType{}
	if net != nil {
		networkPayload = &postgresflex.CreateInstanceRequestPayloadGetNetworkArgType{
			AccessScope: postgresflex.InstanceNetworkGetAccessScopeAttributeType(conversion.StringValueToPointer(net.AccessScope)),
			Acl:         &aclElements,
		}
	}

	return &postgresflex.CreateInstanceRequestPayload{
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       conversion.StringValueToPointer(flavor.Id),
		Name:           conversion.StringValueToPointer(model.Name),
		// TODO - verify working
		Replicas:   postgresflex.CreateInstanceRequestPayloadGetReplicasAttributeType(&replVal),
		Storage:    storagePayload,
		Version:    conversion.StringValueToPointer(model.Version),
		Encryption: encryptionPayload,
		Network:    networkPayload,
	}, nil
}

func toUpdatePayload(model *Model, flavor *flavorModel, storage *storageModel, _ *networkModel) (*postgresflex.UpdateInstancePartiallyRequestPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	if flavor == nil {
		return nil, fmt.Errorf("nil flavor")
	}
	if storage == nil {
		return nil, fmt.Errorf("nil storage")
	}

	return &postgresflex.UpdateInstancePartiallyRequestPayload{
		// Acl: postgresflexalpha.UpdateInstancePartiallyRequestPayloadGetAclAttributeType{
		//	Items: &acl,
		// },
		BackupSchedule: conversion.StringValueToPointer(model.BackupSchedule),
		FlavorId:       conversion.StringValueToPointer(flavor.Id),
		Name:           conversion.StringValueToPointer(model.Name),
		// Replicas:       conversion.Int64ValueToPointer(model.Replicas),
		Storage: &postgresflex.StorageUpdate{
			Size: conversion.Int64ValueToPointer(storage.Size),
		},
		Version: conversion.StringValueToPointer(model.Version),
	}, nil
}

func loadFlavorId(ctx context.Context, client postgresflexClient, model *Model, flavor *flavorModel, storage *storageModel) error {
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
			var useSc *postgresflex.FlavorStorageClassesStorageClass
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

func getAllFlavors(ctx context.Context, client postgresflexClient, projectId, region string) ([]postgresflex.ListFlavors, error) {
	if projectId == "" || region == "" {
		return nil, fmt.Errorf("listing postgresflex flavors: projectId and region are required")
	}
	var flavorList []postgresflex.ListFlavors

	page := int64(1)
	size := int64(10)
	for {
		sort := postgresflex.FLAVORSORT_INDEX_ASC
		res, err := client.GetFlavorsRequestExecute(ctx, projectId, region, &page, &size, &sort)
		if err != nil {
			return nil, fmt.Errorf("listing postgresflex flavors: %w", err)
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

func getFlavorModelById(ctx context.Context, client postgresflexClient, model *Model, flavor *flavorModel) error {
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
