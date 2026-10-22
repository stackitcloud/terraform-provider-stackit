package networks

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const (
	testRegion = "eu01"
)

func Test_mapFields(t *testing.T) {
	testProjectId := uuid.NewString()

	// Using pre-generated UUIDs since they are sorted inside the
	// tested func. Else no reproducible test result is possible.
	const (
		testNetwork01Id = "046bfa2c-4ca3-47b4-a979-63596c0e97e2"
		testNetwork02Id = "44aad1e1-f660-4408-9406-2fb31bbc23b9"
	)

	type params struct {
		networksResp *iaas.NetworkListResponse
		model        *NetworksModel
		region       string
	}

	tests := []struct {
		name    string
		params  params
		want    *NetworksModel
		isValid bool
	}{
		{
			name: "default",
			params: params{
				networksResp: &iaas.NetworkListResponse{
					Items: []iaas.Network{
						// Order is required here: network01 will be before network02 after mapping due to sorting
						{
							Id:   testNetwork02Id,
							Name: "network-02",
						},
						{
							Id:   testNetwork01Id,
							Name: "network-01",
						},
					},
				},
				model: &NetworksModel{
					ProjectId: types.StringValue(testProjectId),
				},
				region: testRegion,
			},
			want: &NetworksModel{
				Id:        utils.BuildInternalTerraformId(testProjectId, testRegion),
				ProjectId: types.StringValue(testProjectId),
				Region:    types.StringValue(testRegion),
				Results: types.ListValueMust(types.ObjectType{AttrTypes: networkItemModelTypes}, []attr.Value{
					types.ObjectValueMust(networkItemModelTypes, map[string]attr.Value{
						"network_id": types.StringValue(testNetwork01Id),
						"name":       types.StringValue("network-01"),
					}),
					types.ObjectValueMust(networkItemModelTypes, map[string]attr.Value{
						"network_id": types.StringValue(testNetwork02Id),
						"name":       types.StringValue("network-02"),
					}),
				}),
			},
			isValid: true,
		},
		{
			name: "response is nil",
			params: params{
				networksResp: nil,
				model:        &NetworksModel{},
				region:       testRegion,
			},
			isValid: false,
		},
		{
			name: "items in response is nil",
			params: params{
				networksResp: &iaas.NetworkListResponse{
					Items: nil,
				},
				model:  &NetworksModel{},
				region: testRegion,
			},
			isValid: false,
		},
		{
			name: "model is nil",
			params: params{
				networksResp: nil,
				model:        &NetworksModel{},
				region:       testRegion,
			},
			isValid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapFields(context.Background(), tt.params.networksResp, tt.params.model, tt.params.region)
			if !tt.isValid && err == nil {
				t.Fatalf("mapFields() should have failed")
			}

			if tt.isValid && err != nil {
				t.Fatalf("mapFields() should not have failed: %v", err)
			}

			if tt.isValid {
				if diff := cmp.Diff(tt.want, tt.params.model); diff != "" {
					t.Errorf("mapFields() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
