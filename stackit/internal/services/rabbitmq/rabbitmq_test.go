package rabbitmq

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	rabbitmq "github.com/stackitcloud/stackit-sdk-go/services/rabbitmq/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestRabbitMQInstanceSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	instanceId := uuid.NewString()
	const (
		name     = "instance-name"
		planName = "plan-name"
		planId   = "plan-id"
		version  = "version"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	rabbitmq_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_rabbitmq_instance" "instance" {
	project_id = "%s"
	name = "%s"
	plan_name = "%s"
	version = "%s"
}
`, s.Server.URL, projectId, name, planName, version)
	offerings := testutil.MockResponse{
		ToJsonBody: &rabbitmq.ListOfferingsResponse{
			Offerings: []rabbitmq.Offering{
				{
					Version: version,
					Plans: []rabbitmq.Plan{
						{
							Name: planName,
							Id:   planId,
						},
					},
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
						offerings,
						testutil.MockResponse{
							Description: "create",
							ToJsonBody: rabbitmq.CreateInstanceResponse{
								InstanceId: instanceId,
							},
						},
						testutil.MockResponse{Description: "create waiter", StatusCode: http.StatusInternalServerError},
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
								expected := fmt.Sprintf("/v1/projects/%s/instances/%s", projectId, instanceId)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusGone},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading instance.*"),
			},
		},
	})
}

func TestRabbitMQCredentialsSavesIDsOnError(t *testing.T) {
	var (
		projectId    = uuid.NewString()
		instanceId   = uuid.NewString()
		credentialId = uuid.NewString()
	)
	s := testutil.NewMockServer(t)
	t.Cleanup(s.Server.Close)
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	rabbitmq_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_rabbitmq_credential" "credential" {
	project_id = "%s"
	instance_id = "%s"
}
`, s.Server.URL, projectId, instanceId)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						// initial post response
						testutil.MockResponse{
							ToJsonBody: rabbitmq.CredentialsResponse{
								Id: credentialId,
							},
						},
						// failing waiter
						testutil.MockResponse{StatusCode: http.StatusInternalServerError},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating credential.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/instances/%s/credentials/%s", projectId, instanceId, credentialId)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusGone},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading credential.*"),
			},
		},
	})
}
