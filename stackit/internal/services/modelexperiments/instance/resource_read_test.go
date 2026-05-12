package instance_test

import (
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

func TestRead_Success(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	description := "description"
	region := "eu01"
	instanceId := uuid.New()
	url := "url"
	instanceNameUpdated := "updatedName"

	getResp := &modelexperiments.GetInstanceResponse{
		Instance: modelexperiments.Instance{
			DeletedExperimentRetention: new("1m"),
			Description:                &description,
			Name:                       instanceNameUpdated,
			Region:                     new("eu01"),
			Url:                        url,
			Id:                         instanceId.String(),
			State:                      "active",
			BucketName:                 new("bucket"),
		},
	}
	tc.MockInstanceCLient.EXPECT().GetInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceRequest{})
	tc.MockInstanceCLient.EXPECT().GetInstanceExecute(gomock.Any()).Return(getResp, nil)

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

	req := testutils.ReadRequest(tc.Ctx, schemaResp, state)
	resp := testutils.ReadResponse(schemaResp)

	instanceRes.Read(tc.Ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "Get should succeed, but got errors: %v", resp.Diagnostics.Errors())

	var refreshedState instance.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	require.False(t, diags.HasError(), "Failed to get state: %v", diags.Errors())

	require.Equal(t, instanceId.String(), refreshedState.InstanceId.ValueString())
	require.Equal(t, projectId.String(), refreshedState.ProjectId.ValueString())
	require.Equal(t, instanceNameUpdated, refreshedState.Name.ValueString())
	require.Equal(t, url, refreshedState.Url.ValueString())
	require.Equal(t, "active", refreshedState.State.ValueString())
	require.Equal(t, region, refreshedState.Region.ValueString())
	require.Equal(t, "bucket", refreshedState.BucketName.ValueString())
}

func TestRead_InstanceIdEmptyFailure(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	region := "eu01"

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
	}
	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, nil, providerData)

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	state := instance.Model{
		ProjectId:  types.StringValue(projectId.String()),
		InstanceId: types.StringValue(""),
		Region:     types.StringValue(region),
		Name:       types.StringValue(instanceName),
	}

	req := testutils.ReadRequest(tc.Ctx, schemaResp, state)
	resp := testutils.ReadResponse(schemaResp)

	instanceRes.Read(tc.Ctx, req, resp)
	require.True(t, resp.Diagnostics.HasError(), "Get should not succeed, but got no errors")

	var refreshedState *instance.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	require.False(t, diags.HasError(), "Failed to get state: %v", diags.Errors())
	require.Nil(t, refreshedState, "State not nil")
}

func TestRead_InstanceNotFound(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceId := uuid.New()
	instanceName := "test"
	region := "eu01"

	oapiErr := oapierror.GenericOpenAPIError{
		StatusCode: 404,
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

	req := testutils.ReadRequest(tc.Ctx, schemaResp, state)
	resp := testutils.ReadResponse(schemaResp)

	instanceRes.Read(tc.Ctx, req, resp)
	require.True(t, resp.Diagnostics.HasError(), "Get should not succeed, but got no errors")

	var refreshedState *instance.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	require.False(t, diags.HasError(), "Failed to get state: %v", diags.Errors())
	require.Nil(t, refreshedState, "State not nil")
}

func TestRead_GetRequestFailed(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	region := "eu01"
	instanceId := uuid.New()

	oapiErr := oapierror.GenericOpenAPIError{
		StatusCode: 400,
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

	req := testutils.ReadRequest(tc.Ctx, schemaResp, state)
	resp := testutils.ReadResponse(schemaResp)

	instanceRes.Read(tc.Ctx, req, resp)
	require.True(t, resp.Diagnostics.HasError(), "Get should not succeed")

	var refreshedState *instance.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	require.False(t, diags.HasError(), "Failed to get state: %v", diags.Errors())
	require.Nil(t, refreshedState)
}
