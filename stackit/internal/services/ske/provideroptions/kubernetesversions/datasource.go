package kubernetesversions

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	Region             types.String `tfsdk:"region"`
	KubernetesVersions types.List   `tfsdk:"kubernetes_versions"`
}

var (
	kubernetesVersionType = map[string]attr.Type{
		"version":         types.StringType,
		"expiration_date": types.StringType,
		"state":           types.StringType,
	}
)

// Ensure implementation satisfies interface
var _ datasource.DataSource = &kubernetesVersionsDataSource{}

// NewKubernetesVersionsDataSource creates the data source instance
func NewKubernetesVersionsDataSource() datasource.DataSource {
	return &kubernetesVersionsDataSource{}
}

type kubernetesVersionsDataSource struct {
	client       *ske.APIClient
	providerData core.ProviderData
}

// Metadata sets the data source type name
func (d *kubernetesVersionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ske_kubernetes_versions"
}

func (d *kubernetesVersionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = skeUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	tflog.Info(ctx, "SKE options client configured")
}

func (d *kubernetesVersionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Returns a list of supported Kubernetes versions for the cluster nodes."

	resp.Schema = schema.Schema{
		Description: description,
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "Region override. If omitted, the provider’s region will be used.",
			},
			"kubernetes_versions": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Supported Kubernetes versions.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"version": schema.StringAttribute{
							Computed:    true,
							Description: "Kubernetes version string (e.g., `1.33`).",
						},
						"expiration_date": schema.StringAttribute{
							Computed:    true,
							Description: "Expiration date of the version in RFC3339 format.",
						},
						"state": schema.StringAttribute{
							Computed:    true,
							Description: "Version state, such as `supported`, `preview`, or `deprecated`.",
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *kubernetesVersionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
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

func mapFields(_ context.Context, optionsResp *ske.ProviderOptions, model *Model) error {
	if optionsResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	// Kubernetes Versions
	kvList := make([]attr.Value, 0)
	if optionsResp.KubernetesVersions != nil {
		for _, kv := range *optionsResp.KubernetesVersions {
			expDate := types.StringNull()
			if kv.ExpirationDate != nil {
				expDate = types.StringValue(kv.ExpirationDate.Format(time.RFC3339))
			}

			obj, diags := types.ObjectValue(kubernetesVersionType, map[string]attr.Value{
				"version":         types.StringPointerValue(kv.Version),
				"state":           types.StringPointerValue(kv.State),
				"expiration_date": expDate,
			})
			if diags.HasError() {
				return core.DiagsToError(diags)
			}
			kvList = append(kvList, obj)
		}
	}
	kvs, diags := types.ListValue(types.ObjectType{AttrTypes: kubernetesVersionType}, kvList)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.KubernetesVersions = kvs

	return nil
}
