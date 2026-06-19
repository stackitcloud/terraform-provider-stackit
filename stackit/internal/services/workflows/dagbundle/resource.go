package dagbundle

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	workflows "github.com/stackitcloud/stackit-sdk-go/services/workflows/v1alphaapi"
	"github.com/stackitcloud/stackit-sdk-go/services/workflows/v1alphaapi/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	workflowsUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/workflows/utils"
	tfutils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

const (
	gitAuthTypeBasic    = "basic"
	gitAuthTypeNone     = "none"
	s3AuthTypeAccessKey = "access_key"
	s3AuthTypeNone      = "none"
)

var bundleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

var (
	_ resource.Resource                   = &dagBundleResource{}
	_ resource.ResourceWithConfigure      = &dagBundleResource{}
	_ resource.ResourceWithImportState    = &dagBundleResource{}
	_ resource.ResourceWithModifyPlan     = &dagBundleResource{}
	_ resource.ResourceWithValidateConfig = &dagBundleResource{}
)

var schemaDescriptions = map[string]string{
	"id":                        "Terraform's internal resource identifier. It is structured as \"`project_id`,`region`,`instance_id`,`name`\".",
	"name":                      "Bundle name. Must be a DNS label: lowercase alphanumeric and hyphens, starting with a letter, max 63 characters. Immutable.",
	"region":                    "STACKIT region name the resource is located in. If not defined, the provider region is used.",
	"project_id":                "STACKIT project ID associated with the Workflows instance.",
	"instance_id":               "ID of the Workflows instance this bundle belongs to.",
	"git":                       "Git-backed DAG bundle source. Exactly one of `git` or `s3` must be set.",
	"s3":                        "S3-backed DAG bundle source. Exactly one of `git` or `s3` must be set.",
	"git.url":                   "Git repository URL.",
	"git.branch":                "Branch, tag, or ref to track.",
	"git.subdir":                "Optional subdirectory inside the Git repository that contains the DAGs. Leading/trailing slashes are stripped by both the server and provider.",
	"git.refresh_interval":      "How often (in seconds) the bundle is re-scanned for changes.",
	"git.auth":                  "Authentication for the Git source.",
	"git.auth.type":             "Authentication scheme: `basic` (username + password) or `none` (public repos).",
	"git.auth.username":         "Git username. Required when `git.auth.type = basic`.",
	"git.auth.password":         "Git password or personal access token. Required when `git.auth.type = basic`. Sensitive. The API never returns this value back.",
	"s3.bucket_name":            "S3 bucket name containing the DAGs.",
	"s3.endpoint":               "S3-compatible endpoint URL. Defaults to STACKIT Object Storage in the region.",
	"s3.prefix":                 "Optional key prefix inside the bucket.",
	"s3.refresh_interval":       "How often (in seconds) the bundle is re-scanned for changes.",
	"s3.auth":                   "Authentication for the S3 source.",
	"s3.auth.type":              "Authentication scheme: `access_key` or `none` (public buckets).",
	"s3.auth.access_key_id":     "S3 access key ID. Required when `s3.auth.type = access_key`.",
	"s3.auth.secret_access_key": "S3 secret access key. Required when `s3.auth.type = access_key`. Sensitive. The API never returns this value back.",
}

type Model struct {
	ID         types.String `tfsdk:"id"`
	ProjectID  types.String `tfsdk:"project_id"`
	Region     types.String `tfsdk:"region"`
	InstanceID types.String `tfsdk:"instance_id"`
	Name       types.String `tfsdk:"name"`
	Git        types.Object `tfsdk:"git"`
	S3         types.Object `tfsdk:"s3"`
}

type gitModel struct {
	URL             types.String `tfsdk:"url"`
	Branch          types.String `tfsdk:"branch"`
	Subdir          types.String `tfsdk:"subdir"`
	RefreshInterval types.Int32  `tfsdk:"refresh_interval"`
	Auth            types.Object `tfsdk:"auth"`
}

