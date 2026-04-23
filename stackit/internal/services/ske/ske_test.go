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
								State: new(serviceenablement.SERVICESTATUSSTATE_ENABLED),
							},
							StatusCode: http.StatusOK,
						},
						testutil.MockResponse{
							Description: "service enablement wait handler",
							ToJsonBody: serviceenablement.ServiceStatus{
								State: new(serviceenablement.SERVICESTATUSSTATE_ENABLED),
								Error: nil,
							},
							StatusCode: http.StatusOK,
						},
						testutil.MockResponse{
							Description: "kubernetes versions",
							ToJsonBody: ske.ProviderOptions{
								MachineImages: new([]ske.MachineImage{
									{
										Name: new("flatcar"),
										Versions: new([]ske.MachineImageVersion{
											{
												State:          new("supported"),
												Version:        new("1.0.0"),
												ExpirationDate: nil,
												Cri: new([]ske.CRI{
													{
														Name: new(ske.CRINAME_CONTAINERD),
													},
												}),
											},
										}),
									},
								}),
								MachineTypes: new([]ske.MachineType{
									{
										Name: new(machineType),
									},
								}),
								KubernetesVersions: new([]ske.KubernetesVersion{
									{
										State:          new("supported"),
										ExpirationDate: nil,
										Version:        new(kubernetesVersionMin),
									},
								}),
							},
						},
						testutil.MockResponse{
							Description: "create",
							ToJsonBody: ske.Cluster{
								Name: new(string(clusterName)),
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
		Nodepools: new([]ske.Nodepool{
			{
				AllowSystemComponents: new(true),
				AvailabilityZones:     new([]string{"eu01-1"}),
				Name:                  new(nodeName),
				Cri: new(ske.CRI{
					Name: new(ske.CRINAME_CONTAINERD),
				}),
				Machine: new(ske.Machine{
					Image: new(ske.Image{
						Name:    new("flatcar"),
						Version: new("1.0.0"),
					}),
					Type: new(machineType),
				}),
				MaxSurge:       new(int64(1)),
				MaxUnavailable: new(int64(0)),
				Maximum:        new(int64(2)),
				Minimum:        new(int64(1)),
				Volume: new(ske.Volume{
					Size: new(int64(50)),
					Type: new("storage_premium_perf4"),
				}),
				Labels: new(map[string]string{}),
			},
		}),
		Kubernetes: new(ske.Kubernetes{
			Version: new(kubernetesVersionMin),
		}),
		Network: &ske.Network{
			Id: nil,
			ControlPlane: new(ske.V2ControlPlaneNetwork{
				AccessScope: new(ske.ACCESSSCOPE_PUBLIC),
			}),
		},
		Maintenance: new(ske.Maintenance{
			AutoUpdate: new(ske.MaintenanceAutoUpdate{
				KubernetesVersion:   new(true),
				MachineImageVersion: new(true),
			}),
			TimeWindow: new(ske.TimeWindow{
				Start: new(time.Now()),
				End:   new(time.Now()),
			}),
		}),
		Status: new(ske.ClusterStatus{
			Aggregated:       new(ske.CLUSTERSTATUSSTATE_HEALTHY),
			PodAddressRanges: new([]string{"100.64.0.0/10"}),
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
								State: new(serviceenablement.SERVICESTATUSSTATE_ENABLED),
							},
							StatusCode: http.StatusOK,
						},
						testutil.MockResponse{
							Description: "service enablement wait handler",
							ToJsonBody: serviceenablement.ServiceStatus{
								State: new(serviceenablement.SERVICESTATUSSTATE_ENABLED),
								Error: nil,
							},
							StatusCode: http.StatusOK,
						},
						testutil.MockResponse{
							Description: "kubernetes versions",
							ToJsonBody: ske.ProviderOptions{
								MachineImages: new([]ske.MachineImage{
									{
										Name: new("flatcar"),
										Versions: new([]ske.MachineImageVersion{
											{
												State:          new("supported"),
												Version:        new("1.0.0"),
												ExpirationDate: nil,
												Cri: new([]ske.CRI{
													{
														Name: new(ske.CRINAME_CONTAINERD),
													},
												}),
											},
										}),
									},
								}),
								MachineTypes: new([]ske.MachineType{
									{
										Name: new(machineType),
									},
								}),
								KubernetesVersions: new([]ske.KubernetesVersion{
									{
										State:          new("supported"),
										ExpirationDate: nil,
										Version:        new(kubernetesVersionMin),
									},
								}),
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
								Items: &[]ske.Cluster{},
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
								Items: &[]ske.Cluster{},
							},
						},
					)
				},
			},
		},
	})
}
