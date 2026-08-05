package waf

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	albWaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1api"
)

const (
	testRegion = "eu01"

	testManagedRuleSetName  = "mrs_name"
	testCustomRuleGroupName = "crg_name"
)

func Test_mapFields(t *testing.T) {
	fixtureModel := func(mods ...func(*Model)) *Model {
		m := Model{
			Id:        types.StringValue(fmt.Sprintf("pid,%s,name", testRegion)),
			ProjectId: types.StringValue("pid"),
			Region:    types.StringValue(testRegion),
			Name:      types.StringValue("name"),
			Labels:    types.MapNull(types.StringType),
		}

		for _, mod := range mods {
			mod(&m)
		}
		return &m
	}
	tests := []struct {
		name    string
		input   *albWaf.GetWAFResponse
		state   *Model
		region  string
		want    *Model
		wantErr bool
	}{
		{
			name: "default_values",
			input: &albWaf.GetWAFResponse{
				Name: "name",
			},
			state:   fixtureModel(),
			region:  testRegion,
			want:    fixtureModel(),
			wantErr: false,
		},
		{
			name: "simple values",
			input: &albWaf.GetWAFResponse{
				Name:                "name",
				Labels:              &map[string]string{"label1": "value1"},
				ManagedRuleSetName:  new(testManagedRuleSetName),
				CustomRuleGroupName: new(testCustomRuleGroupName),
			},
			state:  fixtureModel(),
			region: testRegion,
			want: fixtureModel(
				func(m *Model) {
					m.ManagedRuleSetName = types.StringValue(testManagedRuleSetName)
					m.CustomRuleGroupName = types.StringValue(testCustomRuleGroupName)
					m.Labels = types.MapValueMust(types.StringType, map[string]attr.Value{
						"label1": types.StringValue("value1"),
					})
				},
			),
			wantErr: false,
		},
		{
			name:    "fails when model is nil",
			state:   nil,
			wantErr: true,
		},
		{
			name:    "fails when input is nil",
			input:   nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := mapFields(t.Context(), tt.input, tt.state, tt.region)

			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("mapFieds() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("mapFieds() succeeded unexpectedly")
			}
			if !tt.wantErr {
				diff := cmp.Diff(tt.state, tt.want)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func Test_toCreatePayload(t *testing.T) {
	tests := []struct {
		name    string
		model   *Model
		want    *albWaf.CreateWAFPayload
		wantErr bool
	}{
		{
			name: "basic values",
			model: &Model{
				ManagedRuleSetName:  types.StringValue("example"),
				CustomRuleGroupName: types.StringValue("example group name"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"label1": types.StringValue("value1"),
				}),
			},
			want: &albWaf.CreateWAFPayload{
				ManagedRuleSetName:  new("example"),
				CustomRuleGroupName: new("example group name"),
				Labels:              &map[string]string{"label1": "value1"},
			},
			wantErr: false,
		},
		{
			name:    "fails when model is nil",
			model:   nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := toCreatePayload(t.Context(), tt.model)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("toCreatePayload() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("toCreatePayload() succeeded unexpectedly")
			}
			diff := cmp.Diff(got, tt.want)
			if diff != "" {
				t.Errorf("Data does not match: %s", diff)
			}
		})
	}
}
