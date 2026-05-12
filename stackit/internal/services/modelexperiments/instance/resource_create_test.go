package instance_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/stackitcloud/stackit-sdk-go/core/config"

	modelexperiments "dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/modelexperiments/v1api"
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/testutils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	modelServingServiceId = "cloud.stackit.model-serving"
)

func TestCreate_Success(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	description := "description"
	region := "eu01"
	instanceId := uuid.New()

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == fmt.Sprintf("/v2/projects/%s/regions/%s/services/%s", projectId, region, modelServingServiceId) {
					if r.Method == http.MethodGet {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"state":"ENABLED","scope":"PUBLIC","serviceId":"cloud.stackit.model-serving"}`))
					}
					if r.Method == http.MethodPost {
						w.WriteHeader(http.StatusAccepted)
					}
				}
			},
		),
	)
	defer server.Close()

	providerData := core.ProviderData{
		DefaultRegion:                   "eu01",
		Version:                         "1.0.0",
		ServiceEnablementCustomEndpoint: server.URL,
	}

	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithoutAuthentication(),
		config.WithHTTPClient(server.Client()),
		utils.UserAgentConfigOption(providerData.Version),
		config.WithEndpoint(providerData.ServiceEnablementCustomEndpoint),
	}

	apiClient, err := serviceenablement.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		fmt.Println(err)
	}

	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, apiClient.DefaultAPI, providerData)

	url := "url"

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
	tc.MockInstanceCLient.EXPECT().CreateInstance(gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceRequest{})
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
	tc.MockInstanceCLient.EXPECT().GetInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceRequest{})
	tc.MockInstanceCLient.EXPECT().GetInstanceExecute(gomock.Any()).Return(getResp, nil)

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := testutils.CreateInstanceTestModel(projectId.String(), region, instanceName, description)
	req := testutils.CreateInstanceRequest(tc.Ctx, schemaResp, model)
	resp := testutils.CreateResponse(schemaResp)

	instanceRes.Create(tc.Ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "Create should succeed, but got errors: %v", resp.Diagnostics.Errors())

	var stateAfterCreate instance.Model
	diags := resp.State.Get(tc.Ctx, &stateAfterCreate)
	require.False(t, diags.HasError(), "Failed to get state: %v", diags.Errors())

	require.Equal(t, instanceId.String(), stateAfterCreate.InstanceId.ValueString())
	require.Equal(t, projectId.String(), stateAfterCreate.ProjectId.ValueString())
	require.Equal(t, instanceName, stateAfterCreate.Name.ValueString())
	require.Equal(t, url, stateAfterCreate.Url.ValueString())
	require.Equal(t, "active", stateAfterCreate.State.ValueString())
	require.Equal(t, region, stateAfterCreate.Region.ValueString())
	require.Equal(t, "bucket", stateAfterCreate.BucketName.ValueString())
}

func TestCreate_ServiceEnablementFailure(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	description := "description"
	region := "eu01"

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == fmt.Sprintf("/v2/projects/%s/regions/%s/services/%s", projectId, region, modelServingServiceId) {
					if r.Method == http.MethodGet {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusNotFound)
					}
					if r.Method == http.MethodPost {
						w.WriteHeader(http.StatusNotFound)
					}
				}
			},
		),
	)
	defer server.Close()

	providerData := core.ProviderData{
		DefaultRegion:                   "eu01",
		Version:                         "1.0.0",
		ServiceEnablementCustomEndpoint: server.URL,
	}

	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithoutAuthentication(),
		config.WithHTTPClient(server.Client()),
		utils.UserAgentConfigOption(providerData.Version),
		config.WithEndpoint(providerData.ServiceEnablementCustomEndpoint),
	}

	apiClient, err := serviceenablement.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		fmt.Println(err)
	}

	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, apiClient.DefaultAPI, providerData)

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := testutils.CreateInstanceTestModel(projectId.String(), region, instanceName, description)
	req := testutils.CreateInstanceRequest(tc.Ctx, schemaResp, model)
	resp := testutils.CreateResponse(schemaResp)

	instanceRes.Create(tc.Ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError(), "Create should not succeed, but got no errors")

	var stateAfterCreate *instance.Model
	diags := resp.State.Get(tc.Ctx, &stateAfterCreate)
	require.False(t, diags.HasError(), "Failed to get state: %v", diags.Errors())
	require.Nil(t, stateAfterCreate, "State not nil")
}

func TestCreate_GetInstanceFailure(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	description := "description"
	region := "eu01"
	instanceId := uuid.New()

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == fmt.Sprintf("/v2/projects/%s/regions/%s/services/%s", projectId, region, modelServingServiceId) {
					if r.Method == http.MethodGet {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"state":"ENABLED","scope":"PUBLIC","serviceId":"cloud.stackit.model-serving"}`))
					}
					if r.Method == http.MethodPost {
						w.WriteHeader(http.StatusAccepted)
					}
				}
			},
		),
	)
	defer server.Close()

	providerData := core.ProviderData{
		DefaultRegion:                   "eu01",
		Version:                         "1.0.0",
		ServiceEnablementCustomEndpoint: server.URL,
	}

	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithoutAuthentication(),
		config.WithHTTPClient(server.Client()),
		utils.UserAgentConfigOption(providerData.Version),
		config.WithEndpoint(providerData.ServiceEnablementCustomEndpoint),
	}

	apiClient, err := serviceenablement.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		fmt.Println(err)
	}

	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, apiClient.DefaultAPI, providerData)

	url := "url"

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
	tc.MockInstanceCLient.EXPECT().CreateInstance(gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceRequest{})
	tc.MockInstanceCLient.EXPECT().CreateInstanceExecute(gomock.Any()).Return(createResp, nil)

	tc.MockInstanceCLient.EXPECT().GetInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceRequest{})
	tc.MockInstanceCLient.EXPECT().GetInstanceExecute(gomock.Any()).Return(nil, fmt.Errorf("server error"))

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := testutils.CreateInstanceTestModel(projectId.String(), region, instanceName, description)
	req := testutils.CreateInstanceRequest(tc.Ctx, schemaResp, model)
	resp := testutils.CreateResponse(schemaResp)

	instanceRes.Create(tc.Ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError(), "Create should succeed with errors")

	var stateAfterCreate instance.Model
	diags := resp.State.Get(tc.Ctx, &stateAfterCreate)
	require.False(t, diags.HasError(), "Failed to get state: %v", diags.Errors())

	require.Equal(t, instanceId.String(), stateAfterCreate.InstanceId.ValueString())
	require.Equal(t, projectId.String(), stateAfterCreate.ProjectId.ValueString())
	require.Equal(t, instanceName, stateAfterCreate.Name.ValueString())
	require.Equal(t, url, stateAfterCreate.Url.ValueString())
	require.Equal(t, "unknown", stateAfterCreate.State.ValueString())
	require.Equal(t, region, stateAfterCreate.Region.ValueString())
	require.Equal(t, "", stateAfterCreate.BucketName.ValueString())
}

