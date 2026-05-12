package instance_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stackitcloud/stackit-sdk-go/core/config"

	modelexperiments "dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/modelexperiments/v1api"
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance"
	mock_instance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance/mock"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/testutils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"go.uber.org/mock/gomock"
)

var _ = Describe("STACKIT ModelExperiments Resource", func() {
	const (
		modelServingServiceId = "cloud.stackit.model-serving"
	)

	var (
		ctx                context.Context
		ctrl               *gomock.Controller
		mockInstanceCLient *mock_instance.MockDefaultAPI

		projectId    uuid.UUID
		instanceName string
		description  string
		region       string
		instanceId   uuid.UUID
	)

	BeforeEach(func() {
		ctx = context.Background()
		ctrl = gomock.NewController(GinkgoT())
		mockInstanceCLient = mock_instance.NewMockDefaultAPI(ctrl)
		projectId = uuid.New()
		instanceName = "test"
		description = "description"
		region = "eu01"
		instanceId = uuid.New()
	})

	Describe("Instance resource", func() {
		When("everything works", func() {
			It("should create instance state correct", func() {

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

				instanceRes := instance.NewInstanceResource(mockInstanceCLient, apiClient.DefaultAPI, providerData)

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
				mockInstanceCLient.EXPECT().CreateInstance(gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceRequest{})
				mockInstanceCLient.EXPECT().CreateInstanceExecute(gomock.Any()).Return(createResp, nil)

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
				mockInstanceCLient.EXPECT().GetInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceRequest{})
				mockInstanceCLient.EXPECT().GetInstanceExecute(gomock.Any()).Return(getResp, nil)

				schemaResp := resource.SchemaResponse{}
				instanceRes.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

				model := testutils.CreateInstanceTestModel(projectId.String(), region, instanceName, description)
				req := testutils.CreateInstanceRequest(ctx, schemaResp, model)
				resp := testutils.CreateResponse(schemaResp)

				instanceRes.Create(ctx, req, resp)

				Expect(resp.Diagnostics.HasError()).To(BeFalse())

				var stateAfterCreate instance.Model
				diags := resp.State.Get(ctx, &stateAfterCreate)
				Expect(diags.HasError()).To(BeFalse())

				Expect(stateAfterCreate.InstanceId.ValueString()).To(Equal(instanceId.String()))
				Expect(stateAfterCreate.State.ValueString()).To(Equal("active"))
				Expect(stateAfterCreate.Url.ValueString()).To(Equal(url))
				Expect(stateAfterCreate.ProjectId.ValueString()).To(Equal(projectId.String()))
				Expect(stateAfterCreate.Region.ValueString()).To(Equal(region))
				Expect(stateAfterCreate.Name.ValueString()).To(Equal(instanceName))
				Expect(stateAfterCreate.Id).To(Equal(utils.BuildInternalTerraformId(projectId.String(), region, instanceId.String())))
			})
		})

		When("service enablement not working", func() {
			It("should not create state and throw error", func() {

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

				instanceRes := instance.NewInstanceResource(mockInstanceCLient, apiClient.DefaultAPI, providerData)

				schemaResp := resource.SchemaResponse{}
				instanceRes.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

				model := testutils.CreateInstanceTestModel(projectId.String(), region, instanceName, description)
				req := testutils.CreateInstanceRequest(ctx, schemaResp, model)
				resp := testutils.CreateResponse(schemaResp)

				instanceRes.Create(ctx, req, resp)

				Expect(resp.Diagnostics.HasError()).To(BeTrue())

				var stateAfterCreate *instance.Model
				diags := resp.State.Get(ctx, &stateAfterCreate)
				Expect(diags.HasError()).To(BeFalse())
				Expect(stateAfterCreate).To(BeNil())
			})
		})

		When("instance not found after creation", func() {
			It("should create instance state correct", func() {

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

				instanceRes := instance.NewInstanceResource(mockInstanceCLient, apiClient.DefaultAPI, providerData)

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
				mockInstanceCLient.EXPECT().CreateInstance(gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceRequest{})
				mockInstanceCLient.EXPECT().CreateInstanceExecute(gomock.Any()).Return(createResp, nil)

				mockInstanceCLient.EXPECT().GetInstance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiGetInstanceRequest{})
				mockInstanceCLient.EXPECT().GetInstanceExecute(gomock.Any()).Return(nil, fmt.Errorf("server error"))

				schemaResp := resource.SchemaResponse{}
				instanceRes.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

				model := testutils.CreateInstanceTestModel(projectId.String(), region, instanceName, description)
				req := testutils.CreateInstanceRequest(ctx, schemaResp, model)
				resp := testutils.CreateResponse(schemaResp)

				instanceRes.Create(ctx, req, resp)

				Expect(resp.Diagnostics.HasError()).To(BeTrue())

				var stateAfterCreate instance.Model
				diags := resp.State.Get(ctx, &stateAfterCreate)
				Expect(diags.HasError()).To(BeFalse())

				Expect(stateAfterCreate.InstanceId.ValueString()).To(Equal(instanceId.String()))
				Expect(stateAfterCreate.State.ValueString()).To(Equal("unknown"))
				Expect(stateAfterCreate.Url.ValueString()).To(Equal(url))
				Expect(stateAfterCreate.ProjectId.ValueString()).To(Equal(projectId.String()))
				Expect(stateAfterCreate.Region.ValueString()).To(Equal(region))
				Expect(stateAfterCreate.Name.ValueString()).To(Equal(instanceName))
				Expect(stateAfterCreate.Id).To(Equal(utils.BuildInternalTerraformId(projectId.String(), region, instanceId.String())))
				Expect(stateAfterCreate.BucketName.ValueString()).To(Equal(""))
			})
		})

		When("instance creation not working", func() {
			It("should not create state", func() {

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

				instanceRes := instance.NewInstanceResource(mockInstanceCLient, apiClient.DefaultAPI, providerData)

				mockInstanceCLient.EXPECT().CreateInstance(gomock.Any(), gomock.Any(), gomock.Any()).Return(modelexperiments.ApiCreateInstanceRequest{})
				mockInstanceCLient.EXPECT().CreateInstanceExecute(gomock.Any()).Return(nil, fmt.Errorf("server error"))

				schemaResp := resource.SchemaResponse{}
				instanceRes.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

				model := testutils.CreateInstanceTestModel(projectId.String(), region, instanceName, description)
				req := testutils.CreateInstanceRequest(ctx, schemaResp, model)
				resp := testutils.CreateResponse(schemaResp)

				instanceRes.Create(ctx, req, resp)

				Expect(resp.Diagnostics.HasError()).To(BeTrue())

				var stateAfterCreate *instance.Model
				diags := resp.State.Get(ctx, &stateAfterCreate)
				Expect(diags.HasError()).To(BeFalse())
				Expect(stateAfterCreate).To(BeNil())
			})
		})
	})
})
