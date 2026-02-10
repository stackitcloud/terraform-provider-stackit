package kubernetesversions

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	skeUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/ske/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

// Model types
type Model struct {
	Region             types.String `tfsdk:"region"`
	VersionState       types.String `tfsdk:"version_state"`
	KubernetesVersions types.List   `tfsdk:"kubernetes_versions"`
}

var (
	kubernetesVersionType = map[string]attr.Type{
		"version":         types.StringType,
		"expiration_date": types.StringType,
		"feature_gates":   types.MapType{ElemType: types.StringType},
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
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "SKE options client configured")
}

func (d *kubernetesVersionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Returns Kubernetes versions as reported by the SKE provider options API for the given region."

	resp.Schema = schema.Schema{
		Description: description,
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "Region override. If omitted, the provider’s region will be used.",
			},
			"version_state": schema.StringAttribute{
				Optional:    true,
				Description: "If specified, only returns Kubernetes versions with this version state. " + utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(ske.AllowedGetProviderOptionsRequestVersionStateEnumValues)...),
				Validators: []validator.String{
					stringvalidator.OneOf(sdkUtils.EnumSliceToStringSlice(ske.AllowedGetProviderOptionsRequestVersionStateEnumValues)...),
				},
			},
			"kubernetes_versions": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Kubernetes versions and their metadata.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"version": schema.StringAttribute{
							Computed:    true,
							Description: "Kubernetes version string (e.g., `1.33.6`).",
						},
						"expiration_date": schema.StringAttribute{
							Computed:    true,
							Description: "Expiration date of the version in RFC3339 format.",
						},
						"state": schema.StringAttribute{
							Computed:    true,
							Description: "State of the kubernetes version.",
						},
						"feature_gates": schema.MapAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Map of available feature gates for this version.",
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

	ctx = core.InitProviderContext(ctx)
	ctx = tflog.SetField(ctx, "region", region)

	listProviderOptionsReq := d.client.ListProviderOptions(ctx, region)

	if !utils.IsUndefined(model.VersionState) {
		listProviderOptionsReq = listProviderOptionsReq.VersionState(model.VersionState.ValueString())
	}

	optionsResp, err := listProviderOptionsReq.Execute()
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
		return
	}

	ctx = core.LogResponse(ctx)

	if err := mapFields(ctx, optionsResp, &model); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading provider options", fmt.Sprintf("Mapping API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Read SKE provider options successfully", map[string]any{
		"region":       region,
		"versionState": model.VersionState.ValueString(),
	})
}

func mapFields(_ context.Context, optionsResp *ske.ProviderOptions, model *Model) error {
	if optionsResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if optionsResp.KubernetesVersions == nil {
		emptyList, diags := types.ListValue(
			types.ObjectType{AttrTypes: kubernetesVersionType},
			[]attr.Value{},
		)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}
		model.KubernetesVersions = emptyList
		return nil
	}

	kvSlice := *optionsResp.KubernetesVersions
	kvList := make([]attr.Value, 0, len(kvSlice))

	for _, kv := range kvSlice {
		expDate := types.StringNull()
		if kv.ExpirationDate != nil {
			expDate = types.StringValue(kv.ExpirationDate.Format(time.RFC3339))
		}

		featureGateValues := map[string]attr.Value{}
		if kv.FeatureGates != nil {
			for k, v := range *kv.FeatureGates {
				featureGateValues[k] = types.StringValue(v)
			}
		}

		featureGatesMap, diags := types.MapValue(types.StringType, featureGateValues)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}

		obj, diags := types.ObjectValue(
			kubernetesVersionType,
			map[string]attr.Value{
				"version":         types.StringPointerValue(kv.Version),
				"state":           types.StringPointerValue(kv.State),
				"expiration_date": expDate,
				"feature_gates":   featureGatesMap,
			},
		)
		if diags.HasError() {
			return core.DiagsToError(diags)
		}

		kvList = append(kvList, obj)
	}

	kvs, diags := types.ListValue(
		types.ObjectType{AttrTypes: kubernetesVersionType},
		kvList,
	)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}
	model.KubernetesVersions = kvs

	return nil
}
