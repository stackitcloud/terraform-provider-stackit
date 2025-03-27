package networkarea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

var testOrganizationId = uuid.NewString()
var testAreaId = uuid.NewString()
var testRangeId1 = uuid.NewString()
var testRangeId2 = uuid.NewString()
var testRangeId3 = uuid.NewString()
var testRangeId4 = uuid.NewString()
var testRangeId5 = uuid.NewString()
var testRangeId2Repeated = uuid.NewString()

func TestMapFields(t *testing.T) {
	tests := []struct {
		description       string
		state             Model
		input             *iaas.NetworkArea
		ListNetworkRanges *[]iaas.NetworkRange
		expected          Model
		isValid           bool
	}{
		{
			"id_ok",
			Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId1),
						"prefix":           types.StringValue("prefix-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId2),
						"prefix":           types.StringValue("prefix-2"),
					}),
				}),
			},
			&iaas.NetworkArea{
				AreaId: utils.Ptr("naid"),
				Ipv4:   &iaas.NetworkAreaIPv4{},
			},
			&[]iaas.NetworkRange{
				{
					NetworkRangeId: utils.Ptr(testRangeId1),
					Prefix:         utils.Ptr("prefix-1"),
				},
				{
					NetworkRangeId: utils.Ptr(testRangeId2),
					Prefix:         utils.Ptr("prefix-2"),
				},
			},

			Model{
				Id:                  types.StringValue("oid,naid"),
				OrganizationId:      types.StringValue("oid"),
				NetworkAreaId:       types.StringValue("naid"),
				Name:                types.StringNull(),
				DefaultNameservers:  types.ListNull(types.StringType),
				TransferNetwork:     types.StringNull(),
				DefaultPrefixLength: types.Int64Null(),
				MaxPrefixLength:     types.Int64Null(),
				MinPrefixLength:     types.Int64Null(),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId1),
						"prefix":           types.StringValue("prefix-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId2),
						"prefix":           types.StringValue("prefix-2"),
					}),
				}),
				Labels: types.MapNull(types.StringType),
			},
			true,
		},
		{
			"values_ok",
			Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId1),
						"prefix":           types.StringValue("prefix-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId2),
						"prefix":           types.StringValue("prefix-2"),
					}),
				}),
			},
			&iaas.NetworkArea{
				AreaId: utils.Ptr("naid"),
				Ipv4: &iaas.NetworkAreaIPv4{
					DefaultNameservers: &[]string{
						"nameserver1",
						"nameserver2",
					},
					TransferNetwork:  utils.Ptr("network"),
					DefaultPrefixLen: utils.Ptr(int64(20)),
					MaxPrefixLen:     utils.Ptr(int64(22)),
					MinPrefixLen:     utils.Ptr(int64(18)),
				},
				Name: utils.Ptr("name"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			&[]iaas.NetworkRange{
				{
					NetworkRangeId: utils.Ptr(testRangeId1),
					Prefix:         utils.Ptr("prefix-1"),
				},
				{
					NetworkRangeId: utils.Ptr(testRangeId2),
					Prefix:         utils.Ptr("prefix-2"),
				},
			},
			Model{
				Id:             types.StringValue("oid,naid"),
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				Name:           types.StringValue("name"),
				DefaultNameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("nameserver1"),
					types.StringValue("nameserver2"),
				}),
				TransferNetwork:     types.StringValue("network"),
				DefaultPrefixLength: types.Int64Value(20),
				MaxPrefixLength:     types.Int64Value(22),
				MinPrefixLength:     types.Int64Value(18),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId1),
						"prefix":           types.StringValue("prefix-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId2),
						"prefix":           types.StringValue("prefix-2"),
					}),
				}),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			true,
		},
		{
			"model and response have ranges in different order",
			Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId1),
						"prefix":           types.StringValue("prefix-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId2),
						"prefix":           types.StringValue("prefix-2"),
					}),
				}),
			},
			&iaas.NetworkArea{
				AreaId: utils.Ptr("naid"),
				Ipv4: &iaas.NetworkAreaIPv4{
					DefaultNameservers: &[]string{
						"nameserver1",
						"nameserver2",
					},
					TransferNetwork:  utils.Ptr("network"),
					DefaultPrefixLen: utils.Ptr(int64(20)),
					MaxPrefixLen:     utils.Ptr(int64(22)),
					MinPrefixLen:     utils.Ptr(int64(18)),
				},
				Name: utils.Ptr("name"),
			},
			&[]iaas.NetworkRange{
				{
					NetworkRangeId: utils.Ptr(testRangeId2),
					Prefix:         utils.Ptr("prefix-2"),
				},
				{
					NetworkRangeId: utils.Ptr(testRangeId3),
					Prefix:         utils.Ptr("prefix-3"),
				},
				{
					NetworkRangeId: utils.Ptr(testRangeId1),
					Prefix:         utils.Ptr("prefix-1"),
				},
			},
			Model{
				Id:             types.StringValue("oid,naid"),
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				Name:           types.StringValue("name"),
				DefaultNameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("nameserver1"),
					types.StringValue("nameserver2"),
				}),
				TransferNetwork:     types.StringValue("network"),
				DefaultPrefixLength: types.Int64Value(20),
				MaxPrefixLength:     types.Int64Value(22),
				MinPrefixLength:     types.Int64Value(18),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId1),
						"prefix":           types.StringValue("prefix-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId2),
						"prefix":           types.StringValue("prefix-2"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId3),
						"prefix":           types.StringValue("prefix-3"),
					}),
				}),
				Labels: types.MapNull(types.StringType),
			},
			true,
		},
		{
			"default_nameservers_changed_outside_tf",
			Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				DefaultNameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId1),
						"prefix":           types.StringValue("prefix-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId2),
						"prefix":           types.StringValue("prefix-2"),
					}),
				}),
			},
			&iaas.NetworkArea{
				AreaId: utils.Ptr("naid"),
				Ipv4: &iaas.NetworkAreaIPv4{
					DefaultNameservers: &[]string{
						"ns2",
						"ns3",
					},
				},
			},
			&[]iaas.NetworkRange{
				{
					NetworkRangeId: utils.Ptr(testRangeId1),
					Prefix:         utils.Ptr("prefix-1"),
				},
				{
					NetworkRangeId: utils.Ptr(testRangeId2),
					Prefix:         utils.Ptr("prefix-2"),
				},
			},
			Model{
				Id:             types.StringValue("oid,naid"),
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				DefaultNameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns2"),
					types.StringValue("ns3"),
				}),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId1),
						"prefix":           types.StringValue("prefix-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId2),
						"prefix":           types.StringValue("prefix-2"),
					}),
				}),
				Labels: types.MapNull(types.StringType),
			},
			true,
		},
		{
			"network_ranges_changed_outside_tf",
			Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId1),
						"prefix":           types.StringValue("prefix-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId2),
						"prefix":           types.StringValue("prefix-2"),
					}),
				}),
			},
			&iaas.NetworkArea{
				AreaId: utils.Ptr("naid"),
				Ipv4:   &iaas.NetworkAreaIPv4{},
			},
			&[]iaas.NetworkRange{
				{
					NetworkRangeId: utils.Ptr(testRangeId2),
					Prefix:         utils.Ptr("prefix-2"),
				},
				{
					NetworkRangeId: utils.Ptr(testRangeId3),
					Prefix:         utils.Ptr("prefix-3"),
				},
			},
			Model{
				Id:                 types.StringValue("oid,naid"),
				OrganizationId:     types.StringValue("oid"),
				NetworkAreaId:      types.StringValue("naid"),
				DefaultNameservers: types.ListNull(types.StringType),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId2),
						"prefix":           types.StringValue("prefix-2"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringValue(testRangeId3),
						"prefix":           types.StringValue("prefix-3"),
					}),
				}),
				Labels: types.MapNull(types.StringType),
			},
			true,
		},
		{
			"nil_network_ranges_list",
			Model{},
			&iaas.NetworkArea{},
			nil,
			Model{},
			false,
		},
		{
			"response_nil_fail",
			Model{},
			nil,
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{
				OrganizationId: types.StringValue("oid"),
			},
			&iaas.NetworkArea{},
			&[]iaas.NetworkRange{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, tt.ListNetworkRanges, &tt.state)
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
		expected    *iaas.CreateNetworkAreaPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				DefaultNameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringUnknown(),
						"prefix":           types.StringValue("pr-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"network_range_id": types.StringUnknown(),
						"prefix":           types.StringValue("pr-2"),
					}),
				}),
				TransferNetwork:     types.StringValue("network"),
				DefaultPrefixLength: types.Int64Value(20),
				MaxPrefixLength:     types.Int64Value(22),
				MinPrefixLength:     types.Int64Value(18),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			&iaas.CreateNetworkAreaPayload{
				Name: utils.Ptr("name"),
				AddressFamily: &iaas.CreateAreaAddressFamily{
					Ipv4: &iaas.CreateAreaIPv4{
						DefaultNameservers: &[]string{
							"ns1",
							"ns2",
						},
						NetworkRanges: &[]iaas.NetworkRange{
							{
								Prefix: utils.Ptr("pr-1"),
							},
							{
								Prefix: utils.Ptr("pr-2"),
							},
						},
						TransferNetwork:  utils.Ptr("network"),
						DefaultPrefixLen: utils.Ptr(int64(20)),
						MaxPrefixLen:     utils.Ptr(int64(22)),
						MinPrefixLen:     utils.Ptr(int64(18)),
					},
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			true,
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
		expected    *iaas.PartialUpdateNetworkAreaPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				DefaultNameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns1"),
					types.StringValue("ns2"),
				}),
				DefaultPrefixLength: types.Int64Value(22),
				MaxPrefixLength:     types.Int64Value(24),
				MinPrefixLength:     types.Int64Value(20),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			&iaas.PartialUpdateNetworkAreaPayload{
				Name: utils.Ptr("name"),
				AddressFamily: &iaas.UpdateAreaAddressFamily{
					Ipv4: &iaas.UpdateAreaIPv4{
						DefaultNameservers: &[]string{
							"ns1",
							"ns2",
						},
						DefaultPrefixLen: utils.Ptr(int64(22)),
						MaxPrefixLen:     utils.Ptr(int64(24)),
						MinPrefixLen:     utils.Ptr(int64(20)),
					},
				},
				Labels: &map[string]interface{}{
					"key": "value",
				},
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
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestUpdateNetworkRanges(t *testing.T) {
	getAllNetworkRangesResp := iaas.NetworkRangeListResponse{
		Items: &[]iaas.NetworkRange{
			{
				Prefix:         utils.Ptr("pr-1"),
				NetworkRangeId: utils.Ptr(testRangeId1),
			},
			{
				Prefix:         utils.Ptr("pr-2"),
				NetworkRangeId: utils.Ptr(testRangeId2),
			},
			{
				Prefix:         utils.Ptr("pr-3"),
				NetworkRangeId: utils.Ptr(testRangeId3),
			},
			{
				Prefix:         utils.Ptr("pr-2"),
				NetworkRangeId: utils.Ptr(testRangeId2Repeated),
			},
		},
	}
	getAllNetworkRangesRespBytes, err := json.Marshal(getAllNetworkRangesResp)
	if err != nil {
		t.Fatalf("Failed to marshal get all network ranges response: %v", err)
	}

	// This is the response used whenever an API returns a failure response
	failureRespBytes := []byte("{\"message\": \"Something bad happened\"")

	tests := []struct {
		description                 string
		networkRanges               []networkRange
		ipv4                        []iaas.NetworkRange
		getAllNetworkRangesFails    bool
		createNetworkRangesFails    bool
		deleteNetworkRangesFails    bool
		isValid                     bool
		expectedNetworkRangesStates map[string]bool // Keys are prefix; value is true if prefix should exist at the end, false if should be deleted
	}{
		{
			description: "no_changes",
			networkRanges: []networkRange{
				{
					NetworkRangeId: types.StringValue(testRangeId1),
					Prefix:         types.StringValue("pr-1"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId2),
					Prefix:         types.StringValue("pr-2"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId3),
					Prefix:         types.StringValue("pr-3"),
				},
			},
			expectedNetworkRangesStates: map[string]bool{
				"pr-1": true,
				"pr-2": true,
				"pr-3": true,
			},
			isValid: true,
		},
		{
			description: "create_network_ranges",
			networkRanges: []networkRange{
				{
					NetworkRangeId: types.StringValue(testRangeId1),
					Prefix:         types.StringValue("pr-1"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId2),
					Prefix:         types.StringValue("pr-2"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId3),
					Prefix:         types.StringValue("pr-3"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId4),
					Prefix:         types.StringValue("pr-4"),
				},
			},
			expectedNetworkRangesStates: map[string]bool{
				"pr-1": true,
				"pr-2": true,
				"pr-3": true,
				"pr-4": true,
			},
			isValid: true,
		},
		{
			description: "delete_network_ranges",
			networkRanges: []networkRange{
				{
					NetworkRangeId: types.StringValue(testRangeId1),
					Prefix:         types.StringValue("pr-1"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId3),
					Prefix:         types.StringValue("pr-3"),
				},
			},
			expectedNetworkRangesStates: map[string]bool{
				"pr-1": true,
				"pr-2": false,
				"pr-3": true,
			},
			isValid: true,
		},
		{
			description: "multiple_changes",
			networkRanges: []networkRange{
				{
					NetworkRangeId: types.StringValue(testRangeId1),
					Prefix:         types.StringValue("pr-1"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId3),
					Prefix:         types.StringValue("pr-3"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId4),
					Prefix:         types.StringValue("pr-4"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId5),
					Prefix:         types.StringValue("pr-5"),
				},
			},
			expectedNetworkRangesStates: map[string]bool{
				"pr-1": true,
				"pr-2": false,
				"pr-3": true,
				"pr-4": true,
				"pr-5": true,
			},
			isValid: true,
		},
		{
			description: "multiple_changes_repetition",
			networkRanges: []networkRange{
				{
					NetworkRangeId: types.StringValue(testRangeId1),
					Prefix:         types.StringValue("pr-1"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId3),
					Prefix:         types.StringValue("pr-3"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId4),
					Prefix:         types.StringValue("pr-4"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId5),
					Prefix:         types.StringValue("pr-5"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId5),
					Prefix:         types.StringValue("pr-5"),
				},
			},
			expectedNetworkRangesStates: map[string]bool{
				"pr-1": true,
				"pr-2": false,
				"pr-3": true,
				"pr-4": true,
				"pr-5": true,
			},
			isValid: true,
		},
		{
			description: "multiple_changes_2",
			networkRanges: []networkRange{
				{
					NetworkRangeId: types.StringValue(testRangeId4),
					Prefix:         types.StringValue("pr-4"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId5),
					Prefix:         types.StringValue("pr-5"),
				},
			},
			expectedNetworkRangesStates: map[string]bool{
				"pr-1": false,
				"pr-2": false,
				"pr-3": false,
				"pr-4": true,
				"pr-5": true,
			},
			isValid: true,
		},
		{
			description:   "multiple_changes_3",
			networkRanges: []networkRange{},
			expectedNetworkRangesStates: map[string]bool{
				"pr-1": false,
				"pr-2": false,
				"pr-3": false,
			},
			isValid: true,
		},
		{
			description: "get_fails",
			networkRanges: []networkRange{
				{
					NetworkRangeId: types.StringValue(testRangeId1),
					Prefix:         types.StringValue("pr-1"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId2),
					Prefix:         types.StringValue("pr-2"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId3),
					Prefix:         types.StringValue("pr-3"),
				},
			},
			getAllNetworkRangesFails: true,
			isValid:                  false,
		},
		{
			description: "create_fails_1",
			networkRanges: []networkRange{
				{
					NetworkRangeId: types.StringValue(testRangeId1),
					Prefix:         types.StringValue("pr-1"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId2),
					Prefix:         types.StringValue("pr-2"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId3),
					Prefix:         types.StringValue("pr-3"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId4),
					Prefix:         types.StringValue("pr-4"),
				},
			},
			createNetworkRangesFails: true,
			isValid:                  false,
		},
		{
			description: "create_fails_2",
			networkRanges: []networkRange{
				{
					NetworkRangeId: types.StringValue(testRangeId1),
					Prefix:         types.StringValue("pr-1"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId2),
					Prefix:         types.StringValue("pr-2"),
				},
			},
			createNetworkRangesFails: true,
			expectedNetworkRangesStates: map[string]bool{
				"pr-1": true,
				"pr-2": true,
				"pr-3": false,
			},
			isValid: true,
		},
		{
			description: "delete_fails_1",
			networkRanges: []networkRange{
				{
					NetworkRangeId: types.StringValue(testRangeId1),
					Prefix:         types.StringValue("pr-1"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId2),
					Prefix:         types.StringValue("pr-2"),
				},
			},
			deleteNetworkRangesFails: true,
			isValid:                  false,
		},
		{
			description: "delete_fails_2",
			networkRanges: []networkRange{
				{
					NetworkRangeId: types.StringValue(testRangeId1),
					Prefix:         types.StringValue("pr-1"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId2),
					Prefix:         types.StringValue("pr-2"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId3),
					Prefix:         types.StringValue("pr-3"),
				},
				{
					NetworkRangeId: types.StringValue(testRangeId4),
					Prefix:         types.StringValue("pr-4"),
				},
			},
			deleteNetworkRangesFails: true,
			expectedNetworkRangesStates: map[string]bool{
				"pr-1": true,
				"pr-2": true,
				"pr-3": true,
				"pr-4": true,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			// Will be compared to tt.expectedNetworkRangesStates at the end
			networkRangesStates := make(map[string]bool)
			networkRangesStates["pr-1"] = true
			networkRangesStates["pr-2"] = true
			networkRangesStates["pr-3"] = true

			// Handler for getting all network ranges
			getAllNetworkRangesHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tt.getAllNetworkRangesFails {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write(failureRespBytes)
					if err != nil {
						t.Errorf("Get all network ranges handler: failed to write bad response: %v", err)
					}
					return
				}

				_, err := w.Write(getAllNetworkRangesRespBytes)
				if err != nil {
					t.Errorf("Get all network ranges handler: failed to write response: %v", err)
				}
			})

			// Handler for creating network range
			createNetworkRangeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				decoder := json.NewDecoder(r.Body)
				var payload iaas.CreateNetworkAreaRangePayload
				err := decoder.Decode(&payload)
				if err != nil {
					t.Errorf("Create network range handler: failed to parse payload")
					return
				}
				if payload.Ipv4 == nil {
					t.Errorf("Create network range handler: nil Ipv4")
					return
				}
				ipv4 := *payload.Ipv4

				for _, networkRange := range ipv4 {
					prefix := *networkRange.Prefix
					if prefixExists, prefixWasCreated := networkRangesStates[prefix]; prefixWasCreated && prefixExists {
						t.Errorf("Create network range handler: attempted to create range '%v' that already exists", *payload.Ipv4)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					if tt.createNetworkRangesFails {
						w.WriteHeader(http.StatusInternalServerError)
						_, err := w.Write(failureRespBytes)
						if err != nil {
							t.Errorf("Create network ranges handler: failed to write bad response: %v", err)
						}
						return
					}

					resp := iaas.NetworkRange{
						Prefix:         utils.Ptr("prefix"),
						NetworkRangeId: utils.Ptr("id-range"),
					}
					respBytes, err := json.Marshal(resp)
					if err != nil {
						t.Errorf("Create network range handler: failed to marshal response: %v", err)
						return
					}
					_, err = w.Write(respBytes)
					if err != nil {
						t.Errorf("Create network range handler: failed to write response: %v", err)
					}
					networkRangesStates[prefix] = true
				}
			})

			// Handler for deleting Network range
			deleteNetworkRangeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				networkRangeId, ok := vars["networkRangeId"]
				if !ok {
					t.Errorf("Delete network range handler: no range ID")
					return
				}

				var prefix string
				for _, rangeItem := range *getAllNetworkRangesResp.Items {
					if *rangeItem.NetworkRangeId == networkRangeId {
						prefix = *rangeItem.Prefix
					}
				}
				prefixExists, prefixWasCreated := networkRangesStates[prefix]
				if !prefixWasCreated {
					t.Errorf("Delete network range handler: attempted to delete range '%v' that wasn't created", prefix)
					return
				}
				if prefixWasCreated && !prefixExists {
					t.Errorf("Delete network range handler: attempted to delete range '%v' that was already deleted", prefix)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				if tt.deleteNetworkRangesFails {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write(failureRespBytes)
					if err != nil {
						t.Errorf("Delete network range handler: failed to write bad response: %v", err)
					}
					return
				}

				_, err = w.Write([]byte("{}"))
				if err != nil {
					t.Errorf("Delete network range handler: failed to write response: %v", err)
				}
				networkRangesStates[prefix] = false
			})

			// Setup server and client
			router := mux.NewRouter()
			router.HandleFunc("/v1/organizations/{organizationId}/network-areas/{areaId}/network-ranges", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					getAllNetworkRangesHandler(w, r)
				} else if r.Method == "POST" {
					createNetworkRangeHandler(w, r)
				}
			})
			router.HandleFunc("/v1/organizations/{organizationId}/network-areas/{areaId}/network-ranges/{networkRangeId}", deleteNetworkRangeHandler)
			mockedServer := httptest.NewServer(router)
			defer mockedServer.Close()
			client, err := iaas.NewAPIClient(
				config.WithEndpoint(mockedServer.URL),
				config.WithoutAuthentication(),
			)
			if err != nil {
				t.Fatalf("Failed to initialize client: %v", err)
			}

			// Run test
			err = updateNetworkRanges(context.Background(), testOrganizationId, testAreaId, tt.networkRanges, client)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(networkRangesStates, tt.expectedNetworkRangesStates)
				if diff != "" {
					t.Fatalf("Network range states do not match: %s", diff)
				}
			}
		})
	}
}
