package availabilityzones

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	skeUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// Model types for nested structures
type Model struct {
	Region            types.String `tfsdk:"region"`
	AvailabilityZones types.List   `tfsdk:"availability_zones"`
}

// Ensure implementation satisfies interface
var _ datasource.DataSource = &optionsDataSource{}

// NewKubernetesAvailabilityZonesDataSource creates the data source instance
func NewKubernetesAvailabilityZonesDataSource() datasource.DataSource {
	return &optionsDataSource{}
}

type optionsDataSource struct {
	client       *ske.APIClient
	providerData core.ProviderData
}

// Metadata sets the data source type name
func (d *optionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_availability_zones"
}

func (d *optionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = skeUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	tflog.Info(ctx, "SKE options client configured")
}

func (d *optionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Returns a list of supported Kubernetes Availability Zones for the region."

	resp.Schema = schema.Schema{
		Description: description,
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "Region override. If omitted, the provider’s region will be used.",
			},
			"availability_zones": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of availability zones in the selected region.",
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *optionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	region := d.providerData.GetRegionWithOverride(model.Region)
	ctx = tflog.SetField(ctx, "region", region)

	optionsResp, err := d.client.ListProviderOptions(ctx, region).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading SKE provider options failed",
			"Unable to read SKE provider options",
			map[int]string{
				http.StatusForbidden: "Forbidden access",
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	err = mapFields(ctx, optionsResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &diags, "Error reading provider options", fmt.Sprintf("Mapping API Payload: %v", err))
		return
	}

	// Set final state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "Read SKE provider options successfully")
}

func mapFields(ctx context.Context, optionsResp *ske.ProviderOptions, model *Model) error {
	if optionsResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	// Availability Zones
	azList := make([]types.String, 0)
	if optionsResp.AvailabilityZones != nil {
		for _, az := range *optionsResp.AvailabilityZones {
			if az.Name != nil {
				azList = append(azList, types.StringValue(*az.Name))
			}
		}
	}
	avZones, diags := types.ListValueFrom(ctx, types.StringType, azList)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.AvailabilityZones = avZones

	return nil
}
