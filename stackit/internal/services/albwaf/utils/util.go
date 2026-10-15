package utils

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func WarnIfNameChanges(stateName, planName types.String, resourceLabel string, diags *diag.Diagnostics) {
	if utils.IsUndefined(stateName) {
		return
	}
	if planName.Equal(stateName) {
		return
	}
	diags.AddWarning(
		fmt.Sprintf("%s name change requires resource replacement", resourceLabel),
		fmt.Sprintf(
			"Changing the \"name\" attribute from %q to %q will destroy and recreate this resource. "+
				"If another resource references this %s by name, the replacement will fail. Remove or update that dependency before applying this change.",
			stateName.ValueString(), planName.ValueString(), resourceLabel,
		),
	)
}
