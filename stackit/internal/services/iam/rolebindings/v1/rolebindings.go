package v1

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iam/rolebindings/v1/services/secretsmanager"
)

// NewRoleBindingResources is a helper function to simplify the provider implementation.
func NewRoleBindingResources() []func() resource.Resource {
	return []func() resource.Resource{
		// secretsmanager
		secretsmanager.NewSecretsmanagerInstanceRoleBindingResource,
		secretsmanager.NewSecretsmanagerSecretGroupRoleBindingResource,
	}
}

func NewRoleBindingsDatasources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// secretsmanager
		secretsmanager.NewSecretsmanagerInstanceRoleBindingsDatasource,
		secretsmanager.NewSecretsmanagerSecretGroupRoleBindingsDatasource,
	}
}
