package sqlserverflex

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestSQLServerFlexInstanceSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	instanceId := uuid.NewString()
	const (
		name      = "instance-name"
		flavorCpu = 4
		flavorRam = 16
		flavorId  = "4.16-Single"
		region    = "eu01"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	sqlserverflex_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_sqlserverflex_instance" "instance" {
  project_id = "%s"
  name       = "%s"
  flavor = {
    cpu = %d
    ram = %d
  }
}

`, region, s.Server.URL, projectId, name, flavorCpu, flavorRam)
	flavor := testutil.MockResponse{
		ToJsonBody: &sqlserverflex.ListFlavorsResponse{
			Flavors: &[]sqlserverflex.InstanceFlavorEntry{
				{
					Cpu:         utils.Ptr(int64(flavorCpu)),
					Memory:      utils.Ptr(int64(flavorRam)),
					Id:          utils.Ptr(flavorId),
					Description: utils.Ptr("test-flavor-id"),
				},
			},
		},
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						flavor,
						testutil.MockResponse{
							Description: "create",
							ToJsonBody: sqlserverflex.CreateInstanceResponse{
								Id: utils.Ptr(instanceId),
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating instance.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v2/projects/%s/regions/%s/instances/%s", projectId, region, instanceId)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusNotFound},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading instance*"),
			},
		},
	})
}
