package resourcemanager

import (
	"context"

	resourcemanagerV1Beta "github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iam/policy/v1/generic"
	resourcemanagerUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/resourcemanager/utils"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func mapper(policy *resourcemanagerV1Beta.IAMPolicy, err error) (*generic.IAMPolicy, error) {
	if policy == nil {
		return nil, err
	}

	return &generic.IAMPolicy{
		Etag: policy.Etag,
		RoleBindings: utils.Map(policy.RoleBindings, func(t resourcemanagerV1Beta.RoleBinding) generic.RoleBinding {
			return &t
		}),
	}, err
}

func NewFolderRoleBindingResource() generic.GlobalIAMPolicyResource[resourcemanagerV1Beta.DefaultAPI] {
	const resourceType = "folder"

	return generic.GlobalIAMPolicyResource[resourcemanagerV1Beta.DefaultAPI]{
		ApiName:          "resourcemanager",
		ResourceType:     resourceType,
		Experimental:     true,
		ApiClientFactory: resourcemanagerUtils.ConfigureV1BetaClient,
		ExecReadRequest: func(ctx context.Context, client resourcemanagerV1Beta.DefaultAPI, resourceId string) (*generic.IAMPolicy, error) {
			return mapper(client.GetFolderIamPolicy(ctx, resourceId).Execute())
		},
		ExecUpdateRequest: func(ctx context.Context, client resourcemanagerV1Beta.DefaultAPI, resourceId string, etag *string, roleBindings []generic.RoleBinding) (*generic.IAMPolicy, error) {
			payload := resourcemanagerV1Beta.UpdateFolderIamPolicyPayload{
				ResourceType: resourceType,
				ResourceId:   resourceId,
				RoleBindings: utils.Map(roleBindings, func(t generic.RoleBinding) resourcemanagerV1Beta.RoleBinding {
					return resourcemanagerV1Beta.RoleBinding{
						Role:    t.GetRole(),
						Subject: t.GetSubject(),
					}
				}),
				Etag: etag,
			}

			return mapper(client.UpdateFolderIamPolicy(ctx, resourceId).UpdateFolderIamPolicyPayload(payload).Execute())
		},
	}
}
