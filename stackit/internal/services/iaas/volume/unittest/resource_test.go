package unittest

import (
	_ "embed"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils/clientutils"
)

//go:embed testdata/resource.tf
var tfConfig string

func TestVolumeResource(t *testing.T) {
	projectId := uuid.NewString()
	volumeId := uuid.NewString()

	variables := func(mods ...func(variables config.Variables)) config.Variables {
		vars := config.Variables{
			"project_id":        config.StringVariable(projectId),
			"availability_zone": config.StringVariable("eu01-1"),
			"size":              config.IntegerVariable(64),
		}

		for _, mod := range mods {
			mod(vars)
		}

		return vars
	}

	mockClient := iaas.DefaultAPIServiceMock{
		CreateVolumeExecuteMock: utils.Ptr(func(r iaas.ApiCreateVolumeRequest) (*iaas.Volume, error) {
			return &iaas.Volume{
				Id: new(volumeId),
			}, nil
		}),
		GetVolumeExecuteMock: utils.Ptr(func(r iaas.ApiGetVolumeRequest) (*iaas.Volume, error) {
			return &iaas.Volume{
				Id:               new(volumeId),
				Status:           new("AVAILABLE"),
				Size:             new(int64(64)),
				AvailabilityZone: "eu01-1",
			}, nil
		}),
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.NewTestUnitV6ProviderFactories(&clientutils.MockClientFactory{
			IaaSV2ClientMock: mockClient,
		}),
		Steps: []resource.TestStep{
			{
				Config:          tfConfig,
				ConfigVariables: variables(),
			},
			{
				Config:          tfConfig,
				ConfigVariables: variables(),
				Check: func(s *terraform.State) error {
					// Clear the root module resources so the auto-destroy finds nothing
					s.RootModule().Resources = make(map[string]*terraform.ResourceState)
					return nil
				},
			},
		},
	})
}
