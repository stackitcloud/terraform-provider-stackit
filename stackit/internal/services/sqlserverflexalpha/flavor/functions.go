// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: Apache-2.0

package sqlserverFlexAlphaFlavor

import (
	"context"
	"fmt"

	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha"
)

type flavorsClientReader interface {
	GetFlavorsRequest(
		ctx context.Context,
		projectId, region string,
	) sqlserverflexalpha.ApiGetFlavorsRequestRequest
}

func getAllFlavors(ctx context.Context, client flavorsClientReader, projectId, region string) (
	[]sqlserverflexalpha.ListFlavors,
	error,
) {
	getAllFilter := func(_ sqlserverflexalpha.ListFlavors) bool { return true }
	flavorList, err := getFlavorsByFilter(ctx, client, projectId, region, getAllFilter)
	if err != nil {
		return nil, err
	}
	return flavorList, nil
}

// getFlavorsByFilter is a helper function to retrieve flavors using a filtern function.
// Hint: The API does not have a GetFlavors endpoint, only ListFlavors
func getFlavorsByFilter(
	ctx context.Context,
	client flavorsClientReader,
	projectId, region string,
	filter func(db sqlserverflexalpha.ListFlavors) bool,
) ([]sqlserverflexalpha.ListFlavors, error) {
	if projectId == "" || region == "" {
		return nil, fmt.Errorf("listing sqlserverflexalpha flavors: projectId and region are required")
	}

	const pageSize = 25

	var result = make([]sqlserverflexalpha.ListFlavors, 0)

	for page := int64(1); ; page++ {
		res, err := client.GetFlavorsRequest(ctx, projectId, region).
			Page(page).Size(pageSize).Sort(sqlserverflexalpha.FLAVORSORT_INDEX_ASC).Execute()
		if err != nil {
			return nil, fmt.Errorf("requesting flavors list (page %d): %w", page, err)
		}

		// If the API returns no flavors, we have reached the end of the list.
		if res.Flavors == nil || len(*res.Flavors) == 0 {
			break
		}

		for _, flavor := range *res.Flavors {
			if filter(flavor) {
				result = append(result, flavor)
			}
		}
	}

	return result, nil
}
