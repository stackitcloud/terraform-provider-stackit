package mongodbflex

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// slow test, delete has a 30s sleep...
func TestMongoDBInstanceSavesIDsOnError(t *testing.T) {
	var (
		projectId  = uuid.NewString()
		instanceId = uuid.NewString()
	)
	const (
		name   = "instance-test"
		region = "eu01"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	mongodbflex_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_mongodbflex_instance" "instance" {
	project_id = "%s"
	name    = "%s"
	options = {
		type = "Replica"
		snapshot_retention_days = 1 
		daily_snapshot_retention_days = 1
		point_in_time_window_hours = 1
	}
	storage = {
		class = "premium-perf2-mongodb"
		size = 10 
	}
	replicas = 1 
	acl = ["192.168.0.0/16"]
	flavor = {
		cpu =2 
		ram =4 
	}
	version = "7.0"
	backup_schedule = "00 6 * * *"
}
`, s.Server.URL, projectId, name)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "ListFlavors",
							ToJsonBody: &mongodbflex.ListFlavorsResponse{Flavors: &[]mongodbflex.InstanceFlavor{
								{
									Description: new("flava-flav"),
									Cpu:         new(int64(2)),
									Id:          new("flavor-id"),
									Memory:      new(int64(4)),
								},
							}},
						},
						testutil.MockResponse{
							Description: "create instance",
							ToJsonBody:  &mongodbflex.CreateInstanceResponse{Id: new(instanceId)},
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
								expected := fmt.Sprintf("/v2/projects/%s/regions/%s/instances/%s", projectId, region, instanceId)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete"},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusNotFound},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading instance.*"),
			},
		},
	})
}

func TestMongoDBUserSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	instanceId := uuid.NewString()
	userId := uuid.NewString()
	const region = "eu01"
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	mongodbflex_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
}

resource "stackit_mongodbflex_user" "user" {
	project_id = "%s"
	instance_id = "%s"
	username = "username"
	roles = ["read"]
	database = "db-name"
}
`, s.Server.URL, projectId, instanceId)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create user",
							ToJsonBody:  &mongodbflex.CreateUserResponse{Item: &mongodbflex.User{Id: new(userId)}},
						},
						testutil.MockResponse{Description: "failing waiter", StatusCode: http.StatusInternalServerError},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating user.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh user",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v2/projects/%s/regions/%s/instances/%s/users/%s", projectId, region, instanceId, userId)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete user"},
						testutil.MockResponse{Description: "delete user waiter", StatusCode: http.StatusNotFound},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading user.*"),
			},
		},
	})
}
