package instance_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	modelexperiments "github.com/stackitcloud/stackit-sdk-go/services/modelexperiments/v1api"
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
	"go.uber.org/mock/gomock"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/testutils"
)

func TestCreate_Success(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	description := "description"
	region := "eu01"
	instanceId := uuid.New()
	url := "url"

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
		Version:       "1.0.0",
	}

	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, tc.MockServiceEnablementClient, providerData)

	tc.MockServiceEnablementClient.EXPECT().EnableServiceRegional(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(serviceenablement.ApiEnableServiceRegionalRequest{
		ApiService: tc.MockServiceEnablementClient,
	})
	tc.MockServiceEnablementClient.EXPECT().EnableServiceRegionalExecute(gomock.Any()).Return(nil)

	serviceEnablementResp := &serviceenablement.ServiceStatus{
		State: serviceenablement.SERVICESTATUSSTATE_ENABLED.Ptr(),
	}
	tc.MockServiceEnablementClient.EXPECT().GetServiceStatusRegional(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(serviceenablement.ApiGetServiceStatusRegionalRequest{
		ApiService: tc.MockServiceEnablementClient,
	})
	tc.MockServiceEnablementClient.EXPECT().GetServiceStatusRegionalExecute(gomock.Any()).Return(serviceEnablementResp, nil)

	createResp := &modelexperiments.CreateInstanceResponse{
		Instance: modelexperiments.Instance{
			DeletedExperimentRetention: new("1m"),
			Description:                &description,
			Name:                       instanceName,
			Region:                     new("eu01"),
			Url:                        url,
			Id:                         instanceId.String(),
			State:                      "pending",
		},
	}
	tc.MockInstanceCLient.EXPECT().CreateInstance(gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().CreateInstanceExecute(gomock.Any()).Return(createResp, nil)

	getResp := &modelexperiments.GetInstanceResponse{
		Instance: modelexperiments.Instance{
			DeletedExperimentRetention: new("1m"),
			Description:                &description,
			Name:                       instanceName,
			Region:                     new("eu01"),
			Url:                        url,
			Id:                         instanceId.String(),
			State:                      "active",
			BucketName:                 new("bucket"),
		},
	}

	tc.MockInstanceCLient.EXPECT().GetInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().GetInstanceExecute(gomock.Any()).Return(getResp, nil)

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := testutils.CreateInstanceTestModel(projectId.String(), region, instanceName, description)
	req := testutils.CreateInstanceRequest(tc.Ctx, schemaResp, model)
	resp := testutils.CreateResponse(schemaResp)

	instanceRes.Create(tc.Ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create should succeed, but got errors: %v", resp.Diagnostics.Errors())
	}

	var stateAfterCreate instance.Model
	diags := resp.State.Get(tc.Ctx, &stateAfterCreate)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}

	// state should be created correctly
	if instanceId.String() != stateAfterCreate.InstanceId.ValueString() {
		t.Fatalf("expected %v, got %v", instanceId.String(), stateAfterCreate.InstanceId.ValueString())
	}
	if projectId.String() != stateAfterCreate.ProjectId.ValueString() {
		t.Fatalf("expected %v, got %v", projectId.String(), stateAfterCreate.ProjectId.ValueString())
	}
	if instanceName != stateAfterCreate.Name.ValueString() {
		t.Fatalf("expected %v, got %v", instanceName, stateAfterCreate.Name.ValueString())
	}
	if url != stateAfterCreate.Url.ValueString() {
		t.Fatalf("expected %v, got %v", url, stateAfterCreate.Url.ValueString())
	}
	if stateAfterCreate.State.ValueString() != "active" {
		t.Fatalf("expected %v, got %v", "active", stateAfterCreate.State.ValueString())
	}
	if region != stateAfterCreate.Region.ValueString() {
		t.Fatalf("expected %v, got %v", region, stateAfterCreate.Region.ValueString())
	}
	if stateAfterCreate.BucketName.ValueString() != "bucket" {
		t.Fatalf("expected %v, got %v", "bucket", stateAfterCreate.BucketName.ValueString())
	}
}

func TestCreate_ServiceEnablementFailure(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	description := "description"
	region := "eu01"

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
		Version:       "1.0.0",
	}

	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, tc.MockServiceEnablementClient, providerData)

	oapiErr := &oapierror.GenericOpenAPIError{
		StatusCode: http.StatusNotFound,
	}
	tc.MockServiceEnablementClient.EXPECT().EnableServiceRegional(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(serviceenablement.ApiEnableServiceRegionalRequest{
		ApiService: tc.MockServiceEnablementClient,
	})
	tc.MockServiceEnablementClient.EXPECT().EnableServiceRegionalExecute(gomock.Any()).Return(oapiErr)

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := testutils.CreateInstanceTestModel(projectId.String(), region, instanceName, description)
	req := testutils.CreateInstanceRequest(tc.Ctx, schemaResp, model)
	resp := testutils.CreateResponse(schemaResp)

	instanceRes.Create(tc.Ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("Create should not succeed, but got no errors")
	}

	// state should not be created
	var stateAfterCreate *instance.Model
	diags := resp.State.Get(tc.Ctx, &stateAfterCreate)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}
	if stateAfterCreate != nil {
		t.Fatalf("State not nil")
	}
}

