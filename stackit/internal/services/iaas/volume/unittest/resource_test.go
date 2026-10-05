package unittest

import (
	_ "embed"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
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

	getCallCounter := new(-1)
	mockClient := iaas.DefaultAPIServiceMock{
		CreateVolumeExecuteMock: new(func(_ iaas.ApiCreateVolumeRequest) (*iaas.Volume, error) {
			return &iaas.Volume{
				Id: new(volumeId),
			}, nil
		}),
		GetVolumeExecuteMock: new(func(_ iaas.ApiGetVolumeRequest) (*iaas.Volume, error) {
			*getCallCounter++

			switch *getCallCounter {
			case 0:
				// creation wait handler
				return &iaas.Volume{
					Id:               new(volumeId),
					Status:           new("AVAILABLE"),
					Size:             new(int64(64)),
					AvailabilityZone: "eu01-1",
				}, nil
			case 1:
				// read request of deletion test step
				return &iaas.Volume{
					Id:               new(volumeId),
					Status:           new("AVAILABLE"),
					Size:             new(int64(64)),
					AvailabilityZone: "eu01-1",
				}, nil
			case 2:
				// deletion wait handler
				return nil, oapierror.NewError(http.StatusNotFound, "")
			}

			panic("should be unreachable")
		}),
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.NewTestUnitV6ProviderFactories(&core.MockClientFactory{
			IaaSV2ClientMock: mockClient,
		}),
		Steps: []resource.TestStep{
			{
				Config:          tfConfig,
				ConfigVariables: variables(),
			},
			// Note that Terraform automatically adds a step below which deletes all resources used in the unit test.
			// This means we also have to mock the delete request and the deletion wait handler.
		},
	})
}
