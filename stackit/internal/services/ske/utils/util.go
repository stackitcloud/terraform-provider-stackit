package utils

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ske "github.com/stackitcloud/stackit-sdk-go/services/ske/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils/planmodifiers/stringplanmodifier"
)

func IsEmptyNetwork(network *ske.Network) bool {
	if !network.HasId() && !network.HasControlPlane() {
		return true
	}
	return false
}

func IsEmptyExtension(extension *ske.Extension) bool {
	if !extension.HasDns() && !extension.HasAcl() && !extension.HasObservability() {
		return true
	}
	return false
}

// Deprecated: HasOsVersionMinChanged
func HasOsVersionMinChanged(ctx context.Context, request planmodifier.StringRequest, response *stringplanmodifier.UseStateForUnknownFuncResponse) { // nolint:gocritic // function signature required by Terraform
	dependencyPath := request.Path.ParentPath().AtName("os_version_min")

	var minVersionPlan types.String
	diags := request.Plan.GetAttribute(ctx, dependencyPath, &minVersionPlan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	var minVersionState types.String
	diags = request.State.GetAttribute(ctx, dependencyPath, &minVersionState)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if minVersionState == minVersionPlan {
		response.UseStateForUnknown = true
		return
	}
}
