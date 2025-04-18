package cdn

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/cdn"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &distributionResource{}
	_ resource.ResourceWithConfigure   = &distributionResource{}
	_ resource.ResourceWithImportState = &distributionResource{}
)

type Model struct {
	ID             types.String `tfsdk:"id"`              // Required by Terraform
	DistributionId types.String `tfsdk:"distribution_id"` // DistributionID associated with the cdn distribution
	ProjectId      types.String `tfsdk:"project_id"`      // ProjectId associated with the cdn distribution
	Status         types.String
	CreatedAt      types.String
	UpdatedAt     types.String
	Errors         types.List
	Domains        types.List
	Config         types.Object
}

var configTypes = map[string]attr.Type{
	"backend": types.ObjectType{AttrTypes: backendTypes},
	"regions": types.ListType{ElemType: types.StringType},
}

var backendTypes = map[string]attr.Type{
	"type": types.StringType,
	"originUrl": types.StringType,
	"originRequestHeaders": types.ListType{ElemType: types.StringType},
}

var domainTypes = map[string]attr.Type{
	"name": types.StringType,
	"status": types.StringType,
	"type": types.StringType,
	"errors": types.ListType{ElemType: types.StringType},
}

type distributionResource struct {
	client *cdn.APIClient
}

func (r *distributionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	panic("unimplemented")
}

func (r *distributionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_distribution"
}

func (r *distributionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	panic("unimplemented")
}

func (r *distributionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	panic("unimplemented")
}

func (r *distributionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	panic("unimplemented")
}

func (r *distributionResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	panic("unimplemented")
}

func (r *distributionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	panic("unimplemented")
}

func (r *distributionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	panic("unimplemented")
}
