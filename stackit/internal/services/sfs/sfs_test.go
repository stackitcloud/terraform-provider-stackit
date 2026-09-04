package sfs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	corewait "github.com/stackitcloud/stackit-sdk-go/core/wait"
	sfs "github.com/stackitcloud/stackit-sdk-go/services/sfs/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestSfsResourcePoolSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	instanceId := uuid.NewString()
	const (
		region = "eu01"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	sfs_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
	enable_beta_resources = true
}
resource "stackit_sfs_resource_pool" "resourcepool" {
  project_id        = "%s"
  name              = "sfs-instance"
  availability_zone = "eu01-m"
  performance_class = "Standard"
  size_gigabytes    = 512
  ip_acl            = ["192.168.2.0/24"]
}
`, region, s.Server.URL, projectId)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create instance",
							ToJsonBody: sfs.CreateResourcePoolResponse{
								ResourcePool: &sfs.ResourcePool{
									Id: new(instanceId),
								},
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating resource pool.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/regions/%s/resourcePools/%s", projectId, region, instanceId)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{
							Description: "delete waiter",
							StatusCode:  http.StatusNotFound,
						},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading resource pool*"),
			},
		},
	})
}

func TestSfsShareSavesIDsOnError(t *testing.T) {
	projectId := uuid.NewString()
	instanceId := uuid.NewString()
	resourcePoolId := uuid.NewString()
	const (
		region = "eu01"
	)
	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	sfs_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
	enable_beta_resources = true
}
resource "stackit_sfs_share" "example" {
  project_id                 = "%s"
  resource_pool_id           = "%s"
  name                       = "my-nfs-share"
  export_policy              = "high-performance-class"
  space_hard_limit_gigabytes = 32
}
`, region, s.Server.URL, projectId, resourcePoolId)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create instance",
							ToJsonBody: sfs.CreateShareResponse{
								Share: &sfs.Share{
									Id: new(instanceId),
								},
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating share.*"),
			},
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "refresh",
							Handler: func(w http.ResponseWriter, req *http.Request) {
								expected := fmt.Sprintf("/v1/projects/%s/regions/%s/resourcePools/%s/shares/%s", projectId, region, resourcePoolId, instanceId)
								if req.URL.Path != expected {
									t.Errorf("expected request to %s, got %s", expected, req.URL.Path)
								}
								w.WriteHeader(http.StatusInternalServerError)
							},
						},
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{
							Description: "delete waiter",
							StatusCode:  http.StatusNotFound,
						},
					)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading share*"),
			},
		},
	})
}

// TestSfsResourcePoolCreateTimeoutIsConfigurable asserts that the configured `timeouts.create` value is what ends
// the create wait. The wait handler applies its own hardcoded 10 minutes otherwise, which no configuration reaches.
//
// Three signals are needed, because the provider reports every wait failure through the same message: the error must
// name the configured value (only the deadline branch does that), the pool must have been polled more than once, and
// the polling window must be close to the configured value rather than to zero or to the mock's poll budget.
func TestSfsResourcePoolCreateTimeoutIsConfigurable(t *testing.T) {
	projectId := uuid.NewString()
	resourcePoolId := uuid.NewString()
	const (
		region = "eu01"
		// Longer than the wait handler's 5s throttle, so several polls happen before the deadline ends the wait.
		createTimeout = 12 * time.Second
		// Safety valve. Without a context deadline in Create the wait would run for the handler's own 10 minutes and
		// blow the package test timeout before any assertion could report the regression.
		pollBudget = 30 * time.Second
		poolState  = "creating"
	)

	var (
		mu         sync.Mutex
		createdAt  time.Time
		lastPollAt time.Time
		polls      int
		deleted    bool
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch req.Method {
		case http.MethodPost:
			createdAt = time.Now()
			writeJSON(t, w, sfs.CreateResourcePoolResponse{
				ResourcePool: &sfs.ResourcePool{Id: new(resourcePoolId)},
			})
		case http.MethodDelete:
			deleted = true
			w.WriteHeader(http.StatusAccepted)
		default:
			// Report the pool as gone once the test cleanup has deleted it, so the delete wait can finish.
			if deleted {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			polls++
			lastPollAt = time.Now()
			if time.Since(createdAt) > pollBudget {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// The pool never becomes ready, so only a timeout can end the wait.
			writeJSON(t, w, sfs.GetResourcePoolResponse{
				ResourcePool: &sfs.ResourcePool{Id: new(resourcePoolId), State: new(poolState)},
			})
		}
	}))
	defer server.Close()

	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	sfs_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
	enable_beta_resources = true
}
resource "stackit_sfs_resource_pool" "resourcepool" {
  project_id        = "%s"
  name              = "sfs-instance"
  availability_zone = "eu01-m"
  performance_class = "Standard"
  size_gigabytes    = 512
  ip_acl            = ["192.168.2.0/24"]

  timeouts = {
    create = "%s"
  }
}
`, region, server.URL, projectId, createTimeout)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tfConfig,
				// Only the deadline branch of the error names the configured value.
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(createTimeout.String())),
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	if polls < 2 {
		t.Errorf("the create wait polled %d times, expected it to keep polling until the configured timeout of %s",
			polls, createTimeout)
	}
	waited := lastPollAt.Sub(createdAt)
	if waited < createTimeout/2 {
		t.Errorf("the create wait ran for %s, expected roughly the configured %s: it failed for some other reason "+
			"before the timeout was reached", waited, createTimeout)
	}
	if waited > pollBudget-5*time.Second {
		t.Errorf("the create wait ran for %s, expected it to give up after the configured %s: the configured value "+
			"is ignored and the wait ran into the mock's poll budget", waited, createTimeout)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()
	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Errorf("Error writing response body: %v", err)
	}
}

// TestSfsResourcePoolKeepsConfiguredTimeoutsOnError asserts that the configured timeouts are part of the partial
// state the resource writes before it starts waiting. They are needed there: after a failed create Terraform marks
// the resource tainted and the next run refreshes it and destroys it, and both operations read their timeout from
// that state entry. Were the attribute missing, those steps would silently fall back to the default timeouts.
func TestSfsResourcePoolKeepsConfiguredTimeoutsOnError(t *testing.T) {
	projectId := uuid.NewString()
	resourcePoolId := uuid.NewString()
	const (
		region        = "eu01"
		deleteTimeout = "42m"
	)

	s := testutil.NewMockServer(t)
	defer s.Server.Close()
	tfConfig := fmt.Sprintf(`
