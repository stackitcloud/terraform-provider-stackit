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

func NewSecretsmanagerInstanceRoleBindingResource() resource.Resource {
	return &generic.RoleBindingResource[secretsmanagerV1Alpha.DefaultAPI]{
		ApiName:      "secretsmanager",
		ResourceType: "instance",
		ApiClientExtractor: func(clientCollection core.RoleBindingClientCollection) secretsmanagerV1Alpha.DefaultAPI {
			return clientCollection.SecretsmanagerV1AlphaClient
		},
		ExecCreateRequest: func(ctx context.Context, client secretsmanagerV1Alpha.DefaultAPI, region, resourceId, role, subject string) (generic.GenericRoleBindingResponse, error) {
			payload := secretsmanagerV1Alpha.AddInstanceRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.AddInstanceRoleBindings(ctx, region, resourceId).AddInstanceRoleBindingsPayload(payload).Execute()
		},
		ExecReadRequest: func(ctx context.Context, client secretsmanagerV1Alpha.DefaultAPI, region, resourceId, role, subject string) (generic.GenericRoleBindingResponse, error) {
			payload := secretsmanagerV1Alpha.GetInstanceRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.GetInstanceRoleBindings(ctx, region, resourceId).GetInstanceRoleBindingsPayload(payload).Execute()
		},
		ExecUpdateRequest: func(ctx context.Context, client secretsmanagerV1Alpha.DefaultAPI, region, resourceId, role, subject string) (generic.GenericRoleBindingResponse, error) {
			payload := secretsmanagerV1Alpha.EditInstanceRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.EditInstanceRoleBindings(ctx, region, resourceId).EditInstanceRoleBindingsPayload(payload).Execute()
		},
		ExecDeleteRequest: func(ctx context.Context, client secretsmanagerV1Alpha.DefaultAPI, region, resourceId, role, subject string) error {
			payload := secretsmanagerV1Alpha.RemoveInstanceRoleBindingsPayload{
				Role:    role,
				Subject: subject,
			}

			return client.RemoveInstanceRoleBindings(ctx, region, resourceId).RemoveInstanceRoleBindingsPayload(payload).Execute()
		},
	}
}

func NewSecretsmanagerInstanceRoleBindingsDatasource() datasource.DataSource {
	return &generic.RoleBindingDatasource[secretsmanagerV1Alpha.DefaultAPI]{
		ApiName:      "secretsmanager",
		ResourceType: "instance",
		ApiClientExtractor: func(clientCollection core.RoleBindingClientCollection) secretsmanagerV1Alpha.DefaultAPI {
			return clientCollection.SecretsmanagerV1AlphaClient
		},
		ExecReadRequest: func(ctx context.Context, client secretsmanagerV1Alpha.DefaultAPI, region, resourceId string) ([]generic.GenericRoleBindingResponse, error) {
			resp, err := client.ListInstanceRoleBindings(ctx, region, resourceId).Execute()
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
