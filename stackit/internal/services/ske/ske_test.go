package ske

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceenablement"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestSKEClusterSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	const (
		clusterName          = "cluster-name"
		kubernetesVersionMin = "1.33.8"
		region               = "eu01"
		machineType          = "g2i.2"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	ske_custom_endpoint = "%[2]s"
	service_enablement_custom_endpoint = "%[2]s"
	service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_ske_cluster" "cluster" {
  project_id = "%s"
  name       = "%s"
  kubernetes_version_min = "%s"
  node_pools = [{
    availability_zones = ["eu01-1"]
    machine_type       = "%s"
	os_version_min 	   = "1.0.0"
    maximum            = 2
    minimum            = 1
    name               = "node-name"
    }
  ]
}

`, region, s.Server.URL, projectId, clusterName, kubernetesVersionMin, machineType)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "service enablement request",
							ToJsonBody: serviceenablement.ServiceStatus{
								State: utils.Ptr(serviceenablement.SERVICESTATUSSTATE_ENABLED),
							},
							StatusCode: http.StatusOK,
						},
						testutil.MockResponse{
							Description: "service enablement wait handler",
							ToJsonBody: serviceenablement.ServiceStatus{
								State: utils.Ptr(serviceenablement.SERVICESTATUSSTATE_ENABLED),
								Error: nil,
							},
							StatusCode: http.StatusOK,
						},
						testutil.MockResponse{
							Description: "kubernetes versions",
							ToJsonBody: ske.ProviderOptions{
								MachineImages: utils.Ptr([]ske.MachineImage{
									{
										Name: utils.Ptr("flatcar"),
										Versions: utils.Ptr([]ske.MachineImageVersion{
											{
												State:          utils.Ptr("supported"),
												Version:        utils.Ptr("1.0.0"),
												ExpirationDate: nil,
												Cri: utils.Ptr([]ske.CRI{
													{
														Name: utils.Ptr(ske.CRINAME_CONTAINERD),
													},
												}),
											},
										}),
									},
								}),
								MachineTypes: utils.Ptr([]ske.MachineType{
									{
										Name: utils.Ptr(machineType),
									},
								}),
								KubernetesVersions: utils.Ptr([]ske.KubernetesVersion{
									{
										State:          utils.Ptr("supported"),
										ExpirationDate: nil,
										Version:        utils.Ptr(kubernetesVersionMin),
									},
								}),
							},
						},
						testutil.MockResponse{
							Description: "create",
							ToJsonBody: ske.Cluster{
								Name: utils.Ptr(string(clusterName)),
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating/updating cluster.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v2/projects/%s/regions/%s/clusters/%s", projectId, region, clusterName)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{Description: "ListClusterResponse is called for checking removal",
							ToJsonBody: ske.ListClustersResponse{
								Items: &[]ske.Cluster{},
							},
						},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading cluster*"),
			},
		},
	})
}
