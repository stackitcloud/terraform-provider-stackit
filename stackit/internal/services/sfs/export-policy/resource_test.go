package exportpolicy

import (
	"context"
	_ "embed"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
)

// global stuff
var project_id = "project_id"

func fixtureRulesResponse() *[]sfs.ShareExportPolicyRule {
	return &[]sfs.ShareExportPolicyRule{
		{
			Description: sfs.NewNullableString(utils.Ptr("description")),
			IpAcl:       utils.Ptr([]string{"172.16.0.0/24", "172.16.0.251/32"}),
			Order:       utils.Ptr(int64(0)),
			ReadOnly:    utils.Ptr(false),
			SetUuid:     utils.Ptr(false),
			SuperUser:   utils.Ptr(false),
		},
		{
			Description: sfs.NewNullableString(utils.Ptr("description")),
			IpAcl:       utils.Ptr([]string{"172.32.0.0/24", "172.32.0.251/32"}),
			Order:       utils.Ptr(int64(1)),
			ReadOnly:    utils.Ptr(false),
			SetUuid:     utils.Ptr(false),
			SuperUser:   utils.Ptr(false),
		},
	}
}

func fixtureRulesModel() basetypes.ListValue {
	// create the list
	return types.ListValueMust(types.ObjectType{AttrTypes: rulesTypes}, []attr.Value{
		types.ObjectValueMust(rulesTypes, map[string]attr.Value{
			"description": types.StringValue("description"),
			"ip_acl": types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("172.16.0.0/24"),
				types.StringValue("172.16.0.251/32"),
			}),
			"order":      types.Int64Value(0),
			"read_only":  types.BoolValue(false),
			"set_uuid":   types.BoolValue(false),
			"super_user": types.BoolValue(false),
		}),
		types.ObjectValueMust(rulesTypes, map[string]attr.Value{
			"description": types.StringValue("description"),
			"ip_acl": types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("172.32.0.0/24"),
				types.StringValue("172.32.0.251/32"),
			}),
			"order":      types.Int64Value(1),
			"read_only":  types.BoolValue(false),
			"set_uuid":   types.BoolValue(false),
			"super_user": types.BoolValue(false),
		}),
	})
}

func fixtureResponseModel(rulesModel basetypes.ListValue) *Model {
	return &Model{
		ProjectId:      types.StringValue(project_id),
		Id:             types.StringValue(project_id + ",region,uuid1"),
		ExportPolicyId: types.StringValue("uuid1"),
		Rules:          rulesModel,
		Region:         types.StringValue("region"),
	}
}

func fixtureRulesCreatePayload() []sfs.CreateShareExportPolicyRequestRule {
	return []sfs.CreateShareExportPolicyRequestRule{
		{
			Description: sfs.NewNullableString(utils.Ptr("description")),
			IpAcl: &[]string{
				"172.32.0.0/24",
				"172.32.0.251/32",
			},
			Order:     utils.Ptr(int64(0)),
			ReadOnly:  utils.Ptr(false),
			SetUuid:   utils.Ptr(false),
			SuperUser: utils.Ptr(false),
		},
		{
			Description: sfs.NewNullableString(utils.Ptr("description")),
			IpAcl: &[]string{
				"172.16.0.0/24",
				"172.16.0.251/32",
			},
			Order:     utils.Ptr(int64(1)),
			ReadOnly:  utils.Ptr(false),
			SetUuid:   utils.Ptr(false),
			SuperUser: utils.Ptr(false),
		},
	}
}

func fixtureRulesUpdatePayload() []sfs.UpdateShareExportPolicyBodyRule {
	return []sfs.UpdateShareExportPolicyBodyRule{
		{
			Description: sfs.NewNullableString(utils.Ptr("description")),
			IpAcl: &[]string{
				"172.32.0.0/24",
				"172.32.0.251/32",
			},
			Order:     utils.Ptr(int64(0)),
			ReadOnly:  utils.Ptr(false),
			SetUuid:   utils.Ptr(false),
			SuperUser: utils.Ptr(false),
		},
		{
			Description: sfs.NewNullableString(utils.Ptr("description")),
			IpAcl: &[]string{
				"172.16.0.0/24",
				"172.16.0.251/32",
			},
			Order:     utils.Ptr(int64(1)),
			ReadOnly:  utils.Ptr(false),
			SetUuid:   utils.Ptr(false),
			SuperUser: utils.Ptr(false),
		},
	}
}

func fixtureRulesPayloadModel() []rulesModel {
	return []rulesModel{
		{
			Description: types.StringValue("description"),
			IpAcl:       types.ListValueMust(types.StringType, []attr.Value{types.StringValue("172.32.0.0/24"), types.StringValue("172.32.0.251/32")}),
			Order:       types.Int64Value(0),
			ReadOnly:    types.BoolValue(false),
			SetUuid:     types.BoolValue(false),
			SuperUser:   types.BoolValue(false),
		},
		{
			Description: types.StringValue("description"),
			IpAcl:       types.ListValueMust(types.StringType, []attr.Value{types.StringValue("172.16.0.0/24"), types.StringValue("172.16.0.251/32")}),
			Order:       types.Int64Value(1),
			ReadOnly:    types.BoolValue(false),
			SetUuid:     types.BoolValue(false),
			SuperUser:   types.BoolValue(false),
		},
	}
}

