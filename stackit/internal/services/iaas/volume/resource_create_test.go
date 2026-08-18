package volume_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// TestCreateWriteOnlyKeyPayload is a regression test for the bug where the write-only key payload
// was read from the plan model instead of the config model.
// The test asserts that the value configured via key_payload_base64_wo is actually
// sent to the API in the create request.
func TestCreateWriteOnlyKeyPayload(t *testing.T) {
	projectId := uuid.NewString()
	volumeId := uuid.NewString()
	kekKeyId := uuid.NewString()
	kekKeyringId := uuid.NewString()
	const (
		region              = "eu01"
		availabilityZone    = "eu01-1"
		name                = "test-volume"
		size                = 16
		serviceAccount      = "test-sa@sa.stackit.cloud"
		testKeyPayload      = "VGhlIHF1aWNrIGJyb3duIGZveCBqdW1wcyBvdmVyIDEzIGxhenkgZG9ncy4="
		volumeStatusCreated = "AVAILABLE"
	)
	s := testutil.NewMockServer(t)
	t.Cleanup(s.Server.Close)
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	iaas_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_volume" "volume" {
	project_id = "%s"
	availability_zone = "%s"
	name = "%s"
	size = %d
	encryption_parameters = {
		kek_key_id = "%s"
		kek_key_version = 1
		kek_keyring_id = "%s"
		key_payload_base64_wo = "%s"
		key_payload_base64_wo_version = 1
		service_account = "%s"
	}
}
`, region, s.Server.URL, projectId, availabilityZone, name, size, kekKeyId, kekKeyringId, testKeyPayload, serviceAccount)

	volumeName := name
	volumeSize := int64(size)
	volume := iaas.Volume{
		Id:               &volumeId,
		Status:           new(volumeStatusCreated),
		AvailabilityZone: availabilityZone,
		Name:             &volumeName,
		Size:             &volumeSize,
	}

	var capturedKeyPayload *string
	createCalled := false
	createVolume := testutil.MockResponse{
		Description: "create",
		Handler: func(w http.ResponseWriter, req *http.Request) {
			expected := fmt.Sprintf("/v2/projects/%s/regions/%s/volumes", projectId, region)
			if req.URL.Path != expected {
				t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
			}
			createCalled = true
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Errorf("failed to read create request body: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var payload iaas.CreateVolumePayload
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Errorf("failed to unmarshal create request body: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if payload.EncryptionParameters != nil {
				capturedKeyPayload = payload.EncryptionParameters.KeyPayload
			}

			w.Header().Set("content-type", "application/json")
			_ = json.NewEncoder(w).Encode(iaas.Volume{Id: &volumeId})
		},
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						createVolume,
						testutil.MockResponse{Description: "create waiter", ToJsonBody: volume},
						testutil.MockResponse{Description: "get", ToJsonBody: volume},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusNotFound},
					)
				},
				Config: tfConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_volume.volume", "volume_id", volumeId),
					resource.TestCheckResourceAttr("stackit_volume.volume", "region", region),
					resource.TestCheckNoResourceAttr("stackit_volume.volume", "encryption_parameters.key_payload_base64_wo"),
					resource.TestCheckResourceAttr("stackit_volume.volume", "encryption_parameters.key_payload_base64_wo_version", "1"),
				),
			},
		},
	})

	if !createCalled {
		t.Fatalf("Expected the create endpoint to be called")
	}
	if capturedKeyPayload == nil {
		t.Fatalf("Expected key payload %q to be sent to the API, but none was sent", testKeyPayload)
	}
	if *capturedKeyPayload != testKeyPayload {
		t.Fatalf("Wrong key payload sent to the API: expected %q, got %q", testKeyPayload, *capturedKeyPayload)
	}
}
