package bgpfilterrule

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
)

var (
	projectId = uuid.NewString()
	gatewayId = uuid.NewString()
	filterId  = uuid.NewString()
	region    = "eu01"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *vpn.BGPFilterRule
		expected    Model
		isValid     bool
	}{
		{
			description: "minimal_rule",
			state: Model{
				ProjectId: types.StringValue(projectId),
				GatewayId: types.StringValue(gatewayId),
				FilterId:  types.StringValue(filterId),
			},
			input: &vpn.BGPFilterRule{
				Id:     new("rule-id"),
				Action: vpn.BGPFILTERRULEACTION_PERMIT,
			},
			expected: Model{
				Id:        types.StringValue(fmt.Sprintf("%s,%s,%s,%s,%s", projectId, region, gatewayId, filterId, "rule-id")),
				ProjectId: types.StringValue(projectId),
				Region:    types.StringValue(region),
				GatewayId: types.StringValue(gatewayId),
				FilterId:  types.StringValue(filterId),
				RuleId:    types.StringValue("rule-id"),
				Action:    types.StringValue("PERMIT"),
				Sequence:  types.Int32Null(),
			},
			isValid: true,
		},
		{
			description: "full_rule",
			state: Model{
				ProjectId: types.StringValue(projectId),
				GatewayId: types.StringValue(gatewayId),
				FilterId:  types.StringValue(filterId),
			},
			input: &vpn.BGPFilterRule{
				Id:       new("rule-id"),
				Action:   vpn.BGPFILTERRULEACTION_PERMIT,
				Sequence: new(int32(10)),
				Match: &vpn.BGPFilterRuleMatch{
					AsPathContainsAny: []int64{65001, 65002},
					Communities:       []string{"65000:100"},
					FirstASN:          new(int64(65001)),
					MaxPrefixLength:   new(int32(24)),
					MinPrefixLength:   new(int32(16)),
					Peer:              new("192.0.2.1"),
					Prefixes:          []string{"10.0.0.0/16"},
				},
				Set: &vpn.BGPFilterRuleSet{
					LocalPreference: new(int32(150)),
				},
			},
			expected: Model{
				Id:        types.StringValue(fmt.Sprintf("%s,%s,%s,%s,%s", projectId, region, gatewayId, filterId, "rule-id")),
				ProjectId: types.StringValue(projectId),
				Region:    types.StringValue(region),
				GatewayId: types.StringValue(gatewayId),
				FilterId:  types.StringValue(filterId),
				RuleId:    types.StringValue("rule-id"),
				Action:    types.StringValue("PERMIT"),
				Sequence:  types.Int32Value(10),
				Match: &MatchModel{
					AsPathContainsAny: types.ListValueMust(types.Int64Type, []attr.Value{
						types.Int64Value(65001),
						types.Int64Value(65002),
					}),
					Communities: types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("65000:100"),
					}),
					FirstAsn:        types.Int64Value(65001),
					MaxPrefixLength: types.Int32Value(24),
					MinPrefixLength: types.Int32Value(16),
					Peer:            types.StringValue("192.0.2.1"),
					Prefixes: types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("10.0.0.0/16"),
					}),
				},
				Set: &SetModel{
					LocalPreference: types.Int32Value(150),
				},
			},
			isValid: true,
		},
		{
			description: "nil_response",
			state:       Model{},
			input:       nil,
			expected:    Model{},
			isValid:     false,
		},
		{
			description: "nil_rule_id",
			state: Model{
				ProjectId: types.StringValue(projectId),
			},
			input: &vpn.BGPFilterRule{
				Id:     nil,
				Action: vpn.BGPFILTERRULEACTION_DENY,
			},
			expected: Model{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := tt.state
			err := mapFields(context.Background(), tt.input, &state, region)

			if !tt.isValid && err == nil {
				t.Fatalf("expected error, got none")
			}
			if tt.isValid && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.isValid {
				if diff := cmp.Diff(tt.expected, state); diff != "" {
					t.Fatalf("Data mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *vpn.CreateGatewayBGPFilterRulePayload
	}{
		{
			description: "minimal - sequence omitted",
			input: &Model{
				Action:   types.StringValue("PERMIT"),
				Sequence: types.Int32Null(),
			},
			expected: &vpn.CreateGatewayBGPFilterRulePayload{
				Action: vpn.CREATEGATEWAYBGPFILTERRULEPAYLOADACTION_PERMIT,
			},
		},
		{
			description: "sequence set explicitly",
			input: &Model{
				Action:   types.StringValue("DENY"),
				Sequence: types.Int32Value(20),
			},
			expected: &vpn.CreateGatewayBGPFilterRulePayload{
				Action:   vpn.CREATEGATEWAYBGPFILTERRULEPAYLOADACTION_DENY,
				Sequence: new(int32(20)),
			},
		},
		{
			description: "with match and set",
			input: &Model{
				Action:   types.StringValue("PERMIT"),
				Sequence: types.Int32Null(),
				Match: &MatchModel{
					AsPathContainsAny: types.ListValueMust(types.Int64Type, []attr.Value{
						types.Int64Value(65001),
					}),
					Communities: types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("65000:100"),
					}),
					FirstAsn:        types.Int64Value(65001),
					MaxPrefixLength: types.Int32Value(24),
					MinPrefixLength: types.Int32Value(16),
					Peer:            types.StringValue("192.0.2.1"),
					Prefixes: types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("10.0.0.0/16"),
					}),
				},
				Set: &SetModel{
					LocalPreference: types.Int32Value(150),
				},
			},
			expected: &vpn.CreateGatewayBGPFilterRulePayload{
				Action: vpn.CREATEGATEWAYBGPFILTERRULEPAYLOADACTION_PERMIT,
				Match: &vpn.CreateGatewayBGPFilterRulePayloadMatch{
					AsPathContainsAny: []int64{65001},
					Communities:       []string{"65000:100"},
					FirstASN:          new(int64(65001)),
					MaxPrefixLength:   new(int32(24)),
					MinPrefixLength:   new(int32(16)),
					Peer:              new("192.0.2.1"),
					Prefixes:          []string{"10.0.0.0/16"},
				},
				Set: &vpn.CreateGatewayBGPFilterRulePayloadSet{
					LocalPreference: new(int32(150)),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toCreatePayload(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if diff := cmp.Diff(tt.expected, payload); diff != "" {
				t.Fatalf("Data does not match (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *vpn.UpdateGatewayBGPFilterRulePayload
	}{
		{
			description: "sequence always sent even when unchanged",
			input: &Model{
				Action:   types.StringValue("PERMIT"),
				Sequence: types.Int32Value(10),
			},
			expected: &vpn.UpdateGatewayBGPFilterRulePayload{
				Action:   vpn.UPDATEGATEWAYBGPFILTERRULEPAYLOADACTION_PERMIT,
				Sequence: new(int32(10)),
			},
		},
		{
			description: "with match and set",
			input: &Model{
				Action:   types.StringValue("DENY"),
				Sequence: types.Int32Value(30),
				Match: &MatchModel{
					Peer: types.StringValue("192.0.2.2"),
				},
				Set: &SetModel{
					LocalPreference: types.Int32Value(200),
				},
			},
			expected: &vpn.UpdateGatewayBGPFilterRulePayload{
				Action:   vpn.UPDATEGATEWAYBGPFILTERRULEPAYLOADACTION_DENY,
				Sequence: new(int32(30)),
				Match: &vpn.UpdateGatewayBGPFilterRulePayloadMatch{
					Peer: new("192.0.2.2"),
				},
				Set: &vpn.UpdateGatewayBGPFilterRulePayloadSet{
					LocalPreference: new(int32(200)),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toUpdatePayload(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if diff := cmp.Diff(tt.expected, payload); diff != "" {
				t.Fatalf("Data does not match (-want +got):\n%s", diff)
			}
		})
	}
}
