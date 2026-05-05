package ske

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
	serviceenablementWait "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api/wait"
	legacySke "github.com/stackitcloud/stackit-sdk-go/services/ske"
	ske "github.com/stackitcloud/stackit-sdk-go/services/ske/v2api"

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
								State: new(serviceenablementWait.SERVICESTATUSSTATE_ENABLED),
							},
							StatusCode: http.StatusOK,
						},
						testutil.MockResponse{
							Description: "service enablement wait handler",
							ToJsonBody: serviceenablement.ServiceStatus{
								State: new(serviceenablementWait.SERVICESTATUSSTATE_ENABLED),
								Error: nil,
							},
							StatusCode: http.StatusOK,
						},
						testutil.MockResponse{
							Description: "kubernetes versions",
							ToJsonBody: ske.ProviderOptions{
								MachineImages: []ske.MachineImage{
									{
										Name: new("flatcar"),
										Versions: []ske.MachineImageVersion{
											{
												State:          new("supported"),
												Version:        new("1.0.0"),
												ExpirationDate: nil,
												Cri: []ske.CRI{
													{
														Name: new(string(legacySke.CRINAME_CONTAINERD)),
													},
												},
											},
										},
									},
								},
								MachineTypes: []ske.MachineType{
									{
										Name: new(machineType),
									},
								},
								KubernetesVersions: []ske.KubernetesVersion{
									{
										State:          new("supported"),
										ExpirationDate: nil,
										Version:        new(kubernetesVersionMin),
									},
								},
							},
						},
						testutil.MockResponse{
							Description: "create",
							ToJsonBody: ske.Cluster{
								Name: new(clusterName),
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
								Items: []ske.Cluster{},
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

func TestSKEClusterNetworkEmpty(t *testing.T) {
	projectId := uuid.NewString()
	const (
		clusterName          = "cluster-name"
		kubernetesVersionMin = "1.33.8"
		region               = "eu01"
		machineType          = "g2i.2"
		nodeName             = "node-name"
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
  node_pools = [
	{
	  availability_zones = ["eu01-1"]
	  machine_type       = "%s"
	  os_version_min 	   = "1.0.0"
      maximum            = 2
      minimum            = 1
      max_surge = 1
      max_unavailable = 0
      name               = "%s"
	  volume_type        = "storage_premium_perf4"
	  volume_size        = 50
      labels = {}
    }
  ]
  network = {}
}

`, region, s.Server.URL, projectId, clusterName, kubernetesVersionMin, machineType, nodeName)

	skeCluster := ske.Cluster{
		Name: new(clusterName),
		Nodepools: []ske.Nodepool{
			{
				AllowSystemComponents: new(true),
				AvailabilityZones:     []string{"eu01-1"},
				Name:                  nodeName,
				Cri: new(ske.CRI{
					Name: new(string(legacySke.CRINAME_CONTAINERD)),
				}),
				Machine: ske.Machine{
					Image: ske.Image{
						Name:    "flatcar",
						Version: "1.0.0",
					},
					Type: machineType,
				},
				MaxSurge:       new(int32(1)),
				MaxUnavailable: new(int32(0)),
				Maximum:        2,
				Minimum:        1,
				Volume: ske.Volume{
					Size: 50,
					Type: new("storage_premium_perf4"),
				},
				Labels: new(map[string]string{}),
			},
		},
		Kubernetes: ske.Kubernetes{
			Version: kubernetesVersionMin,
		},
		Network: &ske.Network{
			Id: nil,
			ControlPlane: new(ske.V2ControlPlaneNetwork{
				AccessScope: new(ske.ACCESSSCOPE_PUBLIC),
			}),
		},
		Maintenance: new(ske.Maintenance{
			AutoUpdate: ske.MaintenanceAutoUpdate{
				KubernetesVersion:   new(true),
				MachineImageVersion: new(true),
			},
			TimeWindow: ske.TimeWindow{
				Start: time.Now(),
				End:   time.Now(),
			},
		}),
		Status: new(ske.ClusterStatus{
			Aggregated:       new(ske.CLUSTERSTATUSSTATE_STATE_HEALTHY),
			PodAddressRanges: []string{"100.64.0.0/10"},
		}),
		Extensions: new(ske.Extension{}),
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "service enablement request",
							ToJsonBody: serviceenablement.ServiceStatus{
								State: new(serviceenablementWait.SERVICESTATUSSTATE_ENABLED),
							},
							StatusCode: http.StatusOK,
						},
						testutil.MockResponse{
							Description: "service enablement wait handler",
							ToJsonBody: serviceenablement.ServiceStatus{
								State: new(serviceenablementWait.SERVICESTATUSSTATE_ENABLED),
								Error: nil,
							},
							StatusCode: http.StatusOK,
						},
						testutil.MockResponse{
							Description: "kubernetes versions",
							ToJsonBody: ske.ProviderOptions{
								MachineImages: []ske.MachineImage{
									{
										Name: new("flatcar"),
										Versions: []ske.MachineImageVersion{
											{
												State:          new("supported"),
												Version:        new("1.0.0"),
												ExpirationDate: nil,
												Cri: []ske.CRI{
													{
														Name: new(string(legacySke.CRINAME_CONTAINERD)),
													},
												},
											},
										},
									},
								},
								MachineTypes: []ske.MachineType{
									{
										Name: new(machineType),
									},
								},
								KubernetesVersions: []ske.KubernetesVersion{
									{
										State:          new("supported"),
										ExpirationDate: nil,
										Version:        new(kubernetesVersionMin),
									},
								},
							},
						},
						testutil.MockResponse{
							Description: "create",
							ToJsonBody:  skeCluster,
						},
						testutil.MockResponse{
							Description: "wait done",
							ToJsonBody:  skeCluster,
						},
						testutil.MockResponse{
							Description: "refresh",
							ToJsonBody:  skeCluster,
						},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{Description: "ListClusterResponse is called for checking removal",
							ToJsonBody: ske.ListClustersResponse{
								Items: []ske.Cluster{},
							},
						},
					)
				},
				Config: tfConfig,
			},
			{
				Config: tfConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						// Check that no update or replace will be triggered, if config has not changed
						plancheck.ExpectResourceAction("stackit_ske_cluster.cluster", plancheck.ResourceActionNoop),
					},
				},
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							ToJsonBody:  skeCluster,
						},
						testutil.MockResponse{
							Description: "get",
							ToJsonBody:  skeCluster,
						},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{Description: "ListClusterResponse is called for checking removal",
							ToJsonBody: ske.ListClustersResponse{
								Items: []ske.Cluster{},
							},
						},
					)
				},
			},
		},
	})
}
