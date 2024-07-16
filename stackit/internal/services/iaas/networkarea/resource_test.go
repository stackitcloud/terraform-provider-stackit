package networkarea

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

var testOrganizationId = uuid.NewString()
var testAreaId = uuid.NewString()

func TestMapFields(t *testing.T) {
	tests := []struct {
		description               string
		state                     Model
		input                     *iaas.NetworkArea
		ListNetworkRangesResponse *iaas.NetworkRangeListResponse
		expected                  Model
		isValid                   bool
	}{
		{
			"id_ok",
			Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				//ProjectCount:   types.Int64Value(2),
			},
			&iaas.NetworkArea{
				AreaId:       utils.Ptr("naid"),
				Ipv4:         &iaas.NetworkAreaIPv4{},
				ProjectCount: utils.Ptr(int64(2)),
			},
			&iaas.NetworkRangeListResponse{
				Items: &[]iaas.NetworkRange{
					{
						//NetworkRangeId: utils.Ptr("range-1"),
						Prefix: utils.Ptr("prefix-1"),
					},
					{
						//NetworkRangeId: utils.Ptr("range-2"),
						Prefix: utils.Ptr("prefix-2"),
					},
				},
			},
			Model{
				Id:                  types.StringValue("oid,naid"),
				OrganizationId:      types.StringValue("oid"),
				NetworkAreaId:       types.StringValue("naid"),
				Name:                types.StringNull(),
				ProjectCount:        types.Int64Value(2),
				DefaultNameservers:  types.ListNull(types.StringType),
				TransferNetwork:     types.StringNull(),
				DefaultPrefixLength: types.Int64Null(),
				MaxPrefixLength:     types.Int64Null(),
				MinPrefixLength:     types.Int64Null(),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"prefix": types.StringValue("prefix-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"prefix": types.StringValue("prefix-2"),
					}),
				}),
			},
			true,
		},
		{
			"values_ok",
			Model{
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
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
				ProjectCount: utils.Ptr(int64(2)),
				Name:         utils.Ptr("name"),
			},
			&iaas.NetworkRangeListResponse{
				Items: &[]iaas.NetworkRange{
					{
						//NetworkRangeId: utils.Ptr("range-1"),
						Prefix: utils.Ptr("prefix-1"),
					},
					{
						//NetworkRangeId: utils.Ptr("range-2"),
						Prefix: utils.Ptr("prefix-2"),
					},
				},
			},
			Model{
				Id:             types.StringValue("oid,naid"),
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				Name:           types.StringValue("name"),
				ProjectCount:   types.Int64Value(2),
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
						"prefix": types.StringValue("prefix-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"prefix": types.StringValue("prefix-2"),
					}),
				}),
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
			},
			&iaas.NetworkArea{
				AreaId: utils.Ptr("naid"),
				Ipv4: &iaas.NetworkAreaIPv4{
					DefaultNameservers: &[]string{
						"ns2",
						"ns3",
					},
				},
				ProjectCount: utils.Ptr(int64(2)),
			},
			&iaas.NetworkRangeListResponse{
				Items: &[]iaas.NetworkRange{
					{
						Prefix: utils.Ptr("prefix-1"),
					},
					{
						Prefix: utils.Ptr("prefix-2"),
					},
				},
			},
			Model{
				Id:             types.StringValue("oid,naid"),
				OrganizationId: types.StringValue("oid"),
				NetworkAreaId:  types.StringValue("naid"),
				ProjectCount:   types.Int64Value(2),
				DefaultNameservers: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ns2"),
					types.StringValue("ns3"),
				}),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"prefix": types.StringValue("prefix-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"prefix": types.StringValue("prefix-2"),
					}),
				}),
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
						"prefix": types.StringValue("pr-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"prefix": types.StringValue("pr-2"),
					}),
				}),
			},
			&iaas.NetworkArea{
				AreaId:       utils.Ptr("naid"),
				Ipv4:         &iaas.NetworkAreaIPv4{},
				ProjectCount: utils.Ptr(int64(2)),
			},
			&iaas.NetworkRangeListResponse{
				Items: &[]iaas.NetworkRange{
					{
						Prefix: utils.Ptr("pr-2"),
					},
					{
						Prefix: utils.Ptr("pr-3"),
					},
				},
			},
			Model{
				Id:                 types.StringValue("oid,naid"),
				OrganizationId:     types.StringValue("oid"),
				NetworkAreaId:      types.StringValue("naid"),
				ProjectCount:       types.Int64Value(2),
				DefaultNameservers: types.ListNull(types.StringType),
				NetworkRanges: types.ListValueMust(types.ObjectType{AttrTypes: networkRangeTypes}, []attr.Value{
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"prefix": types.StringValue("pr-2"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"prefix": types.StringValue("pr-3"),
					}),
				}),
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
			&iaas.NetworkRangeListResponse{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, tt.ListNetworkRangesResponse, &tt.state)
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
						"prefix": types.StringValue("pr-1"),
					}),
					types.ObjectValueMust(networkRangeTypes, map[string]attr.Value{
						"prefix": types.StringValue("pr-2"),
					}),
				}),
				TransferNetwork:     types.StringValue("network"),
				DefaultPrefixLength: types.Int64Value(20),
				MaxPrefixLength:     types.Int64Value(22),
				MinPrefixLength:     types.Int64Value(18),
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
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input)
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
				Prefix:         utils.Ptr("acl-1"),
				NetworkRangeId: utils.Ptr("id-acl-1"),
			},
			{
				Prefix:         utils.Ptr("acl-2"),
				NetworkRangeId: utils.Ptr("id-acl-2"),
			},
			{
				Prefix:         utils.Ptr("acl-3"),
				NetworkRangeId: utils.Ptr("id-acl-3"),
			},
			{
				Prefix:         utils.Ptr("acl-2"),
				NetworkRangeId: utils.Ptr("id-acl-2-repeated"),
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
		ipv4                        []iaas.NetworkRange
		getAllNetworkRangesFails    bool
		createNetworkRangesFails    bool
		deleteNetworkRangesFails    bool
		isValid                     bool
		expectedNetworkRangesStates map[string]bool // Keys are prefix; value is true if prefix should exist at the end, false if should be deleted
	}{
		{
			description: "no_changes",
			ipv4: []iaas.NetworkRange{
				{
					Prefix: utils.Ptr("pr-1"),
				},
				{
					Prefix: utils.Ptr("pr-2"),
				},
			},
			expectedNetworkRangesStates: map[string]bool{
				"pr-1": true,
				"pr-2": true,
			},
			isValid: true,
		},
		{
			description: "create_network_ranges",
			ipv4: []iaas.NetworkRange{
				{
					Prefix: utils.Ptr("pr-1"),
				},
				{
					Prefix: utils.Ptr("pr-2"),
				},
				{
					Prefix: utils.Ptr("pr-3"),
				},
				{
					Prefix: utils.Ptr("pr-4"),
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
			ipv4: []iaas.NetworkRange{
				{
					Prefix: utils.Ptr("pr-3"),
				},
				{
					Prefix: utils.Ptr("pr-1"),
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
			ipv4: []iaas.NetworkRange{
				{
					Prefix: utils.Ptr("pr-4"),
				},
				{
					Prefix: utils.Ptr("pr-3"),
				},
				{
					Prefix: utils.Ptr("pr-1"),
				},
				{
					Prefix: utils.Ptr("pr-5"),
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
			ipv4: []iaas.NetworkRange{
				{
					Prefix: utils.Ptr("pr-4"),
				},
				{
					Prefix: utils.Ptr("pr-3"),
				},
				{
					Prefix: utils.Ptr("pr-1"),
				},
				{
					Prefix: utils.Ptr("pr-5"),
				},
				{
					Prefix: utils.Ptr("pr-5"),
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
			ipv4: []iaas.NetworkRange{
				{
					Prefix: utils.Ptr("pr-4"),
				},
				{
					Prefix: utils.Ptr("pr-5"),
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
			description: "multiple_changes_3",
			ipv4:        []iaas.NetworkRange{},
			expectedNetworkRangesStates: map[string]bool{
				"pr-1": false,
				"pr-2": false,
				"pr-3": false,
			},
			isValid: true,
		},
		{
			description: "get_fails",
			ipv4: []iaas.NetworkRange{
				{
					Prefix: utils.Ptr("pr-1"),
				},
				{
					Prefix: utils.Ptr("pr-2"),
				},
				{
					Prefix: utils.Ptr("pr-3"),
				},
			},
			getAllNetworkRangesFails: true,
			isValid:                  false,
		},
		{
			description: "create_fails_1",
			ipv4: []iaas.NetworkRange{
				{
					Prefix: utils.Ptr("pr-1"),
				},
				{
					Prefix: utils.Ptr("pr-2"),
				},
				{
					Prefix: utils.Ptr("pr-3"),
				},
				{
					Prefix: utils.Ptr("pr-4"),
				},
			},
			createNetworkRangesFails: true,
			isValid:                  false,
		},
		{
			description: "create_fails_2",
			ipv4: []iaas.NetworkRange{
				{
					Prefix: utils.Ptr("pr-1"),
				},
				{
					Prefix: utils.Ptr("pr-2"),
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
			ipv4: []iaas.NetworkRange{
				{
					Prefix: utils.Ptr("pr-1"),
				},
				{
					Prefix: utils.Ptr("pr-2"),
				},
			},
			deleteNetworkRangesFails: true,
			isValid:                  false,
		},
		{
			description: "delete_fails_2",
			ipv4: []iaas.NetworkRange{
				{
					Prefix: utils.Ptr("pr-1"),
				},
				{
					Prefix: utils.Ptr("pr-2"),
				},
				{
					Prefix: utils.Ptr("pr-3"),
				},
				{
					Prefix: utils.Ptr("pr-4"),
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
			// Will be compared to tt.expectedACLsStates at the end
			ipv4States := make(map[*[]iaas.NetworkRange]bool)
			ipv4States[&[]iaas.NetworkRange{
				{
					NetworkRangeId: utils.Ptr("uuid"),
					Prefix:         utils.Ptr("pr-1"),
				},
			}] = true
			ipv4States[&[]iaas.NetworkRange{
				{
					NetworkRangeId: utils.Ptr("uuid2"),
					Prefix:         utils.Ptr("pr-2"),
				},
			}] = true
			ipv4States[&[]iaas.NetworkRange{
				{
					NetworkRangeId: utils.Ptr("uuid3"),
					Prefix:         utils.Ptr("pr-3"),
				},
			}] = true

			// Handler for getting all ACLs
			getAllNetworkRangesHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			// Handler for creating ACL
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
				if ipv4Exists, ipv4WasCreated := ipv4States[&ipv4]; ipv4WasCreated && ipv4Exists {
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
					NetworkRangeId: utils.Ptr(fmt.Sprintf("id-range")),
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
				ipv4States[&ipv4] = true
			})

			// Handler for deleting Network range
			deleteNetworkRangeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				networkRange := []iaas.NetworkRange{{
					NetworkRangeId: utils.Ptr("uuid6"),
					Prefix:         utils.Ptr("pr-6"),
				},
				}

				ipv4Exists, ipv4WasCreated := ipv4States[&networkRange]
				if !ipv4WasCreated {
					t.Errorf("Delete network range handler: attempted to delete range '%v' that wasn't created", networkRange)
					return
				}
				if ipv4WasCreated && !ipv4Exists {
					t.Errorf("Delete network range handler: attempted to delete range '%v' that was already deleted", networkRange)
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
				ipv4States[&networkRange] = false
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
			err = updateNetworkRanges(context.Background(), testOrganizationId, testAreaId, tt.ipv4, client)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(ipv4States, tt.expectedNetworkRangesStates)
				if diff != "" {
					t.Fatalf("Network range states do not match: %s", diff)
				}
			}
		})
	}
}
