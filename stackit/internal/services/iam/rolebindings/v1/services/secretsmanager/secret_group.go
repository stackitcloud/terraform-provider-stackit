package secretsmanager

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	secretsmanagerV1Alpha "github.com/stackitcloud/stackit-sdk-go/services/secretsmanager/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iam/rolebindings/v1/generic"

	secretsmanagerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/secretsmanager/utils"
)

func NewSecretsmanagerSecretGroupRoleBindingResource() resource.Resource {
	return &generic.RoleBindingResource[secretsmanagerV1Alpha.APIClient]{
		ApiName:          "secretsmanager",
		ResourceType:     "secret_group",
		ApiClientFactory: secretsmanagerUtils.ConfigureV1AlphaClient,
		ExecCreateRequest: func(ctx context.Context, client *secretsmanagerV1Alpha.APIClient, region, resourceId, role, subject string) (generic.GenericRoleBindingResponse, error) {
			payload := secretsmanagerV1Alpha.AddSecretGroupRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.DefaultAPI.AddSecretGroupRoleBindings(ctx, region, resourceId).AddSecretGroupRoleBindingsPayload(payload).Execute()
		},
		ExecReadRequest: func(ctx context.Context, client *secretsmanagerV1Alpha.APIClient, region, resourceId, role, subject string) (generic.GenericRoleBindingResponse, error) {
			payload := secretsmanagerV1Alpha.GetSecretGroupRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.DefaultAPI.GetSecretGroupRoleBindings(ctx, region, resourceId).GetSecretGroupRoleBindingsPayload(payload).Execute()
		},
		ExecUpdateRequest: func(ctx context.Context, client *secretsmanagerV1Alpha.APIClient, region, resourceId, role, subject string) (generic.GenericRoleBindingResponse, error) {
			payload := secretsmanagerV1Alpha.EditSecretGroupRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.DefaultAPI.EditSecretGroupRoleBindings(ctx, region, resourceId).EditSecretGroupRoleBindingsPayload(payload).Execute()
		},
		ExecDeleteRequest: func(ctx context.Context, client *secretsmanagerV1Alpha.APIClient, region, resourceId, role, subject string) error {
			payload := secretsmanagerV1Alpha.RemoveSecretGroupRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.DefaultAPI.RemoveSecretGroupRoleBindings(ctx, region, resourceId).RemoveSecretGroupRoleBindingsPayload(payload).Execute()
		},
	}
}

func NewSecretsmanagerSecretGroupRoleBindingsDatasource() datasource.DataSource {
	return &generic.RoleBindingDatasource[secretsmanagerV1Alpha.APIClient]{
		ApiName:          "secretsmanager",
		ResourceType:     "secret_group",
		ApiClientFactory: secretsmanagerUtils.ConfigureV1AlphaClient,
		ExecReadRequest: func(ctx context.Context, client *secretsmanagerV1Alpha.APIClient, region, resourceId string) ([]generic.GenericRoleBindingResponse, error) {
			resp, err := client.DefaultAPI.ListSecretGroupRoleBindings(ctx, region, resourceId).Execute()
			if err != nil {
				return nil, err
			}

			if resp == nil {
				return nil, nil
			}

			return utils.Map(resp.RoleBindings, func(t secretsmanagerV1Alpha.RoleBinding) generic.GenericRoleBindingResponse {
				return &t
			}), nil
		},
	}
}
