package utils

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

// AdaptRegion rewrites the region of a terraform plan
func AdaptRegion(ctx context.Context, configRegion types.String, planRegion *types.String, defaultRegion string, resp *resource.ModifyPlanResponse) {
	// Get the intended region. This is either set directly set in the individual
	// config or the provider region has to be used
	var intendedRegion types.String
	if configRegion.IsNull() {
		if defaultRegion == "" {
			core.LogAndAddError(ctx, &resp.Diagnostics, "set region", "no region defined in config or provider")
			return
		}
		intendedRegion = types.StringValue(defaultRegion)
	} else {
		intendedRegion = configRegion
	}

	// check if the currently configured region corresponds to the planned region
	// on mismatch override the planned region with the intended region
	// and force a replacement of the resource
	p := path.Root("region")
	if !intendedRegion.Equal(*planRegion) {
		resp.RequiresReplace.Append(p)
		*planRegion = intendedRegion
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, p, *planRegion)...)
}
