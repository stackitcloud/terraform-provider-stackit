package instance_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	"go.uber.org/mock/gomock"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/testutils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
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
	tfId := utils.BuildInternalTerraformId(projectId.String(), region, instanceId.String())
	bucketName := "bucket"
	deletetExpRetention := "1m"

	updateResp := &modelexperiments.PartialUpdateInstanceResponse{
		Instance: modelexperiments.Instance{
			DeletedExperimentRetention: &deletetExpRetention,
			BucketName:                 &bucketName,
			Description:                &descriptionUpdated,
			Name:                       instanceNameUpdated,
			Region:                     &region,
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
		Id:                         tfId,
		ProjectId:                  types.StringValue(projectId.String()),
		InstanceId:                 types.StringValue(instanceId.String()),
		Name:                       types.StringValue(instanceName),
		Region:                     types.StringValue(region),
		Description:                types.StringValue(description),
		Labels:                     types.MapNull(types.StringType),
		DeletedExperimentRetention: types.StringValue(deletetExpRetention),
		BucketName:                 types.StringValue(bucketName),
		Url:                        types.StringValue(url),
	}

	plannedState := instance.Model{
		ProjectId:   types.StringValue(projectId.String()),
		Region:      types.StringValue(region),
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

	if tfId != finalState.Id {
		t.Fatalf("expected %v, got %v", tfId.String(), finalState.Id.ValueString())
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
	if url != finalState.Url.ValueString() {
		t.Fatalf("expected %v, got %v", url, finalState.Url.ValueString())
	}
	if region != finalState.Region.ValueString() {
		t.Fatalf("expected %v, got %v", region, finalState.Region.ValueString())
	}
	if bucketName != finalState.BucketName.ValueString() {
		t.Fatalf("expected %v, got %v", bucketName, finalState.BucketName.ValueString())
	}
	if deletetExpRetention != finalState.DeletedExperimentRetention.ValueString() {
		t.Fatalf("expected %v, got %v", deletetExpRetention, finalState.DeletedExperimentRetention.ValueString())
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
	url := "url"
	tfId := utils.BuildInternalTerraformId(projectId.String(), region, instanceId.String())
	bucketName := "bucket"
	deletetExpRetention := "1m"

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
		Id:                         tfId,
		ProjectId:                  types.StringValue(projectId.String()),
		InstanceId:                 types.StringValue(instanceId.String()),
		Name:                       types.StringValue(instanceName),
		Region:                     types.StringValue(region),
		Description:                types.StringValue(description),
		Labels:                     types.MapNull(types.StringType),
		DeletedExperimentRetention: types.StringValue(deletetExpRetention),
		BucketName:                 types.StringValue(bucketName),
		Url:                        types.StringValue(url),
	}

	plannedState := instance.Model{
		ProjectId:   types.StringValue(projectId.String()),
		Region:      types.StringValue(region),
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

	if tfId != finalState.Id {
		t.Fatalf("expected %v, got %v", tfId.String(), finalState.Id.ValueString())
	}
	if instanceId.String() != finalState.InstanceId.ValueString() {
		t.Fatalf("expected %v, got %v", instanceId.String(), finalState.InstanceId.ValueString())
	}
	if projectId.String() != finalState.ProjectId.ValueString() {
		t.Fatalf("expected %v, got %v", projectId.String(), finalState.ProjectId.ValueString())
	}
	if instanceName != finalState.Name.ValueString() {
		t.Fatalf("expected %v, got %v", instanceName, finalState.Name.ValueString())
	}
	if description != finalState.Description.ValueString() {
		t.Fatalf("expected %v, got %v", description, finalState.Description.ValueString())
	}
	if url != finalState.Url.ValueString() {
		t.Fatalf("expected %v, got %v", url, finalState.Url.ValueString())
	}
	if region != finalState.Region.ValueString() {
		t.Fatalf("expected %v, got %v", region, finalState.Region.ValueString())
	}
	if bucketName != finalState.BucketName.ValueString() {
		t.Fatalf("expected %v, got %v", bucketName, finalState.BucketName.ValueString())
	}
	if deletetExpRetention != finalState.DeletedExperimentRetention.ValueString() {
		t.Fatalf("expected %v, got %v", deletetExpRetention, finalState.DeletedExperimentRetention.ValueString())
	}
}
