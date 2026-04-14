package v1

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"

	secretsmanager2 "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iam/rolebindings/v1/services/secretsmanager"
)

// NewRoleBindingResources is a helper function to simplify the provider implementation.
func NewRoleBindingResources() []func() resource.Resource {
	return []func() resource.Resource{
		// secretsmanager
		secretsmanager2.NewSecretsmanagerInstanceRoleBindingResource,
		secretsmanager2.NewSecretsmanagerSecretGroupRoleBindingResource,
	}
}