func TestCreate_GetInstanceFailure(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	description := "description"
	region := "eu01"
	instanceId := uuid.New()
	url := "url"

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
		Version:       "1.0.0",
	}

	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, tc.MockServiceEnablementClient, providerData)

	tc.MockServiceEnablementClient.EXPECT().EnableServiceRegional(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(serviceenablement.ApiEnableServiceRegionalRequest{
		ApiService: tc.MockServiceEnablementClient,
	})
	tc.MockServiceEnablementClient.EXPECT().EnableServiceRegionalExecute(gomock.Any()).Return(nil)

	serviceEnablementResp := &serviceenablement.ServiceStatus{
		State: serviceenablement.SERVICESTATUSSTATE_ENABLED.Ptr(),
	}
	tc.MockServiceEnablementClient.EXPECT().GetServiceStatusRegional(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(serviceenablement.ApiGetServiceStatusRegionalRequest{
		ApiService: tc.MockServiceEnablementClient,
	})
	tc.MockServiceEnablementClient.EXPECT().GetServiceStatusRegionalExecute(gomock.Any()).Return(serviceEnablementResp, nil)

	createResp := &modelexperiments.CreateInstanceResponse{
		Instance: modelexperiments.Instance{
			DeletedExperimentRetention: new("1m"),
			Description:                &description,
			Name:                       instanceName,
			Region:                     new("eu01"),
			Url:                        url,
			Id:                         instanceId.String(),
			State:                      "pending",
		},
	}
	tc.MockInstanceCLient.EXPECT().CreateInstance(gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().CreateInstanceExecute(gomock.Any()).Return(createResp, nil)

	tc.MockInstanceCLient.EXPECT().GetInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().GetInstanceExecute(gomock.Any()).Return(nil, fmt.Errorf("server error"))

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := testutils.CreateInstanceTestModel(projectId.String(), region, instanceName, description)
	req := testutils.CreateInstanceRequest(tc.Ctx, schemaResp, model)
	resp := testutils.CreateResponse(schemaResp)

	instanceRes.Create(tc.Ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("Create should succeed with errors")
	}

	var stateAfterCreate instance.Model
	diags := resp.State.Get(tc.Ctx, &stateAfterCreate)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}

	// state should be created even if get request failed
	if instanceId.String() != stateAfterCreate.InstanceId.ValueString() {
		t.Fatalf("expected %v, got %v", instanceId.String(), stateAfterCreate.InstanceId.ValueString())
	}
	if projectId.String() != stateAfterCreate.ProjectId.ValueString() {
		t.Fatalf("expected %v, got %v", projectId.String(), stateAfterCreate.ProjectId.ValueString())
	}
	if instanceName != stateAfterCreate.Name.ValueString() {
		t.Fatalf("expected %v, got %v", instanceName, stateAfterCreate.Name.ValueString())
	}
	if url != stateAfterCreate.Url.ValueString() {
		t.Fatalf("expected %v, got %v", url, stateAfterCreate.Url.ValueString())
	}
	if stateAfterCreate.State.ValueString() != string(createResp.Instance.State) {
		t.Fatalf("expected %v, got %v", "pending", stateAfterCreate.State.ValueString())
	}
	if region != stateAfterCreate.Region.ValueString() {
		t.Fatalf("expected %v, got %v", region, stateAfterCreate.Region.ValueString())
	}
	if stateAfterCreate.BucketName.ValueString() != "" {
		t.Fatalf("expected %v, got %v", "", stateAfterCreate.BucketName.ValueString())
	}
}

func TestCreate_InstanceCreateFailure(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	description := "description"
	region := "eu01"

	providerData := core.ProviderData{
		DefaultRegion: "eu01",
		Version:       "1.0.0",
	}

	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, tc.MockServiceEnablementClient, providerData)

	tc.MockServiceEnablementClient.EXPECT().EnableServiceRegional(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(serviceenablement.ApiEnableServiceRegionalRequest{
		ApiService: tc.MockServiceEnablementClient,
	})
	tc.MockServiceEnablementClient.EXPECT().EnableServiceRegionalExecute(gomock.Any()).Return(nil)

	serviceEnablementResp := &serviceenablement.ServiceStatus{
		State: serviceenablement.SERVICESTATUSSTATE_ENABLED.Ptr(),
	}
	tc.MockServiceEnablementClient.EXPECT().GetServiceStatusRegional(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(serviceenablement.ApiGetServiceStatusRegionalRequest{
		ApiService: tc.MockServiceEnablementClient,
	})
	tc.MockServiceEnablementClient.EXPECT().GetServiceStatusRegionalExecute(gomock.Any()).Return(serviceEnablementResp, nil)

	tc.MockInstanceCLient.EXPECT().CreateInstance(gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceRequest{
		ApiService: tc.MockInstanceCLient,
	})
	tc.MockInstanceCLient.EXPECT().CreateInstanceExecute(gomock.Any()).Return(nil, fmt.Errorf("server error"))

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := testutils.CreateInstanceTestModel(projectId.String(), region, instanceName, description)
	req := testutils.CreateInstanceRequest(tc.Ctx, schemaResp, model)
	resp := testutils.CreateResponse(schemaResp)

	instanceRes.Create(tc.Ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("Create should not succeed, but got no errors")
	}

	// no state should be created
	var stateAfterCreate *instance.Model
	diags := resp.State.Get(tc.Ctx, &stateAfterCreate)
	if diags.HasError() {
		t.Fatalf("Failed to get state: %v", diags.Errors())
	}
	if stateAfterCreate != nil {
		t.Fatalf("State not nil")
	}
}
