package link

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	telemetrylink "github.com/stackitcloud/stackit-sdk-go/services/telemetrylink/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/telemetrylink/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

var (
	_ datasource.DataSource = &telemetryLinkDataSource{}
)

func NewTelemetryLinkDataSource() datasource.DataSource {
	return &telemetryLinkDataSource{}
}

type DataSourceModel struct {
	ID                types.String `tfsdk:"id"` // Required by Terraform
	LinkID            types.String `tfsdk:"link_id"`
	Region            types.String `tfsdk:"region"`
	ResourceType      types.String `tfsdk:"resource_type"`
	ResourceID        types.String `tfsdk:"resource_id"`
	DisplayName       types.String `tfsdk:"display_name"`
	Description       types.String `tfsdk:"description"`
	TelemetryRouterID types.String `tfsdk:"telemetry_router_id"`
	CreateTime        types.String `tfsdk:"create_time"`
	Status            types.String `tfsdk:"status"`
}

type telemetryLinkDataSource struct {
	client       *telemetrylink.APIClient
	providerData core.ProviderData
}

func (d *telemetryLinkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_telemetrylink"
}

func (d *telemetryLinkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.providerData = providerData

	apiClient := utils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "TelemetryLink client configured")
}

func (d *telemetryLinkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("TelemetryLink data source schema. %s", core.DatasourceRegionFallbackDocstring),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
			},
			"link_id": schema.StringAttribute{
				Description: schemaDescriptions["instance_id"],
				Computed:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"resource_type": schema.StringAttribute{
				Description: schemaDescriptions["resource_type"],
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(resourceTypes...),
					validate.NoSeparator(),
				},
			},
			"resource_id": schema.StringAttribute{
				Description: schemaDescriptions["resource_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: schemaDescriptions["region"],
				// the region cannot be found, so it has to be passed
				Optional: true,
			},
			"display_name": schema.StringAttribute{
				Description: schemaDescriptions["display_name"],
				Computed:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				Description: schemaDescriptions["description"],
				Computed:    true,
			},
			"telemetry_router_id": schema.StringAttribute{
				Description: schemaDescriptions["telemetry_router_id"],
				Computed:    true,
			},
			"create_time": schema.StringAttribute{
				Description: schemaDescriptions["create_time"],
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: schemaDescriptions["status"],
				Computed:    true,
			},
		},
	}
}

func (d *telemetryLinkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model DataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	resourceType := model.ResourceType.ValueString()
	resourceID := model.ResourceID.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "resource_type", resourceType)
	ctx = tflog.SetField(ctx, "resource_id", resourceID)
	ctx = tflog.SetField(ctx, "region", region)

	var response *telemetrylink.TelemetryLinkResponse
	var err error
	switch resourceType {
	case resourceTypeOrganization:
		response, err = d.client.DefaultAPI.GetOrganizationTelemetryLink(ctx, resourceID, region).Execute()
	case resourceTypeFolder:
		response, err = d.client.DefaultAPI.GetFolderTelemetryLink(ctx, resourceID, region).Execute()
	case resourceTypeProject:
		response, err = d.client.DefaultAPI.GetProjectTelemetryLink(ctx, resourceID, region).Execute()
	}
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		ok := errors.As(err, &oapiErr)
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		tfutils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading TelemetryLink",
			fmt.Sprintf("TelemetryLink for resource type %q and resource ID %q does not exist.", resourceType, resourceID),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Resource with type %q ID %q not found or forbidden access", resourceType, resourceID),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}
	ctx = core.LogResponse(ctx)

	err = mapDataSourceFields(ctx, response, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading TelemetryLink", fmt.Sprintf("Processing response: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "TelemetryLink read", map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})
}

func mapDataSourceFields(_ context.Context, link *telemetrylink.TelemetryLinkResponse, model *DataSourceModel) error {
	if link == nil {
		return fmt.Errorf("link is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}
	var linkID string
	if model.LinkID.ValueString() != "" {
		linkID = model.LinkID.ValueString()
	} else {
		linkID = link.Id
	}

	model.ID = tfutils.BuildInternalTerraformId(model.ResourceType.ValueString(), model.ResourceID.ValueString(), model.Region.ValueString())
	model.LinkID = types.StringValue(linkID)
	model.DisplayName = types.StringValue(link.DisplayName)
	model.Description = types.StringPointerValue(link.Description)
	model.TelemetryRouterID = types.StringValue(link.TelemetryRouterId)
	model.CreateTime = types.StringValue(link.CreateTime.String())
	model.Status = types.StringValue(link.Status)

	return nil
}
