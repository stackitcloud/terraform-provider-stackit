package v1

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iam/policy/v1/generic"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iam/policy/v1/services/resourcemanager"
)

// NewIamPolicyResources is a helper function to simplify the provider implementation.
func NewIamPolicyResources() []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource { return new(resourcemanager.NewFolderRoleBindingResource()) },
		func() resource.Resource { return new(resourcemanager.NewOrganizationRoleBindingResource()) },
		func() resource.Resource { return new(resourcemanager.NewProjectRoleBindingResource()) },
	}
}

// NewIamPolicyDatasources is a helper function to simplify the provider implementation.
func NewIamPolicyDatasources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		generic.NewDatasource(resourcemanager.NewFolderRoleBindingResource()),
		generic.NewDatasource(resourcemanager.NewOrganizationRoleBindingResource()),
		generic.NewDatasource(resourcemanager.NewProjectRoleBindingResource()),
	}
}
