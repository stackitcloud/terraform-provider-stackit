package tables

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/routingtable/shared"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
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
	createdAt := time.Now()
	updatedAt := time.Now().Add(5 * time.Minute)

	tests := []struct {
		description string
		state       DataSourceModelTables
		input       *iaas.RoutingTableListResponse
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
			&iaas.RoutingTableListResponse{
				Items: &[]iaas.RoutingTable{
					{
						Id:            utils.Ptr(routingTableId.String()),
						Name:          utils.Ptr("test"),
						Description:   utils.Ptr("description"),
						Default:       utils.Ptr(true),
						CreatedAt:     &createdAt,
						UpdatedAt:     &updatedAt,
						SystemRoutes:  utils.Ptr(false),
						DynamicRoutes: utils.Ptr(false),
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
						"dynamic_routes":   types.BoolValue(false),
						"created_at":       types.StringValue(createdAt.Format(time.RFC3339)),
						"updated_at":       types.StringValue(updatedAt.Format(time.RFC3339)),
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
			&iaas.RoutingTableListResponse{
				Items: &[]iaas.RoutingTable{
					{
						Id:            utils.Ptr(routingTableId.String()),
						Name:          utils.Ptr("test"),
						Description:   utils.Ptr("description"),
						Default:       utils.Ptr(true),
						CreatedAt:     &createdAt,
						UpdatedAt:     &updatedAt,
						SystemRoutes:  utils.Ptr(false),
						DynamicRoutes: utils.Ptr(false),
					},
					{
						Id:            utils.Ptr(secondRoutingTableId.String()),
						Name:          utils.Ptr("test2"),
						Description:   utils.Ptr("description2"),
						Default:       utils.Ptr(false),
						CreatedAt:     &createdAt,
						UpdatedAt:     &updatedAt,
						SystemRoutes:  utils.Ptr(false),
						DynamicRoutes: utils.Ptr(false),
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
						"dynamic_routes":   types.BoolValue(false),
						"created_at":       types.StringValue(createdAt.Format(time.RFC3339)),
						"updated_at":       types.StringValue(updatedAt.Format(time.RFC3339)),
						"labels":           types.MapNull(types.StringType),
					}),
					types.ObjectValueMust(shared.RoutingTableReadModelTypes(), map[string]attr.Value{
						"routing_table_id": types.StringValue(secondRoutingTableId.String()),
						"name":             types.StringValue("test2"),
						"description":      types.StringValue("description2"),
						"default":          types.BoolValue(false),
						"system_routes":    types.BoolValue(false),
						"dynamic_routes":   types.BoolValue(false),
						"created_at":       types.StringValue(createdAt.Format(time.RFC3339)),
						"updated_at":       types.StringValue(updatedAt.Format(time.RFC3339)),
						"labels":           types.MapNull(types.StringType),
					}),
				}),
			},
			true,
		},
		{
			"response_fields_items_nil_fail",
			DataSourceModelTables{},
			&iaas.RoutingTableListResponse{
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
