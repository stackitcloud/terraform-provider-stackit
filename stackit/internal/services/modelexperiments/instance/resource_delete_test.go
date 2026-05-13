package instance_test

import (
	"net/http"
	"testing"

	modelexperiments "dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/modelexperiments/v1api"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/testutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDelete_Success(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	region := "eu01"
	instanceId := uuid.New()

	tc.MockInstanceCLient.EXPECT().DeleteInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiDeleteInstanceRequest{})
	tc.MockInstanceCLient.EXPECT().DeleteInstanceExecute(gomock.Any()).Return(nil, nil)

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusNotFound,
	}
	tc.MockInstanceCLient.EXPECT().GetInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceRequest{})
	tc.MockInstanceCLient.EXPECT().GetInstanceExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, nil, providerData)

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	state := instance.Model{
		ProjectId:  types.StringValue(projectId.String()),
		InstanceId: types.StringValue(instanceId.String()),
		Region:     types.StringValue(region),
		Name:       types.StringValue(instanceName),
		Labels:     types.MapNull(types.StringType),
	}

	req := testutils.DeleteInstanceRequest(tc.Ctx, schemaResp, state)
	resp := testutils.DeleteInstanceResponse(tc.Ctx, schemaResp, &state)

	instanceRes.Delete(tc.Ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "Delete should succeed, but got errors: %v", resp.Diagnostics.Errors())
}

func TestDelete_DeleteInstanceFailed(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	region := "eu01"
	instanceId := uuid.New()

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusInternalServerError,
	}
	tc.MockInstanceCLient.EXPECT().DeleteInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiDeleteInstanceRequest{})
	tc.MockInstanceCLient.EXPECT().DeleteInstanceExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, nil, providerData)

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	state := instance.Model{
		ProjectId:  types.StringValue(projectId.String()),
		InstanceId: types.StringValue(instanceId.String()),
		Region:     types.StringValue(region),
		Name:       types.StringValue(instanceName),
		Labels:     types.MapNull(types.StringType),
	}

	req := testutils.DeleteInstanceRequest(tc.Ctx, schemaResp, state)
	resp := testutils.DeleteInstanceResponse(tc.Ctx, schemaResp, &state)

	instanceRes.Delete(tc.Ctx, req, resp)
	require.True(t, resp.Diagnostics.HasError(), "Delete should not succeed, but got no errors")

	var finalState instance.Model
	diags := resp.State.Get(tc.Ctx, &finalState)
	require.False(t, diags.HasError(), "Failed to get state: %v", diags.Errors())

	require.Equal(t, instanceId.String(), finalState.InstanceId.ValueString(), "state should not have been deleted")
}

func TestDelete_InstanceAlreadyDeleted(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	region := "eu01"
	instanceId := uuid.New()

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusNotFound,
	}
	tc.MockInstanceCLient.EXPECT().DeleteInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiDeleteInstanceRequest{})
	tc.MockInstanceCLient.EXPECT().DeleteInstanceExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, nil, providerData)

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	state := instance.Model{
		ProjectId:  types.StringValue(projectId.String()),
		InstanceId: types.StringValue(instanceId.String()),
		Region:     types.StringValue(region),
		Name:       types.StringValue(instanceName),
		Labels:     types.MapNull(types.StringType),
	}

	req := testutils.DeleteInstanceRequest(tc.Ctx, schemaResp, state)
	resp := testutils.DeleteInstanceResponse(tc.Ctx, schemaResp, &state)

	instanceRes.Delete(tc.Ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "Delete should succeed, but got errors: %v", resp.Diagnostics.Errors())

	var finalState *instance.Model
	diags := resp.State.Get(tc.Ctx, &finalState)
	require.False(t, diags.HasError(), "Failed to get state: %v", diags.Errors())
	require.Nil(t, finalState, "state should have been deleted")
}

func TestDelete_GetInstanceFailed(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	region := "eu01"
	instanceId := uuid.New()

	tc.MockInstanceCLient.EXPECT().DeleteInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiDeleteInstanceRequest{})
	tc.MockInstanceCLient.EXPECT().DeleteInstanceExecute(gomock.Any()).Return(nil, nil)

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusInternalServerError,
	}
	tc.MockInstanceCLient.EXPECT().GetInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceRequest{})
	tc.MockInstanceCLient.EXPECT().GetInstanceExecute(gomock.Any()).Return(nil, oapiErr)

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, nil, providerData)

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	state := instance.Model{
		ProjectId:  types.StringValue(projectId.String()),
		InstanceId: types.StringValue(instanceId.String()),
		Region:     types.StringValue(region),
		Name:       types.StringValue(instanceName),
		Labels:     types.MapNull(types.StringType),
	}

	req := testutils.DeleteInstanceRequest(tc.Ctx, schemaResp, state)
	resp := testutils.DeleteInstanceResponse(tc.Ctx, schemaResp, &state)

	instanceRes.Delete(tc.Ctx, req, resp)
	require.True(t, resp.Diagnostics.HasError(), "Delete should not succeed, but got no errors")

	var finalState instance.Model
	diags := resp.State.Get(tc.Ctx, &finalState)
	require.False(t, diags.HasError(), "Failed to get state: %v", diags.Errors())

	require.Equal(t, instanceId.String(), state.InstanceId.ValueString(), "state should not have been deleted")
}
