package secretsmanager

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	secretsmanagerV1Alpha "github.com/stackitcloud/stackit-sdk-go/services/secretsmanager/v1alphaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iam/rolebindings/v1/generic"
)

func NewSecretsmanagerSecretGroupRoleBindingResource() resource.Resource {
	return &generic.RoleBindingResource[secretsmanagerV1Alpha.DefaultAPI]{
		ApiName:      "secretsmanager",
		ResourceType: "secret_group",
		ApiClientExtractor: func(clientCollection core.RoleBindingClientCollection) secretsmanagerV1Alpha.DefaultAPI {
			return clientCollection.SecretsmanagerV1AlphaClient
		},
		ExecCreateRequest: func(ctx context.Context, client secretsmanagerV1Alpha.DefaultAPI, region, resourceId, role, subject string) (generic.GenericRoleBindingResponse, error) {
			payload := secretsmanagerV1Alpha.AddSecretGroupRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.AddSecretGroupRoleBindings(ctx, region, resourceId).AddSecretGroupRoleBindingsPayload(payload).Execute()
		},
		ExecReadRequest: func(ctx context.Context, client secretsmanagerV1Alpha.DefaultAPI, region, resourceId, role, subject string) (generic.GenericRoleBindingResponse, error) {
			payload := secretsmanagerV1Alpha.GetSecretGroupRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.GetSecretGroupRoleBindings(ctx, region, resourceId).GetSecretGroupRoleBindingsPayload(payload).Execute()
		},
		ExecUpdateRequest: func(ctx context.Context, client secretsmanagerV1Alpha.DefaultAPI, region, resourceId, role, subject string) (generic.GenericRoleBindingResponse, error) {
			payload := secretsmanagerV1Alpha.EditSecretGroupRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.EditSecretGroupRoleBindings(ctx, region, resourceId).EditSecretGroupRoleBindingsPayload(payload).Execute()
		},
		ExecDeleteRequest: func(ctx context.Context, client secretsmanagerV1Alpha.DefaultAPI, region, resourceId, role, subject string) error {
			payload := secretsmanagerV1Alpha.RemoveSecretGroupRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.RemoveSecretGroupRoleBindings(ctx, region, resourceId).RemoveSecretGroupRoleBindingsPayload(payload).Execute()
		},
	}
}

func NewSecretsmanagerSecretGroupRoleBindingsDatasource() datasource.DataSource {
	return &generic.RoleBindingDatasource[secretsmanagerV1Alpha.DefaultAPI]{
		ApiName:      "secretsmanager",
		ResourceType: "secret_group",
		ApiClientExtractor: func(clientCollection core.RoleBindingClientCollection) secretsmanagerV1Alpha.DefaultAPI {
			return clientCollection.SecretsmanagerV1AlphaClient
		},
		ExecReadRequest: func(ctx context.Context, client secretsmanagerV1Alpha.DefaultAPI, region, resourceId string) ([]generic.GenericRoleBindingResponse, error) {
			resp, err := client.ListSecretGroupRoleBindings(ctx, region, resourceId).Execute()
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
