package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
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
