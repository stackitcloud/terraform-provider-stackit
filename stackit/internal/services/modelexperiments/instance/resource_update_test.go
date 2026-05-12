package instance_test

import (
	"fmt"
	"testing"

	modelexperiments "dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/modelexperiments/v1api"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/testutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUpdate_Success(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	instanceNameUpdated := "update name"
	description := "description"
	descriptionUpdated := "description updated"
	region := "eu01"
	instanceId := uuid.New()
	url := "url"

	updateResp := modelexperiments.PartialUpdateInstanceResponse{
		Instance: modelexperiments.Instance{
			DeletedExperimentRetention: new("1m"),
			Description:                &descriptionUpdated,
			Name:                       instanceNameUpdated,
			Region:                     new("eu01"),
			Url:                        url,
			Id:                         instanceId.String(),
			State:                      "active",
		},
	}

	tc.MockInstanceCLient.EXPECT().PartialUpdateInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiPartialUpdateInstanceRequest{})
	tc.MockInstanceCLient.EXPECT().PartialUpdateInstanceExecute(gomock.Any()).Return(updateResp, nil)

	providerData := core.ProviderData{
		DefaultRegion: region,
	}
	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, nil, providerData)

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	currentState := instance.Model{
		Id:          types.StringValue(fmt.Sprintf("%s,%s", projectId, instanceId)),
		ProjectId:   types.StringValue(projectId.String()),
		InstanceId:  types.StringValue(instanceId.String()),
		Name:        types.StringValue(instanceName),
		Description: types.StringValue(description),
	}

	plannedState := instance.Model{
		Id:          types.StringValue(fmt.Sprintf("%s,%s", projectId, instanceId)),
		ProjectId:   types.StringValue(projectId.String()),
		Region:      types.StringValue(region),
		InstanceId:  types.StringValue(instanceId.String()),
		Name:        types.StringValue(instanceNameUpdated),
		Description: types.StringValue(descriptionUpdated),
	}

	req := testutils.UpdateRequest(tc.Ctx, schemaResp, currentState, plannedState)
	resp := testutils.UpdateResponse(tc.Ctx, schemaResp, nil)

	// Execute Update
	instanceRes.Update(tc.Ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "Update should succeed, but got errors: %v", resp.Diagnostics.Errors())

	// Extract final state
	var finalState instance.Model
	diags := resp.State.Get(tc.Ctx, &finalState)
	require.False(t, diags.HasError(), "Failed to get state: %v", diags.Errors())

	// Verify all fields match the updated values from GetInstance
	require.Equal(t, instanceId, finalState.InstanceId.ValueString())
	require.Equal(t, projectId, finalState.ProjectId.ValueString())
	require.Equal(t, instanceNameUpdated, finalState.Name.ValueString())
	require.Equal(t, descriptionUpdated, finalState.Description.ValueString())
	require.Equal(t, "active", finalState.State.ValueString())
	require.Equal(t, url, finalState.Url.ValueString())
	require.Equal(t, region, finalState.Region.ValueString())
}
