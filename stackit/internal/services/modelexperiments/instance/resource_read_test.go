package instance_test

import (
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

	req := testutils.ReadInstanceRequest(tc.Ctx, schemaResp, state)
	resp := testutils.ReadInstanceResponse(tc.Ctx, schemaResp, nil)

	instanceRes.Read(tc.Ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Get should succeed, but got errors: %v", resp.Diagnostics.Errors())
	}

	var refreshedState instance.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}

	// state should be written according to GetInstance Response
	if instanceId.String() != refreshedState.InstanceId.ValueString() {
		t.Fatalf("expected %v, got %v", instanceId.String(), refreshedState.InstanceId.ValueString())
	}
	if projectId.String() != refreshedState.ProjectId.ValueString() {
		t.Fatalf("expected %v, got %v", projectId.String(), refreshedState.ProjectId.ValueString())
	}
	if instanceNameUpdated != refreshedState.Name.ValueString() {
		t.Fatalf("expected %v, got %v", instanceNameUpdated, refreshedState.Name.ValueString())
	}
	if url != refreshedState.Url.ValueString() {
		t.Fatalf("expected %v, got %v", url, refreshedState.Url.ValueString())
	}
	if refreshedState.State.ValueString() != "active" {
		t.Fatalf("expected %v, got %v", "active", refreshedState.State.ValueString())
	}
	if region != refreshedState.Region.ValueString() {
		t.Fatalf("expected %v, got %v", region, refreshedState.Region.ValueString())
	}
	if refreshedState.BucketName.ValueString() != "bucket" {
		t.Fatalf("expected %v, got %v", "bucket", refreshedState.BucketName.ValueString())
	}
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

	req := testutils.ReadInstanceRequest(tc.Ctx, schemaResp, state)
	resp := testutils.ReadInstanceResponse(tc.Ctx, schemaResp, &state)

	instanceRes.Read(tc.Ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("Get should not succeed, but got no errors")
	}

	// state should be removed
	var refreshedState *instance.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}
	if refreshedState != nil {
		t.Fatalf("State not nil")
	}
}

func TestRead_InstanceNotFound(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceId := uuid.New()
	instanceName := "test"
	region := "eu01"

	oapiErr := &oapierror.GenericOpenAPIError{
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

	req := testutils.ReadInstanceRequest(tc.Ctx, schemaResp, state)
	resp := testutils.ReadInstanceResponse(tc.Ctx, schemaResp, &state)

	instanceRes.Read(tc.Ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Get should succeed, but got errors: %v", resp.Diagnostics.Errors())
	}

	// state should be removed
	var refreshedState *instance.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}
	if refreshedState != nil {
		t.Fatalf("State not nil")
	}
}

func TestRead_GetRequestFailed(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	region := "eu01"
	instanceId := uuid.New()

	oapiErr := &oapierror.GenericOpenAPIError{
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

	req := testutils.ReadInstanceRequest(tc.Ctx, schemaResp, state)
	resp := testutils.ReadInstanceResponse(tc.Ctx, schemaResp, nil)

	instanceRes.Read(tc.Ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("Get should not succeed")
	}

	// state should not be set
	var refreshedState *instance.Model
	diags := resp.State.Get(tc.Ctx, &refreshedState)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}
	if refreshedState != nil {
		t.Fatalf("expected nil, got %v", refreshedState)
	}
}
