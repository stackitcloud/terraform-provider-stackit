package routingtables

import (
	"context"
	"fmt"
	"testing"
	"time"

	"dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/iaasalpha"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/shared"
)

func TestMapDataFields(t *testing.T) {
	const testRegion = "eu01"
	id := fmt.Sprintf("%s,%s,%s", "oid", testRegion, "rtid")
	tests := []struct {
		description string
		state       DataSourceModelTables
		input       *iaasalpha.RoutingTableListResponse
		expected    DataSourceModelTables
		isValid     bool
	}{
		{
			"default_values",
			DataSourceModelTables{
				Id:             types.StringValue(id),
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("aid"),
				Region:         types.StringValue(testRegion),
			},
			&iaasalpha.RoutingTableListResponse{
				Items: &[]iaasalpha.RoutingTable{
					{
						Id:               utils.Ptr("rid"),
						Name:             utils.Ptr("test"),
						Description:      utils.Ptr("description"),
						MainRoutingTable: utils.Ptr(true),
					},
				},
			},
			DataSourceModelTables{
				Id:             types.StringValue(id),
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("aid"),
				Region:         types.StringValue(testRegion),
				Items: types.ListValueMust(types.ObjectType{AttrTypes: shared.DataSourceTypes}, []attr.Value{
					types.ObjectValueMust(shared.DataSourceTypes, map[string]attr.Value{
						"routing_table_id":   types.StringValue("rid"),
						"name":               types.StringValue("test"),
						"description":        types.StringValue("description"),
						"region":             types.StringValue(testRegion),
						"main_routing_table": types.BoolValue(true),
						"system_routes":      types.BoolValue(false),
						"created_at":         types.StringValue(""),
						"updated_at":         types.StringValue(""),
						"labels":             types.MapNull(types.StringType),
						// TODO: extend when routes are implemented
						"routes": types.ListNull(types.StringType),
					}),
				}),
			},
			true,
		},
		{
			"two routing tables",
			DataSourceModelTables{
				Id:             types.StringValue(id),
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("aid"),
				Region:         types.StringValue(testRegion),
			},
			&iaasalpha.RoutingTableListResponse{
				Items: &[]iaasalpha.RoutingTable{
					{
						Id:               utils.Ptr("rid"),
						Name:             utils.Ptr("test"),
						Description:      utils.Ptr("description"),
						MainRoutingTable: utils.Ptr(true),
						CreatedAt:        &time.Time{},
						UpdatedAt:        &time.Time{},
					},
					{
						Id:               utils.Ptr("rid2"),
						Name:             utils.Ptr("test2"),
						Description:      utils.Ptr("description2"),
						MainRoutingTable: utils.Ptr(false),
						CreatedAt:        &time.Time{},
						UpdatedAt:        &time.Time{},
					},
				},
			},
			DataSourceModelTables{
				Id:             types.StringValue(id),
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("aid"),
				Region:         types.StringValue(testRegion),
				Items: types.ListValueMust(types.ObjectType{AttrTypes: shared.DataSourceTypes}, []attr.Value{
					types.ObjectValueMust(shared.DataSourceTypes, map[string]attr.Value{
						"routing_table_id":   types.StringValue("rid"),
						"name":               types.StringValue("test"),
						"description":        types.StringValue("description"),
						"region":             types.StringValue(testRegion),
						"main_routing_table": types.BoolValue(true),
						"system_routes":      types.BoolValue(false),
						"created_at":         types.StringValue(""),
						"updated_at":         types.StringValue(""),
						"labels":             types.MapNull(types.StringType),
						// TODO: extend when routes are implemented
						"routes": types.ListNull(types.StringType),
					}),
					types.ObjectValueMust(shared.DataSourceTypes, map[string]attr.Value{
						"routing_table_id":   types.StringValue("rid2"),
						"name":               types.StringValue("test2"),
						"description":        types.StringValue("description2"),
						"region":             types.StringValue(testRegion),
						"main_routing_table": types.BoolValue(false),
						"system_routes":      types.BoolValue(false),
						"created_at":         types.StringValue(""),
						"updated_at":         types.StringValue(""),
						"labels":             types.MapNull(types.StringType),
						// TODO: extend when routes are implemented
						"routes": types.ListNull(types.StringType),
					}),
				}),
			},
			true,
		},
		{
			"response_fields_items_nil_fail",
			DataSourceModelTables{},
			&iaasalpha.RoutingTableListResponse{
				Items: nil,
			},
			DataSourceModelTables{},
			false,
		},
		{
			"response_nil_fail",
			DataSourceModelTables{},
			nil,
			DataSourceModelTables{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapDataSourceRoutingTables(context.Background(), tt.input, &tt.state, testRegion)
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
