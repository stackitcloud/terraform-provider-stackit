package federated_identity_provider

import "github.com/hashicorp/terraform-plugin-framework/types"

// Model describes the resource data model.
type Model struct {
	Id                  types.String `tfsdk:"id"`
	ProjectId           types.String `tfsdk:"project_id"`
	ServiceAccountEmail types.String `tfsdk:"service_account_email"`
	FederationId        types.String `tfsdk:"federation_id"`
	Name                types.String `tfsdk:"name"`
	Issuer              types.String `tfsdk:"issuer"`
	Assertions          types.List   `tfsdk:"assertions"`
}

// AssertionModel describes an assertion in the assertions list.
type AssertionModel struct {
	Item     types.String `tfsdk:"item"`
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}
