package table

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/routingtable/shared"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
)

const (
	testRegion = "eu01"
)

var (
	organizationId = uuid.New()
	networkAreaId  = uuid.New()
	routingTableId = uuid.New()
)

func Test_mapDatasourceFields(t *testing.T) {
	id := fmt.Sprintf("%s,%s,%s,%s", organizationId.String(), testRegion, networkAreaId.String(), routingTableId.String())

	tests := []struct {
		description string
		state       shared.RoutingTableDataSourceModel
		input       *iaasalpha.RoutingTable
		expected    shared.RoutingTableDataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			shared.RoutingTableDataSourceModel{
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
			},
			&iaasalpha.RoutingTable{
				Id:   utils.Ptr(routingTableId.String()),
				Name: utils.Ptr("default_values"),
			},
			shared.RoutingTableDataSourceModel{
				Id:             types.StringValue(id),
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
				RoutingTableReadModel: shared.RoutingTableReadModel{
					RoutingTableId: types.StringValue(routingTableId.String()),
					Name:           types.StringValue("default_values"),
					Labels:         types.MapNull(types.StringType),
				},
			},
			true,
		},
		{
			"values_ok",
			shared.RoutingTableDataSourceModel{
				OrganizationId:        types.StringValue(organizationId.String()),
				NetworkAreaId:         types.StringValue(networkAreaId.String()),
				RoutingTableReadModel: shared.RoutingTableReadModel{},
			},
			&iaasalpha.RoutingTable{
				Id:          utils.Ptr(routingTableId.String()),
				Name:        utils.Ptr("values_ok"),
				Description: utils.Ptr("Description"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			shared.RoutingTableDataSourceModel{
				Id:             types.StringValue(id),
				OrganizationId: types.StringValue(organizationId.String()),
				NetworkAreaId:  types.StringValue(networkAreaId.String()),
				Region:         types.StringValue(testRegion),
				RoutingTableReadModel: shared.RoutingTableReadModel{
					RoutingTableId: types.StringValue(routingTableId.String()),
					Name:           types.StringValue("values_ok"),
					Description:    types.StringValue("Description"),
					Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
						"key": types.StringValue("value"),
					}),
				},
			},
			true,
		},
		{
			"response_fields_nil_fail",
			shared.RoutingTableDataSourceModel{},
			&iaasalpha.RoutingTable{
				Id: nil,
			},
			shared.RoutingTableDataSourceModel{},
			false,
		},
		{
			"response_nil_fail",
			shared.RoutingTableDataSourceModel{},
			nil,
			shared.RoutingTableDataSourceModel{},
			false,
		},
		{
			"no_resource_id",
			shared.RoutingTableDataSourceModel{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
			},
			&iaasalpha.RoutingTable{},
			shared.RoutingTableDataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapDatasourceFields(context.Background(), tt.input, &tt.state, testRegion)
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
