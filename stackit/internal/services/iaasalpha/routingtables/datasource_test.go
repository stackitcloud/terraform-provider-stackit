package routingtables

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/shared"
)

const (
	testRegion = "eu01"
)

var (
	organizationId       = uuid.New()
	networkAreaId        = uuid.New()
	routingTableId       = uuid.New()
	secondRoutingTableId = uuid.New()
)

func TestMapDataFields(t *testing.T) {
	terraformId := fmt.Sprintf("%s,%s,%s", organizationId.String(), testRegion, networkAreaId.String())

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
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
			},
			&iaasalpha.RoutingTableListResponse{
				Items: &[]iaasalpha.RoutingTable{
					{
						Id:           utils.Ptr(routingTableId.String()),
						Name:         utils.Ptr("test"),
						Description:  utils.Ptr("description"),
						Default:      utils.Ptr(true),
						CreatedAt:    &time.Time{},
						UpdatedAt:    &time.Time{},
						SystemRoutes: utils.Ptr(false),
					},
				},
			},
			DataSourceModelTables{
				Id:             types.StringValue(terraformId),
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
				Items: types.ListValueMust(types.ObjectType{AttrTypes: shared.RoutingTableReadModelTypes()}, []attr.Value{
					types.ObjectValueMust(shared.RoutingTableReadModelTypes(), map[string]attr.Value{
						"routing_table_id": types.StringValue(routingTableId.String()),
						"name":             types.StringValue("test"),
						"description":      types.StringValue("description"),
						"default":          types.BoolValue(true),
						"system_routes":    types.BoolValue(false),
						"created_at":       types.StringNull(),
						"updated_at":       types.StringNull(),
						"labels":           types.MapNull(types.StringType),
					}),
				}),
			},
			true,
		},
		{
			"two routing tables",
			DataSourceModelTables{
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
			},
			&iaasalpha.RoutingTableListResponse{
				Items: &[]iaasalpha.RoutingTable{
					{
						Id:           utils.Ptr(routingTableId.String()),
						Name:         utils.Ptr("test"),
						Description:  utils.Ptr("description"),
						Default:      utils.Ptr(true),
						CreatedAt:    &time.Time{},
						UpdatedAt:    &time.Time{},
						SystemRoutes: utils.Ptr(false),
					},
					{
						Id:           utils.Ptr(secondRoutingTableId.String()),
						Name:         utils.Ptr("test2"),
						Description:  utils.Ptr("description2"),
						Default:      utils.Ptr(false),
						CreatedAt:    &time.Time{},
						UpdatedAt:    &time.Time{},
						SystemRoutes: utils.Ptr(false),
					},
				},
			},
			DataSourceModelTables{
				Id:             types.StringValue(terraformId),
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
				Items: types.ListValueMust(types.ObjectType{AttrTypes: shared.RoutingTableReadModelTypes()}, []attr.Value{
					types.ObjectValueMust(shared.RoutingTableReadModelTypes(), map[string]attr.Value{
						"routing_table_id": types.StringValue(routingTableId.String()),
						"name":             types.StringValue("test"),
						"description":      types.StringValue("description"),
						"default":          types.BoolValue(true),
						"system_routes":    types.BoolValue(false),
						"created_at":       types.StringNull(),
						"updated_at":       types.StringNull(),
						"labels":           types.MapNull(types.StringType),
					}),
					types.ObjectValueMust(shared.RoutingTableReadModelTypes(), map[string]attr.Value{
						"routing_table_id": types.StringValue(secondRoutingTableId.String()),
						"name":             types.StringValue("test2"),
						"description":      types.StringValue("description2"),
						"default":          types.BoolValue(false),
						"system_routes":    types.BoolValue(false),
						"created_at":       types.StringNull(),
						"updated_at":       types.StringNull(),
						"labels":           types.MapNull(types.StringType),
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
