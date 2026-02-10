package networkarearegion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stackitcloud/stackit-sdk-go/core/config"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"

	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

const (
	testRegion = "eu01"
)

var (
	organizationId = uuid.NewString()
	networkAreaId  = uuid.NewString()

	networkRangeId1         = uuid.NewString()
	networkRangeId2         = uuid.NewString()
	networkRangeId3         = uuid.NewString()
	networkRangeId4         = uuid.NewString()
	networkRangeId5         = uuid.NewString()
	networkRangeId2Repeated = uuid.NewString()
)

func Test_mapFields(t *testing.T) {
	type args struct {
		networkAreaRegion *iaas.RegionalArea
		model             *Model
		region            string
	}
	tests := []struct {
		name    string
		args    args
		want    *Model
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				model: &Model{
					OrganizationId: types.StringValue(organizationId),
					NetworkAreaId:  types.StringValue(networkAreaId),
					Ipv4:           &ipv4Model{},
				},
				networkAreaRegion: &iaas.RegionalArea{
					Ipv4: &iaas.RegionalAreaIPv4{
						DefaultNameservers: &[]string{
							"nameserver1",
							"nameserver2",
						},
						TransferNetwork:  utils.Ptr("network"),
						DefaultPrefixLen: utils.Ptr(int64(20)),
						MaxPrefixLen:     utils.Ptr(int64(22)),
						MinPrefixLen:     utils.Ptr(int64(18)),
						NetworkRanges: &[]iaas.NetworkRange{
							{
								Id:     utils.Ptr(networkRangeId1),
								Prefix: utils.Ptr("prefix-1"),
							},
							{
								Id:     utils.Ptr(networkRangeId2),
								Prefix: utils.Ptr("prefix-2"),
							},
						},
					},
				},
				region: "eu01",
			},
			want: &Model{
				Id:             types.StringValue(fmt.Sprintf("%s,%s,eu01", organizationId, networkAreaId)),
				OrganizationId: types.StringValue(organizationId),
				NetworkAreaId:  types.StringValue(networkAreaId),
				Region:         types.StringValue("eu01"),

				Ipv4: &ipv4Model{
					DefaultNameservers: types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("nameserver1"),
						types.StringValue("nameserver2"),
					}),
					TransferNetwork:     types.StringValue("network"),
					DefaultPrefixLength: types.Int64Value(20),
					MaxPrefixLength:     types.Int64Value(22),
					MinPrefixLength:     types.Int64Value(18),
					NetworkRanges: []networkRangeModel{
						{
							NetworkRangeId: types.StringValue(networkRangeId1),
							Prefix:         types.StringValue("prefix-1"),
						},
						{
							NetworkRangeId: types.StringValue(networkRangeId2),
							Prefix:         types.StringValue("prefix-2"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "model is nil",
			args: args{
				model:             nil,
				networkAreaRegion: &iaas.RegionalArea{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "model.Ipv4 is nil",
			args: args{
				model: &Model{
					OrganizationId: types.StringValue(organizationId),
					NetworkAreaId:  types.StringValue(networkAreaId),
					Ipv4:           nil,
				},
				networkAreaRegion: &iaas.RegionalArea{},
				region:            "eu01",
			},
			want: &Model{
				Id:             types.StringValue(fmt.Sprintf("%s,%s,eu01", organizationId, networkAreaId)),
				OrganizationId: types.StringValue(organizationId),
				NetworkAreaId:  types.StringValue(networkAreaId),
				Region:         types.StringValue("eu01"),
				Ipv4: &ipv4Model{
					DefaultNameservers: types.ListNull(types.StringType),
				},
			},
			wantErr: false,
		},
		{
			name: "network area region response is nil",
			args: args{
				model: &Model{
					Ipv4: &ipv4Model{
						DefaultNameservers: types.ListNull(types.StringType),
						NetworkRanges:      []networkRangeModel{},
					},
				},
			},
			want: &Model{
				Ipv4: &ipv4Model{
					DefaultNameservers: types.ListNull(types.StringType),
					NetworkRanges:      []networkRangeModel{},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := mapFields(ctx, tt.args.networkAreaRegion, tt.args.model, tt.args.region); (err != nil) != tt.wantErr {
				t.Errorf("mapFields() error = %v, wantErr %v", err, tt.wantErr)
			}
			diff := cmp.Diff(tt.args.model, tt.want)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}

func Test_toCreatePayload(t *testing.T) {
	type args struct {
		model *Model
	}
	tests := []struct {
		name    string
		args    args
		want    *iaas.CreateNetworkAreaRegionPayload
		wantErr bool
	}{
		{
			name: "default_ok",
			args: args{
				model: &Model{
					Ipv4: &ipv4Model{
						DefaultNameservers: types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("ns1"),
							types.StringValue("ns2"),
						}),
						NetworkRanges: []networkRangeModel{
							{
								NetworkRangeId: types.StringUnknown(),
								Prefix:         types.StringValue("pr-1"),
							},
							{
								NetworkRangeId: types.StringUnknown(),
								Prefix:         types.StringValue("pr-2"),
							},
						},
						TransferNetwork:     types.StringValue("network"),
						DefaultPrefixLength: types.Int64Value(20),
						MaxPrefixLength:     types.Int64Value(22),
						MinPrefixLength:     types.Int64Value(18),
					},
				},
			},
			want: &iaas.CreateNetworkAreaRegionPayload{
				Ipv4: &iaas.RegionalAreaIPv4{
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
		{
			name: "model is nil",
			args: args{
				model: nil,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toCreatePayload(context.Background(), tt.args.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toCreatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			diff := cmp.Diff(got, tt.want)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}

func Test_toUpdatePayload(t *testing.T) {
	type args struct {
		model *Model
	}
	tests := []struct {
		name    string
		args    args
		want    *iaas.UpdateNetworkAreaRegionPayload
		wantErr bool
	}{
		{
			name: "default_ok",
			args: args{
				model: &Model{
					Ipv4: &ipv4Model{
						DefaultNameservers: types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("ns1"),
							types.StringValue("ns2"),
						}),
						DefaultPrefixLength: types.Int64Value(22),
						MaxPrefixLength:     types.Int64Value(24),
						MinPrefixLength:     types.Int64Value(20),
					},
				},
			},
			want: &iaas.UpdateNetworkAreaRegionPayload{
				Ipv4: &iaas.UpdateRegionalAreaIPv4{
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
		{
			name: "model is nil",
			args: args{
				model: nil,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toUpdatePayload(context.Background(), tt.args.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toUpdatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			diff := cmp.Diff(got, tt.want)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}

func Test_mapIpv4NetworkRanges(t *testing.T) {
	type args struct {
		networkAreaRangesList *[]iaas.NetworkRange
		model                 *Model
	}
	tests := []struct {
		name    string
		args    args
		want    *Model
		wantErr bool
	}{
		{
			name: "model and response have ranges in different order",
			args: args{
				model: &Model{
					OrganizationId: types.StringValue(organizationId),
					NetworkAreaId:  types.StringValue(networkAreaId),
					Ipv4: &ipv4Model{
						DefaultNameservers: types.ListNull(types.StringType),
						NetworkRanges: []networkRangeModel{
							{
								NetworkRangeId: types.StringValue(networkRangeId1),
								Prefix:         types.StringValue("prefix-1"),
							},
							{
								NetworkRangeId: types.StringValue(networkRangeId2),
								Prefix:         types.StringValue("prefix-2"),
							},
						},
					},
				},
				networkAreaRangesList: &[]iaas.NetworkRange{
					{
						Id:     utils.Ptr(networkRangeId2),
						Prefix: utils.Ptr("prefix-2"),
					},
					{
						Id:     utils.Ptr(networkRangeId3),
						Prefix: utils.Ptr("prefix-3"),
					},
					{
						Id:     utils.Ptr(networkRangeId1),
						Prefix: utils.Ptr("prefix-1"),
					},
				},
			},
			want: &Model{
				OrganizationId: types.StringValue(organizationId),
				NetworkAreaId:  types.StringValue(networkAreaId),
				Ipv4: &ipv4Model{
					NetworkRanges: []networkRangeModel{
						{
							NetworkRangeId: types.StringValue(networkRangeId1),
							Prefix:         types.StringValue("prefix-1"),
						},
						{
							NetworkRangeId: types.StringValue(networkRangeId2),
							Prefix:         types.StringValue("prefix-2"),
						},
						{
							NetworkRangeId: types.StringValue(networkRangeId3),
							Prefix:         types.StringValue("prefix-3"),
						},
					},
					DefaultNameservers: types.ListNull(types.StringType),
				},
			},
			wantErr: false,
		},
		{
			name: "network_ranges_changed_outside_tf",
			args: args{
				model: &Model{
					OrganizationId: types.StringValue(organizationId),
					NetworkAreaId:  types.StringValue(networkAreaId),
					Ipv4: &ipv4Model{
						NetworkRanges: []networkRangeModel{
							{
								NetworkRangeId: types.StringValue(networkRangeId1),
								Prefix:         types.StringValue("prefix-1"),
							},
							{
								NetworkRangeId: types.StringValue(networkRangeId2),
								Prefix:         types.StringValue("prefix-2"),
							},
						},
						DefaultNameservers: types.ListNull(types.StringType),
					},
				},
				networkAreaRangesList: &[]iaas.NetworkRange{
					{
						Id:     utils.Ptr(networkRangeId2),
						Prefix: utils.Ptr("prefix-2"),
					},
					{
						Id:     utils.Ptr(networkRangeId3),
						Prefix: utils.Ptr("prefix-3"),
					},
				},
			},
			want: &Model{
				OrganizationId: types.StringValue(organizationId),
				NetworkAreaId:  types.StringValue(networkAreaId),
				Ipv4: &ipv4Model{
					NetworkRanges: []networkRangeModel{
						{
							NetworkRangeId: types.StringValue(networkRangeId2),
							Prefix:         types.StringValue("prefix-2"),
						},
						{
							NetworkRangeId: types.StringValue(networkRangeId3),
							Prefix:         types.StringValue("prefix-3"),
						},
					},
					DefaultNameservers: types.ListNull(types.StringType),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mapIpv4NetworkRanges(context.Background(), tt.args.networkAreaRangesList, tt.args.model); (err != nil) != tt.wantErr {
				t.Errorf("mapIpv4NetworkRanges() error = %v, wantErr %v", err, tt.wantErr)
			}
			diff := cmp.Diff(tt.args.model, tt.want)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}

func Test_updateIpv4NetworkRanges(t *testing.T) {
	getAllNetworkRangesResp := iaas.NetworkRangeListResponse{
		Items: &[]iaas.NetworkRange{
			{
				Prefix: utils.Ptr("pr-1"),
				Id:     utils.Ptr(networkRangeId1),
			},
			{
				Prefix: utils.Ptr("pr-2"),
				Id:     utils.Ptr(networkRangeId2),
			},
			{
				Prefix: utils.Ptr("pr-3"),
				Id:     utils.Ptr(networkRangeId3),
			},
			{
				Prefix: utils.Ptr("pr-2"),
				Id:     utils.Ptr(networkRangeId2Repeated),
			},
		},
	}
	getAllNetworkRangesRespBytes, err := json.Marshal(getAllNetworkRangesResp)
	if err != nil {
		t.Fatalf("Failed to marshal get all network ranges response: %v", err)
	}

	// This is the response used whenever an API returns a failure response
	failureRespBytes := []byte("{\"message\": \"Something bad happened\"")

	type args struct {
		networkRanges []networkRangeModel
	}
	tests := []struct {
		description string
		args        args

		expectedNetworkRangesStates map[string]bool // Keys are prefix; value is true if prefix should exist at the end, false if should be deleted
		isValid                     bool

		// mock control
		createNetworkRangesFails bool
		deleteNetworkRangesFails bool
		getAllNetworkRangesFails bool
	}{
		{
			description: "no_changes",
			args: args{
				networkRanges: []networkRangeModel{
					{
						NetworkRangeId: types.StringValue(networkRangeId1),
						Prefix:         types.StringValue("pr-1"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId2),
						Prefix:         types.StringValue("pr-2"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId3),
						Prefix:         types.StringValue("pr-3"),
					},
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
			args: args{
				networkRanges: []networkRangeModel{
					{
						NetworkRangeId: types.StringValue(networkRangeId1),
						Prefix:         types.StringValue("pr-1"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId2),
						Prefix:         types.StringValue("pr-2"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId3),
						Prefix:         types.StringValue("pr-3"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId4),
						Prefix:         types.StringValue("pr-4"),
					},
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
			args: args{
				networkRanges: []networkRangeModel{
					{
						NetworkRangeId: types.StringValue(networkRangeId1),
						Prefix:         types.StringValue("pr-1"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId3),
						Prefix:         types.StringValue("pr-3"),
					},
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
			args: args{
				networkRanges: []networkRangeModel{
					{
						NetworkRangeId: types.StringValue(networkRangeId1),
						Prefix:         types.StringValue("pr-1"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId3),
						Prefix:         types.StringValue("pr-3"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId4),
						Prefix:         types.StringValue("pr-4"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId5),
						Prefix:         types.StringValue("pr-5"),
					},
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
			args: args{
				networkRanges: []networkRangeModel{
					{
						NetworkRangeId: types.StringValue(networkRangeId1),
						Prefix:         types.StringValue("pr-1"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId3),
						Prefix:         types.StringValue("pr-3"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId4),
						Prefix:         types.StringValue("pr-4"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId5),
						Prefix:         types.StringValue("pr-5"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId5),
						Prefix:         types.StringValue("pr-5"),
					},
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
			args: args{
				networkRanges: []networkRangeModel{
					{
						NetworkRangeId: types.StringValue(networkRangeId4),
						Prefix:         types.StringValue("pr-4"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId5),
						Prefix:         types.StringValue("pr-5"),
					},
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
			args: args{
				networkRanges: []networkRangeModel{},
			},
			expectedNetworkRangesStates: map[string]bool{
				"pr-1": false,
				"pr-2": false,
				"pr-3": false,
			},
			isValid: true,
		},
		{
			description: "get_fails",
			args: args{
				networkRanges: []networkRangeModel{
					{
						NetworkRangeId: types.StringValue(networkRangeId1),
						Prefix:         types.StringValue("pr-1"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId2),
						Prefix:         types.StringValue("pr-2"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId3),
						Prefix:         types.StringValue("pr-3"),
					},
				},
			},
			getAllNetworkRangesFails: true,
			isValid:                  false,
		},
		{
			description: "create_fails_1",
			args: args{
				networkRanges: []networkRangeModel{
					{
						NetworkRangeId: types.StringValue(networkRangeId1),
						Prefix:         types.StringValue("pr-1"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId2),
						Prefix:         types.StringValue("pr-2"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId3),
						Prefix:         types.StringValue("pr-3"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId4),
						Prefix:         types.StringValue("pr-4"),
					},
				},
			},
			createNetworkRangesFails: true,
			isValid:                  false,
		},
		{
			description: "create_fails_2",
			args: args{
				networkRanges: []networkRangeModel{
					{
						NetworkRangeId: types.StringValue(networkRangeId1),
						Prefix:         types.StringValue("pr-1"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId2),
						Prefix:         types.StringValue("pr-2"),
					},
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
			args: args{
				networkRanges: []networkRangeModel{
					{
						NetworkRangeId: types.StringValue(networkRangeId1),
						Prefix:         types.StringValue("pr-1"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId2),
						Prefix:         types.StringValue("pr-2"),
					},
				},
			},
			deleteNetworkRangesFails: true,
			isValid:                  false,
		},
		{
			description: "delete_fails_2",
			args: args{
				networkRanges: []networkRangeModel{
					{
						NetworkRangeId: types.StringValue(networkRangeId1),
						Prefix:         types.StringValue("pr-1"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId2),
						Prefix:         types.StringValue("pr-2"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId3),
						Prefix:         types.StringValue("pr-3"),
					},
					{
						NetworkRangeId: types.StringValue(networkRangeId4),
						Prefix:         types.StringValue("pr-4"),
					},
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
						Prefix: utils.Ptr("prefix"),
						Id:     utils.Ptr("id-range"),
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
					if *rangeItem.Id == networkRangeId {
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
			router.HandleFunc("/v2/organizations/{organizationId}/network-areas/{areaId}/regions/{region}/network-ranges", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					getAllNetworkRangesHandler(w, r)
				} else if r.Method == "POST" {
					createNetworkRangeHandler(w, r)
				}
			})
			router.HandleFunc("/v2/organizations/{organizationId}/network-areas/{areaId}/regions/{region}/network-ranges/{networkRangeId}", deleteNetworkRangeHandler)
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
			err = updateIpv4NetworkRanges(context.Background(), organizationId, networkAreaId, tt.args.networkRanges, client, testRegion)
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

func Test_toDefaultNameserversPayload(t *testing.T) {
	type args struct {
		model *Model
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "values_ok",
			args: args{
				model: &Model{
					Ipv4: &ipv4Model{
						DefaultNameservers: types.ListValueMust(types.StringType, []attr.Value{
							types.StringValue("1.1.1.1"),
							types.StringValue("8.8.8.8"),
							types.StringValue("9.9.9.9"),
						}),
					},
				},
			},
			want: []string{
				"1.1.1.1",
				"8.8.8.8",
				"9.9.9.9",
			},
		},
		{
			name: "model is nil",
			args: args{
				model: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toDefaultNameserversPayload(context.Background(), tt.args.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toDefaultNameserversPayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toDefaultNameserversPayload() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toNetworkRangesPayload(t *testing.T) {
	type args struct {
		model *Model
	}
	tests := []struct {
		name    string
		args    args
		want    *[]iaas.NetworkRange
		wantErr bool
	}{
		{
			name: "values_ok",
			args: args{
				model: &Model{
					Ipv4: &ipv4Model{
						NetworkRanges: []networkRangeModel{
							{
								Prefix: types.StringValue("prefix-1"),
							},
							{
								Prefix: types.StringValue("prefix-2"),
							},
						},
					},
				},
			},
			want: &[]iaas.NetworkRange{
				{
					Prefix: utils.Ptr("prefix-1"),
				},
				{
					Prefix: utils.Ptr("prefix-2"),
				},
			},
		},
		{
			name: "model is nil",
			args: args{
				model: nil,
			},
			wantErr: true,
		},
		{
			name: "network ranges is nil",
			args: args{
				model: &Model{
					Ipv4: &ipv4Model{
						NetworkRanges: nil,
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "network ranges has length 0",
			args: args{
				model: &Model{
					Ipv4: &ipv4Model{
						NetworkRanges: []networkRangeModel{},
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toNetworkRangesPayload(context.Background(), tt.args.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toNetworkRangesPayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			diff := cmp.Diff(got, tt.want)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}
