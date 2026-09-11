package table

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func TestMapFields(t *testing.T) {
	const testRegion = "eu01"
	id := fmt.Sprintf("%s,%s,%s,%s", "oid", testRegion, "aid", "rtid")
	tests := []struct {
		description string
		state       Model
		input       *iaas.RoutingTable
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("aid"),
			},
			&iaas.RoutingTable{
				Id:   new("rtid"),
				Name: "default_values",
			},
			Model{
				Id:             types.StringValue(id),
				OrganizationId: types.StringValue("oid"),
				RoutingTableId: types.StringValue("rtid"),
				Name:           types.StringValue("default_values"),
				NetworkAreaId:  types.StringValue("aid"),
				Labels:         types.MapNull(types.StringType),
				Region:         types.StringValue(testRegion),
			},
			true,
		},
		{
			"values_ok",
			Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("aid"),
			},
			&iaas.RoutingTable{
				Id:          new("rtid"),
				Name:        "values_ok",
				Description: new("Description"),
				Labels: map[string]any{
					"key": "value",
				},
			},
			Model{
				Id:             types.StringValue(id),
				OrganizationId: types.StringValue("oid"),
				RoutingTableId: types.StringValue("rtid"),
				Name:           types.StringValue("values_ok"),
				Description:    types.StringValue("Description"),
				NetworkAreaId:  types.StringValue("aid"),
				Region:         types.StringValue(testRegion),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			true,
		},
		{
			"response_fields_nil_fail",
			Model{},
			&iaas.RoutingTable{
				Id: nil,
			},
			Model{},
			false,
		},
		{
			"response_nil_fail",
			Model{},
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
			},
			&iaas.RoutingTable{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state, testRegion)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *iaas.AddRoutingTableToAreaPayload
		isValid     bool
	}{
		{
			description: "default_ok",
			input: &Model{
				Description: types.StringValue("Description"),
				Name:        types.StringValue("default_ok"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				SystemRoutes:  types.BoolValue(true),
				DynamicRoutes: types.BoolValue(true),
			},
			expected: &iaas.AddRoutingTableToAreaPayload{
				Description: new("Description"),
				Name:        "default_ok",
				Labels: map[string]any{
					"key": "value",
				},
				SystemRoutes:  new(true),
				DynamicRoutes: new(true),
			},
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *iaas.UpdateRoutingTableOfAreaPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Description: types.StringValue("Description"),
				Name:        types.StringValue("default_ok"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key1": types.StringValue("value1"),
					"key2": types.StringValue("value2"),
				}),
				DynamicRoutes: types.BoolValue(false),
				SystemRoutes:  types.BoolValue(false),
			},
			&iaas.UpdateRoutingTableOfAreaPayload{
				Description: new("Description"),
				Name:        new("default_ok"),
				Labels: map[string]any{
					"key1": "value1",
					"key2": "value2",
				},
				DynamicRoutes: new(false),
				SystemRoutes:  new(false),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input, types.MapNull(types.StringType))
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected, cmp.AllowUnexported(iaas.NullableString{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestRead(t *testing.T) {
	const (
		organizationId = "0f0f0f0f-0f0f-0f0f-0f0f-0f0f0f0f0f0f"
		networkAreaId  = "1e1e1e1e-1e1e-1e1e-1e1e-1e1e1e1e1e1e"
		routingTableId = "2d2d2d2d-2d2d-2d2d-2d2d-2d2d2d2d2d2d"
		region         = "eu01"
	)
	priorState := Model{
		Id:             utils.BuildInternalTerraformId(organizationId, region, networkAreaId, routingTableId),
		OrganizationId: types.StringValue(organizationId),
		RoutingTableId: types.StringValue(routingTableId),
		NetworkAreaId:  types.StringValue(networkAreaId),
		Name:           types.StringValue("example"),
		Region:         types.StringValue(region),
		Labels:         types.MapNull(types.StringType),
		SystemRoutes:   types.BoolValue(true),
		DynamicRoutes:  types.BoolValue(true),
	}

	tests := []struct {
		name        string
		statusCode  int
		body        string
		closeServer bool // close the mocked server before Read to provoke a transport error
		wantErr     bool
		wantRemoved bool
		wantState   *Model // expected state after Read; nil keeps the prior state
	}{
		{
			name:       "routing table exists",
			statusCode: http.StatusOK,
			body:       fmt.Sprintf(`{"id": %q, "name": "renamed", "description": "d", "systemRoutes": false, "dynamicRoutes": true, "labels": {"k": "v"}}`, routingTableId),
			wantState: &Model{
				Id:             utils.BuildInternalTerraformId(organizationId, region, networkAreaId, routingTableId),
				OrganizationId: types.StringValue(organizationId),
				RoutingTableId: types.StringValue(routingTableId),
				NetworkAreaId:  types.StringValue(networkAreaId),
				Name:           types.StringValue("renamed"),
				Description:    types.StringValue("d"),
				Region:         types.StringValue(region),
				Labels:         types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringValue("v")}),
				SystemRoutes:   types.BoolValue(false),
				DynamicRoutes:  types.BoolValue(true),
				CreatedAt:      types.StringNull(),
				UpdatedAt:      types.StringNull(),
			},
		},
		{
			// The IaaS API answers 404 for routing tables of a deleted network area or network area region
			name:        "404 removes the routing table from state without an error",
			statusCode:  http.StatusNotFound,
			body:        `{"code": 404, "msg": "resource not found: area"}`,
			wantRemoved: true,
		},
		{
			name:       "403 reports an error and keeps the state",
			statusCode: http.StatusForbidden,
			body:       `{"code": 403, "msg": "forbidden"}`,
			wantErr:    true,
		},
		{
			name:       "500 reports an error and keeps the state",
			statusCode: http.StatusInternalServerError,
			body:       `{"code": 500, "msg": "internal error"}`,
			wantErr:    true,
		},
		{
			name:        "transport error reports an error and keeps the state",
			closeServer: true,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := mux.NewRouter()
			router.HandleFunc("/v2/organizations/{organizationId}/network-areas/{areaId}/regions/{region}/routing-tables/{routingTableId}", func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				if r.Method != http.MethodGet || vars["organizationId"] != organizationId || vars["areaId"] != networkAreaId || vars["region"] != region || vars["routingTableId"] != routingTableId {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if _, err := w.Write([]byte(tt.body)); err != nil {
					t.Errorf("Get routing table handler: failed to write response: %v", err)
				}
			})
			mockedServer := httptest.NewServer(router)
			defer mockedServer.Close()
			client, err := iaas.NewAPIClient(
				config.WithEndpoint(mockedServer.URL),
				config.WithoutAuthentication(),
			)
			if err != nil {
				t.Fatalf("Failed to initialize client: %v", err)
			}
			if tt.closeServer {
				mockedServer.Close()
			}
			r := &routingTableResource{client: client, providerData: core.ProviderData{DefaultRegion: region}}

			ctx := context.Background()
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			state := tfsdk.State{Schema: schemaResp.Schema}
			if diags := state.Set(ctx, priorState); diags.HasError() {
				t.Fatalf("Failed to build state: %v", diags)
			}

			resp := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &resp)

			if resp.Diagnostics.HasError() != tt.wantErr {
				t.Errorf("Read() error diagnostics = %v, wantErr %v: %v", resp.Diagnostics.HasError(), tt.wantErr, resp.Diagnostics)
			}
			if resp.State.Raw.IsNull() != tt.wantRemoved {
				t.Errorf("Read() removed resource from state = %v, want %v", resp.State.Raw.IsNull(), tt.wantRemoved)
			}
			if tt.wantRemoved {
				return
			}
			var got Model
			if diags := resp.State.Get(ctx, &got); diags.HasError() {
				t.Fatalf("Failed to read state back: %v", diags)
			}
			want := priorState
			if tt.wantState != nil {
				want = *tt.wantState
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Read() state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
