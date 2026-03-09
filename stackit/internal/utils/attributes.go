package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils/planmodifiers/int64planmodifier"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils/planmodifiers/stringplanmodifier"
)

type attributeGetter interface {
	GetAttribute(ctx context.Context, attributePath path.Path, target interface{}) diag.Diagnostics
}

func ToTime(ctx context.Context, format string, val types.String, target *time.Time) (diags diag.Diagnostics) {
	var err error
	text := val.ValueString()
	*target, err = time.Parse(format, text)
	if err != nil {
		core.LogAndAddError(ctx, &diags, "cannot parse date", fmt.Sprintf("cannot parse date %q with format %q: %v", text, format, err))
		return diags
	}
	return diags
}

// GetTimeFromStringAttribute retrieves a string attribute from e.g. a [plan.Plan], [tfsdk.Config] or a [tfsdk.State] and
// converts it to a [time.Time] object with a given format, if possible.
func GetTimeFromStringAttribute(ctx context.Context, attributePath path.Path, source attributeGetter, dateFormat string, target *time.Time) (diags diag.Diagnostics) {
	var date types.String
	diags.Append(source.GetAttribute(ctx, attributePath, &date)...)
	if diags.HasError() {
		return diags
	}
	if date.IsNull() || date.IsUnknown() {
		return diags
	}
	diags.Append(ToTime(ctx, dateFormat, date, target)...)
	if diags.HasError() {
		return diags
	}

	return diags
}

// Int64Changed sets UseStateForUnkown to true if the attribute's planned value matches the current state
func Int64Changed(ctx context.Context, attributeName string, request planmodifier.Int64Request, response *int64planmodifier.UseStateForUnknownFuncResponse) { // nolint:gocritic // function signature required by Terraform
	dependencyPath := request.Path.ParentPath().AtName(attributeName)

	var attributePlan types.Int64
	diags := request.Plan.GetAttribute(ctx, dependencyPath, &attributePlan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	var attributeState types.Int64
	diags = request.State.GetAttribute(ctx, dependencyPath, &attributeState)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if attributeState == attributePlan {
		response.UseStateForUnknown = true
		return
	}
}

// StringChanged sets UseStateForUnkown to true if the attribute's planned value matches the current state
func StringChanged(ctx context.Context, attributeName string, request planmodifier.StringRequest, response *stringplanmodifier.UseStateForUnknownFuncResponse) { // nolint:gocritic // function signature required by Terraform
	dependencyPath := request.Path.ParentPath().AtName(attributeName)

	var attributePlan types.String
	diags := request.Plan.GetAttribute(ctx, dependencyPath, &attributePlan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	var attributeState types.String
	diags = request.State.GetAttribute(ctx, dependencyPath, &attributeState)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if attributeState == attributePlan {
		response.UseStateForUnknown = true
		return
	}
}
