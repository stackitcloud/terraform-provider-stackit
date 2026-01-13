package postgresFlexAlphaFlavor

import (
	"context"
	"fmt"

	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
)

type flavorsClient interface {
	GetFlavorsRequestExecute(
		ctx context.Context,
		projectId, region string,
		page, size *int64,
		sort *postgresflex.FlavorSort,
	) (*postgresflex.GetFlavorsResponse, error)
}

//func loadFlavorId(ctx context.Context, client flavorsClient, model *Model, flavor *flavorModel, storage *storageModel) error {
//	if model == nil {
//		return fmt.Errorf("nil model")
//	}
//	if flavor == nil {
//		return fmt.Errorf("nil flavor")
//	}
//	cpu := flavor.CPU.ValueInt64()
//	if cpu == 0 {
//		return fmt.Errorf("nil CPU")
//	}
//	ram := flavor.RAM.ValueInt64()
//	if ram == 0 {
//		return fmt.Errorf("nil RAM")
//	}
//
//	nodeType := flavor.NodeType.ValueString()
//	if nodeType == "" {
//		if model.Replicas.IsNull() || model.Replicas.IsUnknown() {
//			return fmt.Errorf("nil NodeType")
//		}
//		switch model.Replicas.ValueInt64() {
//		case 1:
//			nodeType = "Single"
//		case 3:
//			nodeType = "Replica"
//		default:
//			return fmt.Errorf("unknown Replicas value: %d", model.Replicas.ValueInt64())
//		}
//	}
//
//	storageClass := conversion.StringValueToPointer(storage.Class)
//	if storageClass == nil {
//		return fmt.Errorf("nil StorageClass")
//	}
//	storageSize := conversion.Int64ValueToPointer(storage.Size)
//	if storageSize == nil {
//		return fmt.Errorf("nil StorageSize")
//	}
//
//	projectId := model.ProjectId.ValueString()
//	region := model.Region.ValueString()
//
//	flavorList, err := getAllFlavors(ctx, client, projectId, region)
//	if err != nil {
//		return err
//	}
//
//	avl := ""
//	foundFlavorCount := 0
//	var foundFlavors []string
//	for _, f := range flavorList {
//		if f.Id == nil || f.Cpu == nil || f.Memory == nil {
//			continue
//		}
//		if !strings.EqualFold(*f.NodeType, nodeType) {
//			continue
//		}
//		if *f.Cpu == cpu && *f.Memory == ram {
//			var useSc *postgresflex.FlavorStorageClassesStorageClass
//			for _, sc := range *f.StorageClasses {
//				if *sc.Class != *storageClass {
//					continue
//				}
//				if *storageSize < *f.MinGB || *storageSize > *f.MaxGB {
//					return fmt.Errorf("storage size %d out of bounds (min: %d - max: %d)", *storageSize, *f.MinGB, *f.MaxGB)
//				}
//				useSc = &sc
//			}
//			if useSc == nil {
//				return fmt.Errorf("no storage class found for %s", *storageClass)
//			}
//
//			flavor.Id = types.StringValue(*f.Id)
//			flavor.Description = types.StringValue(*f.Description)
//			foundFlavors = append(foundFlavors, fmt.Sprintf("%s (%d/%d - %s)", *f.Id, *f.Cpu, *f.Memory, *f.NodeType))
//			foundFlavorCount++
//		}
//		for _, cls := range *f.StorageClasses {
//			avl = fmt.Sprintf("%s\n- %d CPU, %d GB RAM, storage %s (min: %d - max: %d)", avl, *f.Cpu, *f.Memory, *cls.Class, *f.MinGB, *f.MaxGB)
//		}
//	}
//	if foundFlavorCount > 1 {
//		return fmt.Errorf(
//			"number of flavors returned: %d\nmultiple flavors found: %d flavors\n  %s",
//			len(flavorList),
//			foundFlavorCount,
//			strings.Join(foundFlavors, "\n  "),
//		)
//	}
//	if flavor.Id.ValueString() == "" {
//		return fmt.Errorf("couldn't find flavor, available specs are:%s", avl)
//	}
//
//	return nil
//}

func getAllFlavors(ctx context.Context, client flavorsClient, projectId, region string) ([]postgresflex.ListFlavors, error) {
	if projectId == "" || region == "" {
		return nil, fmt.Errorf("listing postgresflex flavors: projectId and region are required")
	}
	var flavorList []postgresflex.ListFlavors

	page := int64(1)
	size := int64(10)
	sort := postgresflex.FLAVORSORT_INDEX_ASC
	counter := 0
	for {
		res, err := client.GetFlavorsRequestExecute(ctx, projectId, region, &page, &size, &sort)
		if err != nil {
			return nil, fmt.Errorf("listing postgresflex flavors: %w", err)
		}
		if res.Flavors == nil {
			return nil, fmt.Errorf("finding flavors for project %s", projectId)
		}
		pagination := res.GetPagination()
		flavors := res.GetFlavors()
		flavorList = append(flavorList, flavors...)

		if *pagination.TotalRows < int64(len(flavorList)) {
			return nil, fmt.Errorf("total rows is smaller than current accumulated list - that should not happen")
		}
		if *pagination.TotalRows == int64(len(flavorList)) {
			break
		}
		page++

		if page > *pagination.TotalPages {
			break
		}

		// implement a breakpoint
		counter++
		if counter > 1000 {
			panic("too many pagination results")
		}
	}
	return flavorList, nil
}

//func getFlavorModelById(ctx context.Context, client flavorsClient, model *Model, flavor *flavorModel) error {
//	if model == nil {
//		return fmt.Errorf("nil model")
//	}
//	if flavor == nil {
//		return fmt.Errorf("nil flavor")
//	}
//	id := conversion.StringValueToPointer(flavor.Id)
//	if id == nil {
//		return fmt.Errorf("nil flavor ID")
//	}
//
//	flavor.Id = types.StringValue("")
//
//	projectId := model.ProjectId.ValueString()
//	region := model.Region.ValueString()
//
//	flavorList, err := getAllFlavors(ctx, client, projectId, region)
//	if err != nil {
//		return err
//	}
//
//	avl := ""
//	for _, f := range flavorList {
//		if f.Id == nil || f.Cpu == nil || f.Memory == nil {
//			continue
//		}
//		if *f.Id == *id {
//			flavor.Id = types.StringValue(*f.Id)
//			flavor.Description = types.StringValue(*f.Description)
//			flavor.CPU = types.Int64Value(*f.Cpu)
//			flavor.RAM = types.Int64Value(*f.Memory)
//			flavor.NodeType = types.StringValue(*f.NodeType)
//			break
//		}
//		avl = fmt.Sprintf("%s\n- %d CPU, %d GB RAM", avl, *f.Cpu, *f.Memory)
//	}
//	if flavor.Id.ValueString() == "" {
//		return fmt.Errorf("couldn't find flavor, available specs are: %s", avl)
//	}
//
//	return nil
//}
