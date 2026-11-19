package key_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

// TestServiceAccountKeyReadNonAPIError checks that a non-API error on read (e.g. a dropped connection)
// is reported as a diagnostic instead of crashing the provider.
func TestServiceAccountKeyReadNonAPIError(t *testing.T) {
	projectId := uuid.NewString()
	keyId := uuid.NewString()
	const email = "sa@sa.stackit.cloud"

	keyJSON := fmt.Sprintf(`{
		"active": true,
		"createdAt": %q,
		"credentials": {"aud": "aud", "iss": %q, "kid": %q, "sub": "sub"},
		"id": %q,
		"keyAlgorithm": "RSA_2048",
		"keyOrigin": "GENERATED",
		"keyType": "USER_MANAGED",
		"publicKey": "public-key"
	}`, time.Now().UTC().Format(time.RFC3339), email, keyId, keyId)

	var dropConnection atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(keyJSON))
		case http.MethodGet:
			if dropConnection.Load() {
				hijacker, ok := w.(http.Hijacker)
				if !ok {
					t.Errorf("response writer does not support hijacking")
					return
				}
				conn, _, err := hijacker.Hijack()
				if err != nil {
					t.Errorf("hijacking connection: %v", err)
					return
				}
				_ = conn.Close()
				return
			}
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(keyJSON))
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	config := fmt.Sprintf(`
provider "stackit" {
	service_account_custom_endpoint = "%s"
	service_account_token           = "mock-server-needs-no-auth"
}

resource "stackit_service_account_key" "key" {
	project_id            = "%s"
	service_account_email = "%s"
}
`, server.URL, projectId, email)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				PreConfig:    func() { dropConnection.Store(true) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile("Error reading service account key"),
			},
			{
				PreConfig: func() { dropConnection.Store(false) },
				Config:    config,
			},
		},
	})
}
