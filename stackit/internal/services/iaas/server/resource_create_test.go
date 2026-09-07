package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// TestCreateReportsFailingGetServerDetails covers a missing return: the GetServer call that fetches
// the server details logged its error and then carried on with a nil server. The create path was
// saved by the nil check inside mapFields, but the diagnostic named the wrong cause. The same
// omission in Update has no such guard and panics there.
func TestCreateReportsFailingGetServerDetails(t *testing.T) {
	projectId := uuid.NewString()
	serverId := uuid.NewString()
	imageId := uuid.NewString()
	nicId := uuid.NewString()
	const (
		region      = "eu01"
		machineType = "g1.1"
	)

	server := iaas.Server{
		Id:          new(serverId),
		Name:        "srv",
		MachineType: machineType,
		Status:      new("ACTIVE"),
	}

	s := testutil.NewMockServer(t,
		testutil.MockResponse{Description: "create server", ToJsonBody: server},
		testutil.MockResponse{Description: "create waiter", ToJsonBody: server},
		testutil.MockResponse{Description: "failing get server details", StatusCode: http.StatusInternalServerError},
		testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
		testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusNotFound},
	)
	t.Cleanup(s.Server.Close)

	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	iaas_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
}
resource "stackit_server" "server" {
	project_id   = "%s"
	name         = "srv"
	machine_type = "%s"
	boot_volume = {
		size        = 32
		source_type = "image"
		source_id   = "%s"
	}
	network_interfaces = ["%s"]
}
`, region, s.Server.URL, projectId, machineType, imageId, nicId)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		// ExpectError only matches, it cannot assert an absence. Without the return the apply also
		// fails, but with a second diagnostic naming the wrong cause, so the absence is the assertion.
		ErrorCheck: func(err error) error {
			msg := err.Error()
			if !strings.Contains(msg, "get server details") {
				return fmt.Errorf("expected the diagnostic to name the failing GetServer call, got: %s", msg)
			}
			if strings.Contains(msg, "Processing API payload") {
				return fmt.Errorf("create continued past the failing GetServer and blamed the payload: %s", msg)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{Config: tfConfig},
		},
	})
}
