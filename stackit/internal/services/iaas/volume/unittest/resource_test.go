package unittest

import (
	_ "embed"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
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

	var deleted bool
	mockClient := iaas.DefaultAPIServiceMock{
		CreateVolumeExecuteMock: utils.Ptr(func(_ iaas.ApiCreateVolumeRequest) (*iaas.Volume, error) {
			return &iaas.Volume{
				Id: new(volumeId),
			}, nil
		}),
		GetVolumeExecuteMock: utils.Ptr(func(_ iaas.ApiGetVolumeRequest) (*iaas.Volume, error) {
			if deleted {
				return nil, oapierror.NewError(http.StatusNotFound, "volume not found")
			}
			return &iaas.Volume{
				Id:               new(volumeId),
				Status:           new("AVAILABLE"),
				Size:             new(int64(64)),
				AvailabilityZone: "eu01-1",
			}, nil
		}),
		DeleteVolumeExecuteMock: utils.Ptr(func(_ iaas.ApiDeleteVolumeRequest) error {
			deleted = true
			return nil
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
		},
	})
}