type gitAuthModel struct {
	Type     types.String `tfsdk:"type"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

type s3Model struct {
	BucketName      types.String `tfsdk:"bucket_name"`
	Endpoint        types.String `tfsdk:"endpoint"`
	Prefix          types.String `tfsdk:"prefix"`
	RefreshInterval types.Int32  `tfsdk:"refresh_interval"`
	Auth            types.Object `tfsdk:"auth"`
}

type s3AuthModel struct {
	Type            types.String `tfsdk:"type"`
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
}

var gitAuthTypes = map[string]attr.Type{
	"type":     basetypes.StringType{},
	"username": basetypes.StringType{},
	"password": basetypes.StringType{},
}

var s3AuthTypes = map[string]attr.Type{
	"type":              basetypes.StringType{},
	"access_key_id":     basetypes.StringType{},
	"secret_access_key": basetypes.StringType{},
}

var gitTypes = map[string]attr.Type{
	"url":              basetypes.StringType{},
	"branch":           basetypes.StringType{},
	"subdir":           basetypes.StringType{},
	"refresh_interval": basetypes.Int32Type{},
	"auth":             basetypes.ObjectType{AttrTypes: gitAuthTypes},
}

var s3Types = map[string]attr.Type{
	"bucket_name":      basetypes.StringType{},
	"endpoint":         basetypes.StringType{},
	"prefix":           basetypes.StringType{},
	"refresh_interval": basetypes.Int32Type{},
	"auth":             basetypes.ObjectType{AttrTypes: s3AuthTypes},
}

type dagBundleResource struct {
	client       *workflows.APIClient
	providerData core.ProviderData
}

func NewWorkflowsDagBundleResource() resource.Resource {
	return &dagBundleResource{}
}

func (r *dagBundleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflows_dag_bundle"
}

func (r *dagBundleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.providerData = providerData

	features.CheckExperimentEnabled(ctx, &r.providerData, features.WorkflowsExperiment, "stackit_workflows_dag_bundle", core.Resource, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := workflowsUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
}

func (r *dagBundleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) { //nolint:gocritic // function signature required by Terraform
	if req.Config.Raw.IsNull() {
		return
	}
	var configModel Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &configModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var planModel Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tfutils.AdaptRegion(ctx, configModel.Region, &planModel.Region, r.providerData.GetRegion(), resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, planModel)...)
}

func (r *dagBundleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateBundleConfig(ctx, &model, &resp.Diagnostics)
}

func validateBundleConfig(ctx context.Context, model *Model, diags *diag.Diagnostics) {
	gitSet := isSetObj(model.Git)
	s3Set := isSetObj(model.S3)
	gitUnset := model.Git.IsNull()
	s3Unset := model.S3.IsNull()

	// Exactly one source block must be set. Unknown values defer (the variant
	// will be known by apply time).
	switch {
	case gitUnset && s3Unset:
		diags.AddError(
			"Invalid Workflows DAG bundle config",
			"Exactly one of `git` or `s3` must be set.",
		)
		return
	case gitSet && s3Set:
		diags.AddError(
			"Invalid Workflows DAG bundle config",
			"Only one of `git` or `s3` may be set, not both.",
		)
		return
	}

	if gitSet {
		validateGitBlock(ctx, model.Git, diags)
	}
	if s3Set {
		validateS3Block(ctx, model.S3, diags)
	}
}

func isSetObj(o types.Object) bool { return !o.IsNull() && !o.IsUnknown() }

func validateGitBlock(ctx context.Context, gitObj types.Object, diags *diag.Diagnostics) {
	var gm gitModel
	if d := gitObj.As(ctx, &gm, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); d.HasError() {
		return
	}
	// The server stores subdir without leading/trailing slashes. Rejecting them
	// here keeps state/config in sync — Terraform forbids ModifyPlan from
	// rewriting user-provided values, so the canonical form must come from the
	// user.
	if !gm.Subdir.IsNull() && !gm.Subdir.IsUnknown() {
		v := gm.Subdir.ValueString()
		if v != "" && v != strings.Trim(v, "/") {
			diags.AddError(
				"Invalid Workflows DAG bundle config",
				fmt.Sprintf("git.subdir %q must not have leading or trailing slashes; use %q.", v, strings.Trim(v, "/")),
			)
		}
	}
	if gm.Auth.IsNull() || gm.Auth.IsUnknown() {
		return
	}
	validateGitAuth(ctx, gm.Auth, diags)
}

func validateS3Block(ctx context.Context, s3Obj types.Object, diags *diag.Diagnostics) {
	var sm s3Model
	if d := s3Obj.As(ctx, &sm, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); d.HasError() {
		return
	}
	if sm.Auth.IsNull() || sm.Auth.IsUnknown() {
		return
	}
	validateS3Auth(ctx, sm.Auth, diags)
}

func validateGitAuth(ctx context.Context, authObj types.Object, diags *diag.Diagnostics) {
	var am gitAuthModel
	if d := authObj.As(ctx, &am, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); d.HasError() {
		return
	}
	if am.Type.IsNull() || am.Type.IsUnknown() {
		return
	}
	switch am.Type.ValueString() {
	case gitAuthTypeBasic:
		if isEmptyStr(am.Username) {
			diags.AddError("Invalid Workflows DAG bundle config", "git.auth.username is required when git.auth.type = basic.")
		}
		if isEmptyStr(am.Password) {
			diags.AddError("Invalid Workflows DAG bundle config", "git.auth.password is required when git.auth.type = basic.")
		}
	case gitAuthTypeNone:
		if isNonEmptyStr(am.Username) || isNonEmptyStr(am.Password) {
			diags.AddError("Invalid Workflows DAG bundle config", "git.auth.username/git.auth.password must not be set when git.auth.type = none.")
		}
	}
}

func validateS3Auth(ctx context.Context, authObj types.Object, diags *diag.Diagnostics) {
	var sm s3AuthModel
	if d := authObj.As(ctx, &sm, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); d.HasError() {
		return
	}
	if sm.Type.IsNull() || sm.Type.IsUnknown() {
		return
	}
	switch sm.Type.ValueString() {
	case s3AuthTypeAccessKey:
		if isEmptyStr(sm.AccessKeyID) {
			diags.AddError("Invalid Workflows DAG bundle config", "s3.auth.access_key_id is required when s3.auth.type = access_key.")
		}
		if isEmptyStr(sm.SecretAccessKey) {
			diags.AddError("Invalid Workflows DAG bundle config", "s3.auth.secret_access_key is required when s3.auth.type = access_key.")
		}
	case s3AuthTypeNone:
		if isNonEmptyStr(sm.AccessKeyID) || isNonEmptyStr(sm.SecretAccessKey) {
			diags.AddError("Invalid Workflows DAG bundle config", "s3.auth.access_key_id/s3.auth.secret_access_key must not be set when s3.auth.type = none.")
		}
	}
}

// isEmptyStr is the must-BE-set check used inside auth blocks: the empty
// literal is treated as "absent" because the server rejects empty credentials
// the same way as missing ones. Unknown defers.
func isEmptyStr(s types.String) bool {
	if s.IsUnknown() {
		return false
	}
	return s.IsNull() || s.ValueString() == ""
}

func isNonEmptyStr(s types.String) bool {
	return !s.IsNull() && !s.IsUnknown() && s.ValueString() != ""
}

func (r *dagBundleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := fmt.Sprintf("Workflows DAG bundle resource (Airflow 3 only). Bundle CRUD is synchronous server-side. %s", core.ResourceRegionFallbackDocstring)
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: features.AddExperimentDescription(description, features.WorkflowsExperiment, core.Resource),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: schemaDescriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: schemaDescriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: schemaDescriptions["region"],
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: schemaDescriptions["instance_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: schemaDescriptions["name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(bundleNamePattern, "must be a DNS label: lowercase alphanumeric and hyphens, starting with a letter, max 63 characters"),
				},
			},
			"git": schema.SingleNestedAttribute{
				Description: schemaDescriptions["git"],
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Description: schemaDescriptions["git.url"],
						Required:    true,
						Validators: []validator.String{
							workflowsUtils.URL(),
						},
					},
					"branch": schema.StringAttribute{
						Description: schemaDescriptions["git.branch"],
						Required:    true,
					},
					"subdir": schema.StringAttribute{
						Description: schemaDescriptions["git.subdir"],
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"refresh_interval": schema.Int32Attribute{
						Description: schemaDescriptions["git.refresh_interval"],
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int32{
							int32validator.Between(10, math.MaxInt32),
						},
						PlanModifiers: []planmodifier.Int32{
							int32planmodifier.UseStateForUnknown(),
						},
					},
					"auth": schema.SingleNestedAttribute{
						Description: schemaDescriptions["git.auth"],
						Required:    true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Description: schemaDescriptions["git.auth.type"],
								Required:    true,
								Validators: []validator.String{
									stringvalidator.OneOf(gitAuthTypeBasic, gitAuthTypeNone),
								},
							},
							"username": schema.StringAttribute{
								Description: schemaDescriptions["git.auth.username"],
								Optional:    true,
							},
							"password": schema.StringAttribute{
								Description: schemaDescriptions["git.auth.password"],
								Optional:    true,
								Sensitive:   true,
							},
						},
					},
				},
			},
			"s3": schema.SingleNestedAttribute{
				Description: schemaDescriptions["s3"],
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"bucket_name": schema.StringAttribute{
						Description: schemaDescriptions["s3.bucket_name"],
						Required:    true,
					},
					"endpoint": schema.StringAttribute{
						Description: schemaDescriptions["s3.endpoint"],
						Optional:    true,
						Validators: []validator.String{
							workflowsUtils.URL(),
						},
					},
					"prefix": schema.StringAttribute{
						Description: schemaDescriptions["s3.prefix"],
						Optional:    true,
					},
					"refresh_interval": schema.Int32Attribute{
						Description: schemaDescriptions["s3.refresh_interval"],
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int32{
							int32validator.Between(10, math.MaxInt32),
						},
						PlanModifiers: []planmodifier.Int32{
							int32planmodifier.UseStateForUnknown(),
						},
					},
					"auth": schema.SingleNestedAttribute{
						Description: schemaDescriptions["s3.auth"],
						Required:    true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Description: schemaDescriptions["s3.auth.type"],
								Required:    true,
								Validators: []validator.String{
									stringvalidator.OneOf(s3AuthTypeAccessKey, s3AuthTypeNone),
								},
							},
							"access_key_id": schema.StringAttribute{
								Description: schemaDescriptions["s3.auth.access_key_id"],
								Optional:    true,
							},
							"secret_access_key": schema.StringAttribute{
								Description: schemaDescriptions["s3.auth.secret_access_key"],
								Optional:    true,
								Sensitive:   true,
							},
						},
					},
				},
			},
		},
	}
}

func (r *dagBundleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceID := model.InstanceID.ValueString()
	name := model.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)
	ctx = tflog.SetField(ctx, "bundle_name", name)

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Workflows DAG bundle", fmt.Sprintf("Building payload: %v", err))
		return
	}

	createResp, err := r.client.DefaultAPI.CreateDagBundle(ctx, projectID, region, instanceID).CreateDagBundlePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Workflows DAG bundle", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	// Persist composite ID before mapFields can fail. The bundle is on the
	// server; without this, a mapFields error orphans it (the next apply hits
	// the RequiresReplace `name` field and gets a duplicate-name error).
	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  projectID,
		"region":      region,
		"instance_id": instanceID,
		"name":        name,
	})
	if resp.Diagnostics.HasError() {
		return
	}

	if err := mapFields(ctx, createResp, &model, region); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Workflows DAG bundle", fmt.Sprintf("Processing response: %v", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Bundle CRUD is synchronous, but the parent instance briefly enters
	// `updating` while the change is reconciled; wait for it to settle.
	if _, err := wait.UpdateInstanceWaitHandler(ctx, r.client.DefaultAPI, projectID, region, instanceID).WaitWithContext(ctx); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating Workflows DAG bundle", fmt.Sprintf("Waiting for instance to settle: %v", err))
		return
	}
	tflog.Debug(ctx, "Workflows DAG bundle created")
}

func (r *dagBundleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceID := model.InstanceID.ValueString()
	name := model.Name.ValueString()

	if name == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)
	ctx = tflog.SetField(ctx, "bundle_name", name)

	bundle, err := r.client.DefaultAPI.GetDagBundle(ctx, projectID, region, instanceID, name).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Workflows DAG bundle", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	if err := mapFields(ctx, bundle, &model, region); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading Workflows DAG bundle", fmt.Sprintf("Processing response: %v", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	tflog.Debug(ctx, "Workflows DAG bundle read")
}

func (r *dagBundleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { //nolint:gocritic // function signature required by Terraform
	var plan, state Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := plan.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(plan.Region)
	instanceID := plan.InstanceID.ValueString()
	name := plan.Name.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectID)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "instance_id", instanceID)
	ctx = tflog.SetField(ctx, "bundle_name", name)

	payload, err := toUpdatePayload(ctx, &plan, &state)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Workflows DAG bundle", fmt.Sprintf("Building payload: %v", err))
		return
	}

	updateResp, err := r.client.DefaultAPI.UpdateDagBundle(ctx, projectID, region, instanceID, name).UpdateDagBundlePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Workflows DAG bundle", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	if err := mapFields(ctx, updateResp, &plan, region); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Workflows DAG bundle", fmt.Sprintf("Processing response: %v", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := wait.UpdateInstanceWaitHandler(ctx, r.client.DefaultAPI, projectID, region, instanceID).WaitWithContext(ctx); err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating Workflows DAG bundle", fmt.Sprintf("Waiting for instance to settle: %v", err))
		return
	}
	tflog.Debug(ctx, "Workflows DAG bundle updated")
}

func (r *dagBundleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { //nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectID := model.ProjectID.ValueString()
	region := r.providerData.GetRegionWithOverride(model.Region)
	instanceID := model.InstanceID.ValueString()
	name := model.Name.ValueString()

	if err := r.client.DefaultAPI.DeleteDagBundle(ctx, projectID, region, instanceID, name).Execute(); err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		// Tolerate 404: the bundle may have already been deleted (e.g. if the
		// parent instance is being torn down and cascaded the delete).
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Workflows DAG bundle", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = core.LogResponse(ctx)

	// Bundle is gone server-side; remove from state now so a wait failure
	// below does not leave Terraform thinking the bundle still exists.
	resp.State.RemoveResource(ctx)

	if _, err := wait.UpdateInstanceWaitHandler(ctx, r.client.DefaultAPI, projectID, region, instanceID).WaitWithContext(ctx); err != nil {
		// If the parent instance itself is gone, the bundle is gone too.
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) && oapiErr.StatusCode == http.StatusNotFound {
			tflog.Debug(ctx, "Workflows DAG bundle deleted (parent instance gone)")
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting Workflows DAG bundle", fmt.Sprintf("Waiting for instance to settle: %v", err))
		return
	}
	tflog.Debug(ctx, "Workflows DAG bundle deleted")
}

// The expected format of the resource import identifier is: project_id,region,instance_id,name
func (r *dagBundleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing Workflows DAG bundle", fmt.Sprintf("Invalid import ID %q: expected format is `project_id`,`region`,`instance_id`,`name`", req.ID))
		return
	}
	// Cheap precondition: project_id and instance_id must be UUIDs. Caught
	// here so a typo in the import string fails fast instead of returning
	// an opaque server error on the subsequent Read.
	if !uuidRE.MatchString(idParts[0]) {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing Workflows DAG bundle", fmt.Sprintf("Invalid import ID %q: project_id segment %q is not a UUID", req.ID, idParts[0]))
		return
	}
	if !uuidRE.MatchString(idParts[2]) {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error importing Workflows DAG bundle", fmt.Sprintf("Invalid import ID %q: instance_id segment %q is not a UUID", req.ID, idParts[2]))
		return
	}
	ctx = tfutils.SetAndLogStateFields(ctx, &resp.Diagnostics, &resp.State, map[string]any{
		"project_id":  idParts[0],
		"region":      idParts[1],
		"instance_id": idParts[2],
		"name":        idParts[3],
	})
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Workflows DAG bundle state imported")
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func toCreatePayload(ctx context.Context, model *Model) (*workflows.CreateDagBundlePayload, error) {
	switch {
	case !model.Git.IsNull() && !model.Git.IsUnknown():
		git, err := buildGitDagBundle(ctx, model)
		if err != nil {
			return nil, err
		}
		wrapped := workflows.GitDagBundleAsCreateDagBundlePayload(git)
		return &wrapped, nil
	case !model.S3.IsNull() && !model.S3.IsUnknown():
		s3, err := buildS3DagBundle(ctx, model)
		if err != nil {
			return nil, err
		}
		wrapped := workflows.S3DagBundleAsCreateDagBundlePayload(s3)
		return &wrapped, nil
	default:
		return nil, errors.New("exactly one of `git` or `s3` must be set")
	}
}

func toUpdatePayload(ctx context.Context, plan, state *Model) (*workflows.UpdateDagBundlePayload, error) {
	switch {
	case !plan.Git.IsNull() && !plan.Git.IsUnknown():
		patch, err := toUpdateGitDagBundlePayload(ctx, plan, state)
		if err != nil {
			return nil, err
		}
		wrapped := workflows.UpdateGitDagBundlePayloadAsUpdateDagBundlePayload(patch)
		return &wrapped, nil
	case !plan.S3.IsNull() && !plan.S3.IsUnknown():
		patch, err := toUpdateS3DagBundlePayload(ctx, plan, state)
		if err != nil {
			return nil, err
		}
		wrapped := workflows.UpdateS3DagBundlePayloadAsUpdateDagBundlePayload(patch)
		return &wrapped, nil
	default:
		return nil, errors.New("exactly one of `git` or `s3` must be set")
	}
}

func extractGitModel(ctx context.Context, m *Model) (*gitModel, error) {
	var gm gitModel
	diags := m.Git.As(ctx, &gm, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting git object: %w", core.DiagsToError(diags))
	}
	return &gm, nil
}

func extractS3Model(ctx context.Context, m *Model) (*s3Model, error) {
	var sm s3Model
	diags := m.S3.As(ctx, &sm, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting s3 object: %w", core.DiagsToError(diags))
	}
	return &sm, nil
}

func buildGitDagBundle(ctx context.Context, model *Model) (*workflows.GitDagBundle, error) {
	gm, err := extractGitModel(ctx, model)
	if err != nil {
		return nil, err
	}
	if gm.URL.IsNull() || gm.URL.IsUnknown() {
		return nil, errors.New("git.url is required")
	}
	if gm.Branch.IsNull() || gm.Branch.IsUnknown() {
		return nil, errors.New("git.branch is required")
	}
	auth, err := buildGitAuth(ctx, gm)
	if err != nil {
		return nil, err
	}
	bundle := &workflows.GitDagBundle{
		Type:            workflows.GITDAGBUNDLETYPE_GIT,
		Name:            model.Name.ValueString(),
		Url:             gm.URL.ValueString(),
		Branch:          gm.Branch.ValueString(),
		Subdir:          conversion.StringValueToPointer(gm.Subdir),
		RefreshInterval: conversion.Int32ValueToPointer(gm.RefreshInterval),
		Auth:            *auth,
	}
	return bundle, nil
}

func buildS3DagBundle(ctx context.Context, model *Model) (*workflows.S3DagBundle, error) {
	sm, err := extractS3Model(ctx, model)
	if err != nil {
		return nil, err
	}
	if sm.BucketName.IsNull() || sm.BucketName.IsUnknown() {
		return nil, errors.New("s3.bucket_name is required")
	}
	auth, err := buildS3Auth(ctx, sm)
	if err != nil {
		return nil, err
	}
	bundle := &workflows.S3DagBundle{
		Type:            workflows.S3DAGBUNDLETYPE_S3,
		Name:            model.Name.ValueString(),
		BucketName:      sm.BucketName.ValueString(),
		Endpoint:        conversion.StringValueToPointer(sm.Endpoint),
		Prefix:          conversion.StringValueToPointer(sm.Prefix),
		RefreshInterval: conversion.Int32ValueToPointer(sm.RefreshInterval),
		S3Auth:          *auth,
	}
	return bundle, nil
}

// toUpdateGitDagBundlePayload builds a PATCH payload. Subdir uses "empty
// string clears" semantics. Auth is only attached when set in the plan — when
// omitted, the server leaves the stored credentials untouched, which lets the
// user update other fields without rotating the password every time.
func toUpdateGitDagBundlePayload(ctx context.Context, plan, state *Model) (*workflows.UpdateGitDagBundlePayload, error) {
	gm, err := extractGitModel(ctx, plan)
	if err != nil {
		return nil, err
	}
	var prior gitModel
	if !state.Git.IsNull() && !state.Git.IsUnknown() {
		if d := state.Git.As(ctx, &prior, basetypes.ObjectAsOptions{}); d.HasError() {
			return nil, fmt.Errorf("converting prior git object: %w", core.DiagsToError(d))
		}
	}
	patch := &workflows.UpdateGitDagBundlePayload{
		Type:            workflows.UPDATEGITDAGBUNDLEPAYLOADTYPE_GIT,
		Url:             conversion.StringValueToPointer(gm.URL),
		Branch:          conversion.StringValueToPointer(gm.Branch),
		Subdir:          conversion.ClearableString(gm.Subdir, prior.Subdir),
		RefreshInterval: conversion.Int32ValueToPointer(gm.RefreshInterval),
	}
	if !gm.Auth.IsNull() && !gm.Auth.IsUnknown() {
		auth, err := buildGitAuth(ctx, gm)
		if err != nil {
			return nil, err
		}
		patch.Auth = auth
	}
	return patch, nil
}

// toUpdateS3DagBundlePayload builds a PATCH payload. Endpoint and prefix use
// "empty string clears" semantics. Auth is only attached when set (see
// toUpdateGitDagBundlePayload for rationale).
func toUpdateS3DagBundlePayload(ctx context.Context, plan, state *Model) (*workflows.UpdateS3DagBundlePayload, error) {
	sm, err := extractS3Model(ctx, plan)
	if err != nil {
		return nil, err
	}
	var prior s3Model
	if !state.S3.IsNull() && !state.S3.IsUnknown() {
		if d := state.S3.As(ctx, &prior, basetypes.ObjectAsOptions{}); d.HasError() {
			return nil, fmt.Errorf("converting prior s3 object: %w", core.DiagsToError(d))
		}
	}
	patch := &workflows.UpdateS3DagBundlePayload{
		Type:            workflows.UPDATES3DAGBUNDLEPAYLOADTYPE_S3,
		BucketName:      conversion.StringValueToPointer(sm.BucketName),
		Endpoint:        conversion.ClearableString(sm.Endpoint, prior.Endpoint),
		Prefix:          conversion.ClearableString(sm.Prefix, prior.Prefix),
		RefreshInterval: conversion.Int32ValueToPointer(sm.RefreshInterval),
	}
	if !sm.Auth.IsNull() && !sm.Auth.IsUnknown() {
		auth, err := buildS3Auth(ctx, sm)
		if err != nil {
			return nil, err
		}
		patch.S3Auth = auth
	}
	return patch, nil
}

func buildGitAuth(ctx context.Context, gm *gitModel) (*workflows.GitAuth, error) {
	if gm.Auth.IsNull() || gm.Auth.IsUnknown() {
		return nil, errors.New("git.auth is required")
	}
	var am gitAuthModel
	diags := gm.Auth.As(ctx, &am, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting git auth object: %w", core.DiagsToError(diags))
	}

	switch am.Type.ValueString() {
	case gitAuthTypeBasic:
		basic := &workflows.BasicAuth{
			Type:     conversion.StringValueToPointer(am.Type),
			Username: conversion.StringValueToPointer(am.Username),
			Password: conversion.StringValueToPointer(am.Password),
		}
		wrapped := workflows.BasicAuthAsGitAuth(basic)
		return &wrapped, nil
	case gitAuthTypeNone:
		none := &workflows.NoAuth{
			Type: conversion.StringValueToPointer(am.Type),
		}
		wrapped := workflows.NoAuthAsGitAuth(none)
		return &wrapped, nil
	default:
		return nil, fmt.Errorf("unsupported git.auth.type %q", am.Type.ValueString())
	}
}

func buildS3Auth(ctx context.Context, sm *s3Model) (*workflows.S3Auth, error) {
	if sm.Auth.IsNull() || sm.Auth.IsUnknown() {
		return nil, errors.New("s3.auth is required")
	}
	var am s3AuthModel
	diags := sm.Auth.As(ctx, &am, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("converting s3 auth object: %w", core.DiagsToError(diags))
	}

	switch am.Type.ValueString() {
	case s3AuthTypeAccessKey:
		ak := &workflows.S3AccessKeyAuth{
			Type:            workflows.S3ACCESSKEYAUTHTYPE_ACCESS_KEY,
			AccessKeyId:     am.AccessKeyID.ValueString(),
			SecretAccessKey: am.SecretAccessKey.ValueString(),
		}
		wrapped := workflows.S3AccessKeyAuthAsS3Auth(ak)
		return &wrapped, nil
	case s3AuthTypeNone:
		none := &workflows.NoAuth{
			Type: conversion.StringValueToPointer(am.Type),
		}
		wrapped := workflows.NoAuthAsS3Auth(none)
		return &wrapped, nil
	default:
		return nil, fmt.Errorf("unsupported s3.auth.type %q", am.Type.ValueString())
	}
}

// mapFields populates `model` from a server response. It carries forward the
// sensitive password / secret_access_key from the prior plan/state because the
// API never returns them.
func mapFields(ctx context.Context, bundle *workflows.DagBundleResponse, model *Model, region string) error {
	if bundle == nil {
		return errors.New("bundle is nil")
	}
	if model == nil {
		return errors.New("model is nil")
	}

	priorPassword := priorGitPassword(ctx, model)
	priorSecret := priorS3Secret(ctx, model)

	switch {
	case bundle.GitDagBundleResponse != nil:
		g := bundle.GitDagBundleResponse
		model.Name = types.StringValue(g.Name)

		auth := gitAuthModel{Password: priorPassword}
		switch {
		case g.Auth.BasicAuthResponse != nil:
			auth.Type = types.StringValue(gitAuthTypeBasic)
			auth.Username = types.StringPointerValue(g.Auth.BasicAuthResponse.Username)
		case g.Auth.NoAuth != nil:
			auth.Type = types.StringValue(gitAuthTypeNone)
		default:
			return errors.New("server returned an unknown git auth variant; upgrade the provider")
		}
		authObj, diags := types.ObjectValueFrom(ctx, gitAuthTypes, auth)
		if diags.HasError() {
			return fmt.Errorf("mapping git auth: %w", core.DiagsToError(diags))
		}
		gitObj, diags := types.ObjectValueFrom(ctx, gitTypes, gitModel{
			URL:             types.StringValue(g.Url),
			Branch:          types.StringValue(g.Branch),
			Subdir:          types.StringPointerValue(g.Subdir),
			RefreshInterval: types.Int32PointerValue(g.RefreshInterval),
			Auth:            authObj,
		})
		if diags.HasError() {
			return fmt.Errorf("mapping git block: %w", core.DiagsToError(diags))
		}
		model.Git = gitObj
		model.S3 = types.ObjectNull(s3Types)

	case bundle.S3DagBundleResponse != nil:
		s := bundle.S3DagBundleResponse
		model.Name = types.StringValue(s.Name)

		auth := s3AuthModel{SecretAccessKey: priorSecret}
		switch {
		case s.S3Auth.S3AccessKeyAuthResponse != nil:
			auth.Type = types.StringValue(s3AuthTypeAccessKey)
			auth.AccessKeyID = types.StringValue(s.S3Auth.S3AccessKeyAuthResponse.AccessKeyId)
		case s.S3Auth.NoAuth != nil:
			auth.Type = types.StringValue(s3AuthTypeNone)
		default:
			return errors.New("server returned an unknown s3 auth variant; upgrade the provider")
		}
		authObj, diags := types.ObjectValueFrom(ctx, s3AuthTypes, auth)
		if diags.HasError() {
			return fmt.Errorf("mapping s3 auth: %w", core.DiagsToError(diags))
		}
		s3Obj, diags := types.ObjectValueFrom(ctx, s3Types, s3Model{
			BucketName:      types.StringValue(s.BucketName),
			Endpoint:        types.StringPointerValue(s.Endpoint),
			Prefix:          types.StringPointerValue(s.Prefix),
			RefreshInterval: types.Int32PointerValue(s.RefreshInterval),
			Auth:            authObj,
		})
		if diags.HasError() {
			return fmt.Errorf("mapping s3 block: %w", core.DiagsToError(diags))
		}
		model.S3 = s3Obj
		model.Git = types.ObjectNull(gitTypes)

	default:
		return errors.New("server returned an unknown DagBundle variant; upgrade the provider")
	}

	model.Region = types.StringValue(region)
	model.ID = tfutils.BuildInternalTerraformId(model.ProjectID.ValueString(), region, model.InstanceID.ValueString(), model.Name.ValueString())
	return nil
}

// priorGitPassword extracts the password from the prior model (if any) so the
// caller can carry it forward — the API never returns the password back.
func priorGitPassword(ctx context.Context, model *Model) types.String {
	if model.Git.IsNull() || model.Git.IsUnknown() {
		return types.StringNull()
	}
	var gm gitModel
	if d := model.Git.As(ctx, &gm, basetypes.ObjectAsOptions{}); d.HasError() {
		return types.StringNull()
	}
	if gm.Auth.IsNull() || gm.Auth.IsUnknown() {
		return types.StringNull()
	}
	var am gitAuthModel
	if d := gm.Auth.As(ctx, &am, basetypes.ObjectAsOptions{}); d.HasError() {
		return types.StringNull()
	}
	return am.Password
}

// priorS3Secret extracts the secret_access_key from the prior model (if any)
// so the caller can carry it forward — the API never returns it back.
func priorS3Secret(ctx context.Context, model *Model) types.String {
	if model.S3.IsNull() || model.S3.IsUnknown() {
		return types.StringNull()
	}
	var sm s3Model
	if d := model.S3.As(ctx, &sm, basetypes.ObjectAsOptions{}); d.HasError() {
		return types.StringNull()
	}
	if sm.Auth.IsNull() || sm.Auth.IsUnknown() {
		return types.StringNull()
	}
	var am s3AuthModel
	if d := sm.Auth.As(ctx, &am, basetypes.ObjectAsOptions{}); d.HasError() {
		return types.StringNull()
	}
	return am.SecretAccessKey
}
