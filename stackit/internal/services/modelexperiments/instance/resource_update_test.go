package instance_test

import (
	"fmt"
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

	updateResp := &modelexperiments.PartialUpdateInstanceResponse{
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

	tc.MockInstanceCLient.EXPECT().PartialUpdateInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiPartialUpdateInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
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
		Region:      types.StringValue(region),
		Description: types.StringValue(description),
		Labels:      types.MapNull(types.StringType),
	}

	plannedState := instance.Model{
		Id:          types.StringValue(fmt.Sprintf("%s,%s", projectId, instanceId)),
		ProjectId:   types.StringValue(projectId.String()),
		Region:      types.StringValue(region),
		InstanceId:  types.StringValue(instanceId.String()),
		Name:        types.StringValue(instanceNameUpdated),
		Description: types.StringValue(descriptionUpdated),
		Labels:      types.MapNull(types.StringType),
	}

	req := testutils.UpdateInstanceRequest(tc.Ctx, schemaResp, currentState, plannedState)
	resp := testutils.UpdateInstanceResponse(tc.Ctx, schemaResp, &currentState)

	// Execute Update
	instanceRes.Update(tc.Ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update should succeed, but got errors: %v", resp.Diagnostics.Errors())
	}

	//  state should be updated
	var finalState instance.Model
	diags := resp.State.Get(tc.Ctx, &finalState)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}

	if instanceId.String() != finalState.InstanceId.ValueString() {
		t.Fatalf("expected %v, got %v", instanceId.String(), finalState.InstanceId.ValueString())
	}
	if projectId.String() != finalState.ProjectId.ValueString() {
		t.Fatalf("expected %v, got %v", projectId.String(), finalState.ProjectId.ValueString())
	}
	if instanceNameUpdated != finalState.Name.ValueString() {
		t.Fatalf("expected %v, got %v", instanceNameUpdated, finalState.Name.ValueString())
	}
	if descriptionUpdated != finalState.Description.ValueString() {
		t.Fatalf("expected %v, got %v", descriptionUpdated, finalState.Description.ValueString())
	}
	if finalState.State.ValueString() != "active" {
		t.Fatalf("expected %v, got %v", "active", finalState.State.ValueString())
	}
	if url != finalState.Url.ValueString() {
		t.Fatalf("expected %v, got %v", url, finalState.Url.ValueString())
	}
	if region != finalState.Region.ValueString() {
		t.Fatalf("expected %v, got %v", region, finalState.Region.ValueString())
	}
}

func TestUpdate_InstanceNotFound(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	instanceNameUpdated := "update name"
	description := "description"
	descriptionUpdated := "description updated"
	region := "eu01"
	instanceId := uuid.New()

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: 404,
	}
	tc.MockInstanceCLient.EXPECT().PartialUpdateInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiPartialUpdateInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().PartialUpdateInstanceExecute(gomock.Any()).Return(nil, oapiErr)

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
		Labels:      types.MapNull(types.StringType),
	}

	plannedState := instance.Model{
		Id:          types.StringValue(fmt.Sprintf("%s,%s", projectId, instanceId)),
		ProjectId:   types.StringValue(projectId.String()),
		Region:      types.StringValue(region),
		InstanceId:  types.StringValue(instanceId.String()),
		Name:        types.StringValue(instanceNameUpdated),
		Description: types.StringValue(descriptionUpdated),
		Labels:      types.MapNull(types.StringType),
	}

	req := testutils.UpdateInstanceRequest(tc.Ctx, schemaResp, currentState, plannedState)
	resp := testutils.UpdateInstanceResponse(tc.Ctx, schemaResp, &currentState)

	// Execute Update
	instanceRes.Update(tc.Ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update should succeed, but got errors: %v", resp.Diagnostics.Errors())
	}

	// state should be deleted
	var finalState *instance.Model
	diags := resp.State.Get(tc.Ctx, &finalState)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}
	if finalState != nil {
		t.Fatalf("State should not be written")
	}
}

func TestUpdate_InstanceUpdateError(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	instanceNameUpdated := "update name"
	description := "description"
	descriptionUpdated := "description updated"
	region := "eu01"
	instanceId := uuid.New()

	tc.MockInstanceCLient.EXPECT().PartialUpdateInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiPartialUpdateInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().PartialUpdateInstanceExecute(gomock.Any()).Return(nil, fmt.Errorf("server error"))

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
		Region:      types.StringValue(region),
		Name:        types.StringValue(instanceName),
		Description: types.StringValue(description),
		Labels:      types.MapNull(types.StringType),
	}

	plannedState := instance.Model{
		Id:          types.StringValue(fmt.Sprintf("%s,%s", projectId, instanceId)),
		ProjectId:   types.StringValue(projectId.String()),
		Region:      types.StringValue(region),
		InstanceId:  types.StringValue(instanceId.String()),
		Name:        types.StringValue(instanceNameUpdated),
		Description: types.StringValue(descriptionUpdated),
		Labels:      types.MapNull(types.StringType),
	}

	req := testutils.UpdateInstanceRequest(tc.Ctx, schemaResp, currentState, plannedState)
	resp := testutils.UpdateInstanceResponse(tc.Ctx, schemaResp, &currentState)

	// Execute Update
	instanceRes.Update(tc.Ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("Update should not succeed, but got no errors")
	}

	// state should not be updated
	var finalState instance.Model
	diags := resp.State.Get(tc.Ctx, &finalState)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}

	if description != finalState.Description.ValueString() {
		t.Fatalf("value should not have changed - expected %v, got %v", description, finalState.Description.ValueString())
	}
	if instanceName != finalState.Name.ValueString() {
		t.Fatalf("value should not have changed - expected %v, got %v", instanceName, finalState.Name.ValueString())
	}
	if instanceId.String() != finalState.InstanceId.ValueString() {
		t.Fatalf("value should not have changed - expected %v, got %v", instanceId.String(), finalState.InstanceId.ValueString())
	}
	if region != finalState.Region.ValueString() {
		t.Fatalf("value should not have changed - expected %v, got %v", region, finalState.Region.ValueString())
	}
	if projectId.String() != finalState.ProjectId.ValueString() {
		t.Fatalf("value should not have changed - expected %v, got %v", projectId.String(), finalState.ProjectId.ValueString())
	}
	if fmt.Sprintf("%s,%s", projectId, instanceId) != finalState.Id.ValueString() {
		t.Fatalf("value should not have changed - expected %v, got %v", fmt.Sprintf("%s,%s", projectId, instanceId), finalState.Id.ValueString())
	}
}
