package secretsmanager

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	secretsmanagerV1Alpha "github.com/stackitcloud/stackit-sdk-go/services/secretsmanager/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iam/rolebindings/v1/generic"

	secretsmanagerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/secretsmanager/utils"
)

func NewSecretsmanagerInstanceRoleBindingResource() resource.Resource {
	return &generic.RoleBindingResource[secretsmanagerV1Alpha.APIClient]{
		ApiName:          "secretsmanager",
		ResourceType:     "instance",
		ApiClientFactory: secretsmanagerUtils.ConfigureV1AlphaClient,
		ExecCreateRequest: func(ctx context.Context, client *secretsmanagerV1Alpha.APIClient, region, resourceId, role, subject string) (generic.GenericRoleBindingResponse, error) {
			payload := secretsmanagerV1Alpha.AddInstanceRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.DefaultAPI.AddInstanceRoleBindings(ctx, region, resourceId).AddInstanceRoleBindingsPayload(payload).Execute()
		},
		ExecReadRequest: func(ctx context.Context, client *secretsmanagerV1Alpha.APIClient, region, resourceId, role, subject string) (generic.GenericRoleBindingResponse, error) {
			payload := secretsmanagerV1Alpha.GetInstanceRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.DefaultAPI.GetInstanceRoleBindings(ctx, region, resourceId).GetInstanceRoleBindingsPayload(payload).Execute()
		},
		ExecUpdateRequest: func(ctx context.Context, client *secretsmanagerV1Alpha.APIClient, region, resourceId, role, subject string) (generic.GenericRoleBindingResponse, error) {
			payload := secretsmanagerV1Alpha.EditInstanceRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.DefaultAPI.EditInstanceRoleBindings(ctx, region, resourceId).EditInstanceRoleBindingsPayload(payload).Execute()
		},
		ExecDeleteRequest: func(ctx context.Context, client *secretsmanagerV1Alpha.APIClient, region, resourceId, role, subject string) error {
			payload := secretsmanagerV1Alpha.RemoveInstanceRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.DefaultAPI.RemoveInstanceRoleBindings(ctx, region, resourceId).RemoveInstanceRoleBindingsPayload(payload).Execute()
		},
	}
}
