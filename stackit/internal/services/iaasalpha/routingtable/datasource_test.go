package routingtable

import (
	"context"
	"fmt"
	"testing"

	"dev.azure.com/schwarzit/schwarzit.stackit-public/stackit-sdk-go-internal.git/services/iaasalpha"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaasalpha/shared"
)

// TODO: adjust and extend when Route Model is present
func TestMapDataFields(t *testing.T) {
	const testRegion = "eu01"
	id := fmt.Sprintf("%s,%s,%s", "oid", testRegion, "rtid")
	tests := []struct {
		description string
		state       shared.DataSourceModel
		input       *iaasalpha.RoutingTable
		expected    shared.DataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			shared.DataSourceModel{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("aid"),
				Routes:         types.ListNull(types.StringType),
			},
			&iaasalpha.RoutingTable{
				Id:   utils.Ptr("rtid"),
				Name: utils.Ptr("default_values"),
			},
			shared.DataSourceModel{
				Id:             types.StringValue(id),
				OrganizationId: types.StringValue("oid"),
				RoutingTableId: types.StringValue("rtid"),
				Name:           types.StringValue("default_values"),
				NetworkAreaId:  types.StringValue("aid"),
				Labels:         types.MapNull(types.StringType),
				Region:         types.StringValue(testRegion),
				Routes:         types.ListNull(types.StringType),
			},
			true,
		},
		{
			"values_ok",
			shared.DataSourceModel{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("aid"),
				Routes:         types.ListNull(types.StringType),
			},
			&iaasalpha.RoutingTable{
				Id:          utils.Ptr("rtid"),
				Name:        utils.Ptr("values_ok"),
				Description: utils.Ptr("Description"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			shared.DataSourceModel{
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
				Routes: types.ListNull(types.StringType),
			},
			true,
		},
		{
			"response_fields_nil_fail",
			shared.DataSourceModel{},
			&iaasalpha.RoutingTable{
				Id: nil,
			},
			shared.DataSourceModel{},
			false,
		},
		{
			"response_nil_fail",
			shared.DataSourceModel{},
			nil,
			shared.DataSourceModel{},
			false,
		},
		{
			"no_resource_id",
			shared.DataSourceModel{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
			},
			&iaasalpha.RoutingTable{},
			shared.DataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := shared.MapDataSourceFields(context.Background(), tt.input, &tt.state, testRegion)
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
