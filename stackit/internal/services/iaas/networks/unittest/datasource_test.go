package unittest

import (
	_ "embed"
	"fmt"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/datasource.tf
var tfConfig string

func TestNetworksResource(t *testing.T) {
	projectId := uuid.NewString()

	// Using pre-generated UUIDs since they are sorted inside the
	// tested func. Else no reproducible test result is possible.
	const (
		network01Id = "046bfa2c-4ca3-47b4-a979-63596c0e97e2"
		network02Id = "44aad1e1-f660-4408-9406-2fb31bbc23b9"
	)

	variables := func(mods ...func(variables config.Variables)) config.Variables {
		vars := config.Variables{
			"project_id": config.StringVariable(projectId),
		}

		for _, mod := range mods {
			mod(vars)
		}

		return vars
	}

	mockClient := iaas.DefaultAPIServiceMock{
		ListNetworksExecuteMock: new(func(_ iaas.ApiListNetworksRequest) (*iaas.NetworkListResponse, error) {
			return &iaas.NetworkListResponse{
				Items: []iaas.Network{
					// assert the networks will be sorted by their UUIDs for reproducible datasource outputs
					{
						Id:   network02Id,
						Name: "network-02",
					},
					{
						Id:   network01Id,
						Name: "network-01",
					},
				},
			}, nil
		}),
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.NewTestUnitV6ProviderFactories(&core.MockClientFactory{
			IaaSV2ClientMock: mockClient,
		}),
		Steps: []resource.TestStep{
			{
				// beta is not enabled - expect error
				Config:          fmt.Sprintf("%s\n\n%s", testutil.NewConfigBuilder().ServiceAccountToken("mock-server-needs-no-auth").EnableBetaResources(false).BuildProviderConfig(), tfConfig),
				ConfigVariables: variables(),
				ExpectError:     regexp.MustCompile(`The datasource "stackit_networks" is in beta and beta is not enabled`),
			},
			{
				Config:          fmt.Sprintf("%s\n\n%s", testutil.NewConfigBuilder().ServiceAccountToken("mock-server-needs-no-auth").EnableBetaResources(true).BuildProviderConfig(), tfConfig),
				ConfigVariables: variables(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_networks.networks", "project_id", projectId),
					resource.TestCheckResourceAttr("data.stackit_networks.networks", "results.#", "2"),
					resource.TestCheckResourceAttr("data.stackit_networks.networks", "results.0.network_id", network01Id),
					resource.TestCheckResourceAttr("data.stackit_networks.networks", "results.0.name", "network-01"),
					resource.TestCheckResourceAttr("data.stackit_networks.networks", "results.1.network_id", network02Id),
					resource.TestCheckResourceAttr("data.stackit_networks.networks", "results.1.name", "network-02"),
				),
			},
			// Note that Terraform automatically adds a step below which deletes all resources used in the unit test.
			// This means we also have to mock the delete request and the deletion wait handler.
		},
	})
}
