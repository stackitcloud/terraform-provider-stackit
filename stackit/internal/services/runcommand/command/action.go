package runcommand

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/runcommand/v1api"
	"github.com/stackitcloud/stackit-sdk-go/services/runcommand/v1api/wait"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	runCommandUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/runcommand/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ action.Action              = &runCommandAction{}
	_ action.ActionWithConfigure = &runCommandAction{}
)

type runCommandModel struct {
	ProjectId           types.String `tfsdk:"project_id"`
	ServerId            types.String `tfsdk:"server_id"`
	Region              types.String `tfsdk:"region"`
	CommandTemplateName types.String `tfsdk:"command_template_name"`
	Parameters          types.Map    `tfsdk:"parameters"`
}

// NewRunCommandAction is a helper function to simplify the provider implementation.
func NewRunCommandAction() action.Action {
	return &runCommandAction{}
}

// runCommandAction is the action implementation.
type runCommandAction struct {
	client       *v1api.APIClient
	providerData core.ProviderData
}

// Metadata returns the action type name.
func (a *runCommandAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_run_command"
}

// Configure adds the provider configured client to the action.
func (a *runCommandAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	var ok bool
	a.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	a.client = runCommandUtils.ConfigureClient(ctx, &a.providerData, a.providerData.DefaultRegion, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Run command client configured")
}

// Schema defines the schema for the action.
func (a *runCommandAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	descriptions := map[string]string{
		"main":                  "Executes a command on an IaaS server using the STACKIT Run Commands API. " + core.ResourceRegionFallbackDocstring,
		"project_id":            "STACKIT Project ID to which the server belongs.",
		"server_id":             "The ID of the server on which to execute the command.",
		"region":                "The region of the server. If not defined, the provider default_region is used.",
		"command_template_name": "The name of the command template to execute (e.g. RunShellScript). Available templates can be listed with: `stackit server command template list`",
		"parameters":            "Optional parameters passed to the command template as key-value pairs.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"server_id": schema.StringAttribute{
				Description: descriptions["server_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"region": schema.StringAttribute{
				Description: descriptions["region"],
				Optional:    true,
			},
			"command_template_name": schema.StringAttribute{
				Description: descriptions["command_template_name"],
				Required:    true,
			},
			"parameters": schema.MapAttribute{
				Description: descriptions["parameters"],
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Invoke executes the run command action.
func (a *runCommandAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var model runCommandModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	serverId := model.ServerId.ValueString()
	region := a.providerData.GetRegionWithOverride(model.Region)

	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "server_id", serverId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "command_template_name", model.CommandTemplateName.ValueString())

	payload, err := toCreatePayload(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error invoking run command", fmt.Sprintf("Building API payload: %v", err))
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{
		Message: fmt.Sprintf("Waiting for agent on server %s to be ready...", serverId),
	})

	// waits for the agent to register (404 while booting) and submits the command in one step
	createResp, err := wait.AgentReadyWaitHandler(ctx, a.client.DefaultAPI, projectId, serverId, *payload).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error invoking run command", fmt.Sprintf("Waiting for agent / calling API: %v", err))
		return
	}
	if createResp == nil || createResp.Id == nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error invoking run command", "API returned empty response or missing command ID")
		return
	}

	commandId := createResp.GetId()
	commandIdStr := strconv.Itoa(int(commandId))
	ctx = tflog.SetField(ctx, "command_id", commandIdStr)

	resp.SendProgress(action.InvokeProgressEvent{
		Message: fmt.Sprintf("Command %q submitted (ID: %s). Waiting for completion...", model.CommandTemplateName.ValueString(), commandIdStr),
	})

	details, err := wait.RunCommandWaitHandler(ctx, a.client.DefaultAPI, projectId, serverId, commandIdStr).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error waiting for run command", fmt.Sprintf("Polling API: %v", err))
		return
	}

	if details.GetStatus() == v1api.COMMANDDETAILSSTATUS_FAILED {
		resp.Diagnostics.AddError(
			"Run command failed",
			fmt.Sprintf("Command %s finished with status %q (exit code: %d).\nOutput:\n%s", commandIdStr, details.GetStatus(), details.GetExitCode(), details.GetOutput()),
		)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Run command %s completed successfully", commandIdStr))
}

func toCreatePayload(ctx context.Context, model *runCommandModel) (*v1api.CreateCommandPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	payload := v1api.NewCreateCommandPayload(model.CommandTemplateName.ValueString())

	if !model.Parameters.IsNull() && !model.Parameters.IsUnknown() {
		params := map[string]string{}
		diags := model.Parameters.ElementsAs(ctx, &params, false)
		if diags.HasError() {
			return nil, fmt.Errorf("converting parameters: %v", diags.Errors())
		}
		payload.SetParameters(params)
	}

	return payload, nil
}
