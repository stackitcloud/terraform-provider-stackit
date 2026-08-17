package volume

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

const (
	testProjectId        = "4e684f79-a12c-449d-aa89-bcd9d8aafaf2"
	testRegion           = "eu01"
	testVolumeId         = "3dee3fb9-59f0-4f97-8eeb-a4da37d05a00"
	testKeyPayloadBase64 = "VGhlIHF1aWNrIGJyb3duIGZveCBqdW1wcyBvdmVyIDEzIGxhenkgZG9ncy4="
)

// buildCreateRequest builds a resource.CreateRequest from a plan and a config model.
// Terraform populates write-only attribute values only in the config model - never in the plan or state model.
// That's why we need both, the plan model AND config model to build the request.
func buildCreateRequest(ctx context.Context, t *testing.T, schemaResp *resource.SchemaResponse, planModel, configModel *Model) resource.CreateRequest {
	t.Helper()

	req := resource.CreateRequest{}
	req.Plan = tfsdk.Plan{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	if diags := req.Plan.Set(ctx, planModel); diags.HasError() {
		t.Fatalf("Failed to set plan: %v", diags.Errors())
	}

	configScratch := tfsdk.Plan{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	if diags := configScratch.Set(ctx, configModel); diags.HasError() {
		t.Fatalf("Failed to set config: %v", diags.Errors())
	}
	req.Config = tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    configScratch.Raw,
	}

	return req
}

type volumeFixture struct {
	server             *httptest.Server
	capturedKeyPayload *string
	createCalled       bool
}

// newVolumeFixture spins up a mock IaaS API server handling volume creation and the subsequent
// polling of the wait handler. The create handler decodes the request body and records the
// encryption key payload that the provider sent to the API.
func newVolumeFixture(t *testing.T) *volumeFixture {
	t.Helper()
	fixture := &volumeFixture{}

	mux := http.NewServeMux()
	// Create volume
	mux.HandleFunc(fmt.Sprintf("POST /v2/projects/%s/regions/%s/volumes", testProjectId, testRegion), func(w http.ResponseWriter, r *http.Request) {
		fixture.createCalled = true
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read create request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var payload iaas.CreateVolumePayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("Failed to unmarshal create request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if payload.EncryptionParameters != nil {
			fixture.capturedKeyPayload = payload.EncryptionParameters.KeyPayload
		}

		w.Header().Set("content-type", "application/json")
		volumeId := testVolumeId
		_ = json.NewEncoder(w).Encode(iaas.Volume{Id: &volumeId})
	})
	// Get volume (used by the create wait handler and by mapFields via the response of the wait handler)
	mux.HandleFunc(fmt.Sprintf("GET /v2/projects/%s/regions/%s/volumes/%s", testProjectId, testRegion, testVolumeId), func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		volumeId := testVolumeId
		status := "AVAILABLE"
		_ = json.NewEncoder(w).Encode(iaas.Volume{
			Id:               &volumeId,
			Status:           &status,
			AvailabilityZone: "eu01-1",
		})
	})

	fixture.server = httptest.NewServer(mux)
	t.Cleanup(fixture.server.Close)
	return fixture
}

// newTestVolumeResource builds a volumeResource with the client's URL being set to the mock URL
func newTestVolumeResource(t *testing.T, server *httptest.Server) *volumeResource {
	t.Helper()
	client, err := iaas.NewAPIClient(
		sdkConfig.WithEndpoint(server.URL),
		sdkConfig.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}
	return &volumeResource{
		client: client,
		providerData: core.ProviderData{
			DefaultRegion: testRegion,
		},
	}
}

func encryptionParametersTestModel() *encryptionParametersModel {
	return &encryptionParametersModel{
		KekKeyId:                         types.StringValue("11111111-1111-1111-1111-111111111111"),
		KekKeyVersion:                    types.Int64Value(1),
		KekKeyringId:                     types.StringValue("22222222-2222-2222-2222-222222222222"),
		KeyPayloadBase64:                 types.StringNull(),
		KeyPayloadBase64WriteOnly:        types.StringNull(), // will be set manually for the config model
		KeyPayloadBase64WriteOnlyVersion: types.Int64Value(1),
		ServiceAccount:                   types.StringValue("test-sa@sa.stackit.cloud"),
	}
}

func baseTestModel() Model {
	return Model{
		ProjectId:        types.StringValue(testProjectId),
		Region:           types.StringValue(testRegion),
		AvailabilityZone: types.StringValue("eu01-1"),
		Name:             types.StringValue("test-volume"),
		Size:             types.Int64Value(16),
		Labels:           types.MapNull(types.StringType),
		Source:           types.ObjectNull(sourceTypes),
	}
}

// TestCreate_WriteOnlyKeyPayload is a regression test for the bug where the write-only key payload
// was read from the plan model instead of the config model.
// The test asserts that the value configured via key_payload_base64_wo is actually
// sent to the API in the create request.
func TestCreate_WriteOnlyKeyPayload(t *testing.T) {
	ctx := context.Background()

	// Usually terraform will only ever write write-only fields in the config model, not the plan.
	// Since we're setting the models manually here, we have to ensure this is done correctly.
	// Ensuring that the write-only fields never go into the state/plan model is not part of this test's scope here
	planModel := baseTestModel()
	planModel.EncryptionParameters = encryptionParametersTestModel()

	configModel := baseTestModel()
	configEncryptionParams := encryptionParametersTestModel()
	configEncryptionParams.KeyPayloadBase64WriteOnly = types.StringValue(testKeyPayloadBase64)
	configModel.EncryptionParameters = configEncryptionParams

	fixture := newVolumeFixture(t)
	iaasRessource := newTestVolumeResource(t, fixture.server)

	schemaResp := &resource.SchemaResponse{}
	iaasRessource.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	req := buildCreateRequest(ctx, t, schemaResp, &planModel, &configModel)
	// we have to set an initial empty state so it is != nil
	resp := &resource.CreateResponse{}
	resp.State = tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}

	iaasRessource.Create(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create should succeed, but got errors: %v", resp.Diagnostics.Errors())
	}
	if !fixture.createCalled {
		t.Fatalf("Expected the create endpoint to be called")
	}

	if fixture.capturedKeyPayload == nil {
		t.Fatalf("Expected key payload %q to be sent to the API, but none was sent", testKeyPayloadBase64)
	}
	if *fixture.capturedKeyPayload != testKeyPayloadBase64 {
		t.Fatalf("Wrong key payload sent to the API: expected %q, got %q", testKeyPayloadBase64, *fixture.capturedKeyPayload)
	}
}
