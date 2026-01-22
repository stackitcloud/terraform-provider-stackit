// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: Apache-2.0

package sqlserverflexalpha

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/stackitcloud/stackit-sdk-go/core/config"

	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/utils"

	sqlserverflexalphaGen "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/sqlserverflexalpha/database/resources_gen"
)

var (
	_ resource.Resource                = &databaseResource{}
	_ resource.ResourceWithConfigure   = &databaseResource{}
	_ resource.ResourceWithImportState = &databaseResource{}
	_ resource.ResourceWithModifyPlan  = &databaseResource{}
)

func NewDatabaseResource() resource.Resource {
	return &databaseResource{}
}

type databaseResource struct {
	client       *sqlserverflexalpha.APIClient
	providerData core.ProviderData
}

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sqlserverflexalpha_database"
}

func (r *databaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = sqlserverflexalphaGen.DatabaseResourceSchema(ctx)
}

// Configure adds the provider configured client to the resource.
func (r *databaseResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	var ok bool
	r.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(r.providerData.RoundTripper),
		utils.UserAgentConfigOption(r.providerData.Version),
	}
	if r.providerData.PostgresFlexCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(r.providerData.PostgresFlexCustomEndpoint))
	} else {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithRegion(r.providerData.GetRegion()))
	}
	apiClient, err := sqlserverflexalpha.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error configuring API client",
			fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err),
		)
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "sqlserverflexalpha.Database client configured")
}

func (r *databaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data sqlserverflexalphaGen.DatabaseModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Create API call logic

	// Example data value setting
	// data.DatabaseId = types.StringValue("id-from-response")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "sqlserverflexalpha.Database created")
}

func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data sqlserverflexalphaGen.DatabaseModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Todo: Read API call logic

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "sqlserverflexalpha.Database read")
}

func (r *databaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data sqlserverflexalphaGen.DatabaseModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Todo: Update API call logic

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "sqlserverflexalpha.Database updated")
}

func (r *databaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data sqlserverflexalphaGen.DatabaseModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Todo: Delete API call logic

	tflog.Info(ctx, "sqlserverflexalpha.Database deleted")
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
// Use the modifier to set the effective region in the current plan.
func (r *databaseResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) { // nolint:gocritic // function signature required by Terraform
	var configModel sqlserverflexalphaGen.DatabaseModel
	// skip initial empty configuration to avoid follow-up errors
	if req.Config.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planModel sqlserverflexalphaGen.DatabaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	utils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,zone_id,record_set_id
func (r *databaseResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, core.Separator)

	// Todo: Import logic
	if len(idParts) < 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(
			ctx, &resp.Diagnostics,
			"Error importing database",
			fmt.Sprintf(
				"Expected import identifier with format [project_id],[region],..., got %q",
				req.ID,
			),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[1])...)
	// ... more ...

	core.LogAndAddWarning(
		ctx,
		&resp.Diagnostics,
		"Sqlserverflexalpha database imported with empty password",
		"The database password is not imported as it is only available upon creation of a new database. The password field will be empty.",
	)
	tflog.Info(ctx, "Sqlserverflexalpha database state imported")
}