func fixtureExportPolicyCreatePayload(rules *[]sfs.CreateShareExportPolicyRequestRule) *sfs.CreateShareExportPolicyPayload {
	return &sfs.CreateShareExportPolicyPayload{
		Name:  utils.Ptr("createPayloadName"),
		Rules: rules,
	}
}

func fixtureExportPolicyUpdatePayload(rules []sfs.UpdateShareExportPolicyBodyRule) *sfs.UpdateShareExportPolicyPayload {
	return &sfs.UpdateShareExportPolicyPayload{
		Rules: &rules,
	}
}

func TestMapFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		name          string
		input         *sfs.GetShareExportPolicyResponse
		state         *Model
		expectedModel *Model
		isValid       bool
		region        string
	}{
		{
			name: "resp is nil",
			state: &Model{
				ProjectId: types.StringValue(project_id),
			},
			input:   nil,
			region:  testRegion,
			isValid: false,
		},
		{
			name: "shared export policy in response is nil",
			state: &Model{
				ProjectId: types.StringValue(project_id),
			},
			input:   &sfs.GetShareExportPolicyResponse{},
			region:  testRegion,
			isValid: false,
		},
		{
			name: "rules list is empty",
			state: &Model{
				ProjectId: types.StringValue(project_id),
			},
			input: &sfs.GetShareExportPolicyResponse{
				ShareExportPolicy: &sfs.GetShareExportPolicyResponseShareExportPolicy{
					Id:    utils.Ptr("uuid1"),
					Rules: &[]sfs.ShareExportPolicyRule{},
				},
			},
			expectedModel: fixtureResponseModel(types.ListValueMust(types.ObjectType{AttrTypes: rulesTypes}, []attr.Value{})),
			region:        testRegion,
			isValid:       true,
		},
		{
			name: "normal data",
			state: &Model{
				ProjectId: types.StringValue(project_id),
			},
			input: &sfs.GetShareExportPolicyResponse{
				ShareExportPolicy: &sfs.GetShareExportPolicyResponseShareExportPolicy{
					Id:    utils.Ptr("uuid1"),
					Rules: fixtureRulesResponse(),
				},
			},
			expectedModel: fixtureResponseModel(fixtureRulesModel()),
			region:        testRegion,
			isValid:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, tt.state, tt.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.state, tt.expectedModel)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		name     string
		model    *Model
		rules    []rulesModel
		expected *sfs.CreateShareExportPolicyPayload
		wantErr  bool
	}{
		{
			name: "nil rules",
			model: &Model{
				ProjectId: types.StringValue(project_id),
				Name:      types.StringValue("createPayloadName"),
			},
			rules:   nil,
			wantErr: true,
		},
		{
			name:    "nil model",
			model:   nil,
			rules:   []rulesModel{},
			wantErr: true,
		},
		{
			name: "empty rule payload",
			model: &Model{
				ProjectId: types.StringValue(project_id),
				Name:      types.StringValue("createPayloadName"),
			},
			rules:    []rulesModel{},
			expected: fixtureExportPolicyCreatePayload(nil),
			wantErr:  false,
		},
		{
			name: "valid rule payload",
			model: &Model{
				ProjectId: types.StringValue(project_id),
				Name:      types.StringValue("createPayloadName"),
			},
			rules:    fixtureRulesPayloadModel(),
			expected: fixtureExportPolicyCreatePayload(utils.Ptr(fixtureRulesCreatePayload())),
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toCreatePayload(tt.model, tt.rules)
			if (err != nil) != tt.wantErr {
				t.Errorf("toCreatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("toCreatePayload() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		name     string
		model    *Model
		rules    []rulesModel
		expected *sfs.UpdateShareExportPolicyPayload
		wantErr  bool
	}{
		{
			name: "nil rules",
			model: &Model{
				ProjectId: types.StringValue(project_id),
				Name:      types.StringValue("updatePayloadName"),
			},
			rules:   nil,
			wantErr: true,
		},
		{
			name:    "nil model",
			model:   nil,
			rules:   []rulesModel{},
			wantErr: true,
		},
		{
			name: "empty rule payload",
			model: &Model{
				ProjectId: types.StringValue(project_id),
				Name:      types.StringValue("createPayloadName"),
			},
			rules:    []rulesModel{},
			expected: fixtureExportPolicyUpdatePayload([]sfs.UpdateShareExportPolicyBodyRule{}),
			wantErr:  false,
		},
		{
			name: "valid rule payload",
			model: &Model{
				ProjectId: types.StringValue(project_id),
				Name:      types.StringValue("createPayloadName"),
			},
			rules:    fixtureRulesPayloadModel(),
			expected: fixtureExportPolicyUpdatePayload(fixtureRulesUpdatePayload()),
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toUpdatePayload(tt.model, tt.rules)
			if (err != nil) != tt.wantErr {
				t.Errorf("toUpdatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("toUpdatePayload() = %v, want %v", got, tt.expected)
			}
		})
	}
}