provider "stackit" {
	default_region = "%s"
	sfs_custom_endpoint = "%s"
	service_account_token = "mock-server-needs-no-auth"
	enable_beta_resources = true
}
resource "stackit_sfs_resource_pool" "resourcepool" {
  project_id        = "%s"
  name              = "sfs-instance"
  availability_zone = "eu01-m"
  performance_class = "Standard"
  size_gigabytes    = 512
  ip_acl            = ["192.168.2.0/24"]

  timeouts = {
    delete = "%s"
  }
}
`, region, s.Server.URL, projectId, deleteTimeout)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					s.Reset(
						testutil.MockResponse{
							Description: "create resource pool",
							ToJsonBody: sfs.CreateResourcePoolResponse{
								ResourcePool: &sfs.ResourcePool{Id: new(resourcePoolId)},
							},
						},
						testutil.MockResponse{
							Description: "failing waiter",
							StatusCode:  http.StatusInternalServerError,
						},
					)
				},
				Config:      tfConfig,
				ExpectError: regexp.MustCompile("Error creating resource pool"),
			},
			{
				PreConfig: func() {
					pool := testutil.MockResponse{
						Description: "read resource pool",
						ToJsonBody: sfs.GetResourcePoolResponse{
							ResourcePool: &sfs.ResourcePool{Id: new(resourcePoolId)},
						},
					}
					// The step refreshes and then plans, and both read the resource.
					s.Reset(
						pool,
						pool,
						testutil.MockResponse{Description: "delete", StatusCode: http.StatusAccepted},
						testutil.MockResponse{Description: "delete waiter", StatusCode: http.StatusNotFound},
					)
				},
				RefreshState: true,
				// The failed create left the resource tainted, so the follow-up plan is a replacement.
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "resource_pool_id", resourcePoolId),
					resource.TestCheckResourceAttr("stackit_sfs_resource_pool.resourcepool", "timeouts.delete", deleteTimeout),
				),
			},
		},
	})
}

// TestWaitHandlerTimeoutIsBoundedByContext pins the SDK behavior that makes the `timeouts`
// attribute effective at all: the wait handler applies its own timeout only when the passed
// context carries no deadline. Should a future SDK version enforce the handler timeout
// unconditionally, a `timeouts.create` above the handler default would silently be capped
// again, which is the bug the attribute was added for.
func TestWaitHandlerTimeoutIsBoundedByContext(t *testing.T) {
	const (
		handlerTimeout = 50 * time.Millisecond
		contextTimeout = time.Second
	)

	// The check never finishes, so only one of the two timeouts can end the wait.
	handler := corewait.New(func() (bool, *struct{}, error) { return false, nil, nil })
	handler.SetTimeout(handlerTimeout).SetThrottle(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	start := time.Now()
	if _, err := handler.WaitWithContext(ctx); err == nil {
		t.Fatal("expected the wait to time out")
	}
	if elapsed := time.Since(start); elapsed < contextTimeout {
		t.Errorf("wait gave up after %s, expected it to run until the context deadline at %s: "+
			"the SDK wait handler enforces its own timeout despite the context deadline, so the "+
			"`timeouts` attribute can no longer raise it", elapsed, contextTimeout)
	}
}