func TestCreate_InstanceCreateFailure(t *testing.T) {
	tc := testutils.NewTestContext(t)

	projectId := uuid.New()
	instanceName := "test"
	description := "description"
	region := "eu01"

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == fmt.Sprintf("/v2/projects/%s/regions/%s/services/%s", projectId, region, modelServingServiceId) {
					if r.Method == http.MethodGet {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"state":"ENABLED","scope":"PUBLIC","serviceId":"cloud.stackit.model-serving"}`))
					}
					if r.Method == http.MethodPost {
						w.WriteHeader(http.StatusAccepted)
					}
				}
			},
		),
	)
	defer server.Close()

	providerData := core.ProviderData{
		DefaultRegion:                   "eu01",
		Version:                         "1.0.0",
		ServiceEnablementCustomEndpoint: server.URL,
	}

	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithoutAuthentication(),
		config.WithHTTPClient(server.Client()),
		utils.UserAgentConfigOption(providerData.Version),
		config.WithEndpoint(providerData.ServiceEnablementCustomEndpoint),
	}

	apiClient, err := serviceenablement.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		fmt.Println(err)
	}

	instanceRes := instance.NewInstanceResource(tc.MockInstanceCLient, apiClient.DefaultAPI, providerData)

	tc.MockInstanceCLient.EXPECT().CreateInstance(gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceRequest{})
	tc.MockInstanceCLient.EXPECT().CreateInstanceExecute(gomock.Any()).Return(nil, fmt.Errorf("server error"))

	schemaResp := resource.SchemaResponse{}
	instanceRes.Schema(tc.Ctx, resource.SchemaRequest{}, &schemaResp)

	model := testutils.CreateInstanceTestModel(projectId.String(), region, instanceName, description)
	req := testutils.CreateInstanceRequest(tc.Ctx, schemaResp, model)
	resp := testutils.CreateResponse(schemaResp)

	instanceRes.Create(tc.Ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError(), "Create should not succeed, but got no errors")

	var stateAfterCreate *instance.Model
	diags := resp.State.Get(tc.Ctx, &stateAfterCreate)
	require.False(t, diags.HasError(), "Failed to get state: %v", diags.Errors())
	require.Nil(t, stateAfterCreate, "State not nil")
}
