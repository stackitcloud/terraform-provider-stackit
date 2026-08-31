package instance_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	"go.uber.org/mock/gomock"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/testutils"
)

// NOTE: These tests will be refactored.
// Please DO NOT use this file as a pattern or reference for writing new tests.
func TestDelete_Success(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	region := "eu01"
	instanceId := uuid.New()

	tc.MockInstanceCLient.EXPECT().DeleteInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiDeleteInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().DeleteInstanceExecute(gomock.Any()).Return(nil, nil)

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusNotFound,
	}
	tc.MockInstanceCLient.EXPECT().GetInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
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
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete should succeed, but got errors: %v", resp.Diagnostics.Errors())
	}
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
	tc.MockInstanceCLient.EXPECT().DeleteInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiDeleteInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
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
	if !resp.Diagnostics.HasError() {
		t.Fatalf("Delete should not succeed, but got no errors")
	}

	// state should not be removed
	var finalState instance.Model
	diags := resp.State.Get(tc.Ctx, &finalState)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}

	if instanceId.String() != finalState.InstanceId.ValueString() {
		t.Fatalf("state should not have been deleted - expected %v, got %v", instanceId.String(), finalState.InstanceId.ValueString())
	}
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
	tc.MockInstanceCLient.EXPECT().DeleteInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiDeleteInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
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
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete should succeed, but got errors: %v", resp.Diagnostics.Errors())
	}

	// state should be removed
	var finalState *instance.Model
	diags := resp.State.Get(tc.Ctx, &finalState)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}
	if finalState != nil {
		t.Fatalf("state should have been deleted - got %v", finalState)
	}
}

func TestDelete_GetInstanceFailed(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	region := "eu01"
	instanceId := uuid.New()

	tc.MockInstanceCLient.EXPECT().DeleteInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiDeleteInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().DeleteInstanceExecute(gomock.Any()).Return(nil, nil)

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusInternalServerError,
	}
	tc.MockInstanceCLient.EXPECT().GetInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
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
	if !resp.Diagnostics.HasError() {
		t.Fatalf("Delete should not succeed, but got no errors")
	}

	// state should not be removed
	var finalState instance.Model
	diags := resp.State.Get(tc.Ctx, &finalState)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}

	if instanceId.String() != finalState.InstanceId.ValueString() {
		t.Fatalf("state should not have been deleted - expected %v, got %v", instanceId.String(), state.InstanceId.ValueString())
	}
}
