package iplists

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	lbiplists "github.com/stackitcloud/stackit-sdk-go/services/lbiplists/v1alphaapi"
)

const testRegion = "eu01"

var testProjectId = uuid.NewString()

func fixtureIPListResponse(mods ...func(m *lbiplists.GetIPListResponse)) *lbiplists.GetIPListResponse {
	resp := &lbiplists.GetIPListResponse{
		Name:        "test-ip-list",
		ContentHash: new("5b2ed459546d7f3e1eeef5c2a07b1b1b50db749b3e04a3bc72d146fe51f1e21c"),
		NumberOfIps: new(int32(2)),
		Labels:      &map[string]string{"key": "value"},
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func fixtureModel(mods ...func(m *Model)) Model {
	resp := Model{
		Id:                 types.StringValue(fmt.Sprintf("%s,%s,%s", testProjectId, testRegion, "test-ip-list")),
		ProjectId:          types.StringValue(testProjectId),
		Region:             types.StringValue(testRegion),
		Name:               types.StringValue("test-ip-list"),
		Labels:             types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
		NumberOfIPs:        types.Int32Value(2),
		ContentHash:        types.StringValue("5b2ed459546d7f3e1eeef5c2a07b1b1b50db749b3e04a3bc72d146fe51f1e21c"),
		FileContentVersion: types.Int64Value(1),
	}
	for _, mod := range mods {
		mod(&resp)
	}
	return resp
}

func fixtureUploadPayload(mods ...func(m *lbiplists.UploadIPListPayload)) *lbiplists.UploadIPListPayload {
	resp := &lbiplists.UploadIPListPayload{
		Name:        "test-ip-list",
		FileContent: "10.0.0.0/16\n192.168.0.0/24",
		Labels:      &map[string]string{"key": "value"},
	}
	for _, mod := range mods {
		mod(resp)
	}
	return resp
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *lbiplists.GetIPListResponse
		state       *Model
		expected    Model
		isValid     bool
	}{
		{
			description: "basic_ip_list",
			input:       fixtureIPListResponse(),
			state: &Model{
				ProjectId:          types.StringValue(testProjectId),
				FileContentVersion: types.Int64Value(1),
			},
			expected: fixtureModel(),
			isValid:  true,
		},
		{
			description: "minimal_ip_list",
			input: &lbiplists.GetIPListResponse{
				Name: "test-ip-list",
			},
			state: &Model{
				ProjectId:          types.StringValue(testProjectId),
				FileContentVersion: types.Int64Value(1),
			},
			expected: fixtureModel(func(m *Model) {
				m.Labels = types.MapNull(types.StringType)
				m.NumberOfIPs = types.Int32Null()
				m.ContentHash = types.StringNull()
			}),
			isValid: true,
		},
		{
			description: "empty_labels",
			input: fixtureIPListResponse(func(m *lbiplists.GetIPListResponse) {
				m.Labels = &map[string]string{}
			}),
			state: &Model{
				ProjectId:          types.StringValue(testProjectId),
				FileContentVersion: types.Int64Value(1),
			},
			expected: fixtureModel(func(m *Model) {
				m.Labels = types.MapNull(types.StringType)
			}),
			isValid: true,
		},
		{
			description: "name_falls_back_to_model",
			input: fixtureIPListResponse(func(m *lbiplists.GetIPListResponse) {
				m.Name = ""
			}),
			state: &Model{
				ProjectId:          types.StringValue(testProjectId),
				Name:               types.StringValue("test-ip-list"),
				FileContentVersion: types.Int64Value(1),
			},
			expected: fixtureModel(),
			isValid:  true,
		},
		{
			description: "nil_response",
			input:       nil,
			state: &Model{
				ProjectId:          types.StringValue(testProjectId),
				FileContentVersion: types.Int64Value(1),
			},
			expected: Model{},
			isValid:  false,
		},
		{
			description: "missing_name",
			input: fixtureIPListResponse(func(m *lbiplists.GetIPListResponse) {
				m.Name = ""
			}),
			state: &Model{
				ProjectId:          types.StringValue(testProjectId),
				FileContentVersion: types.Int64Value(1),
			},
			expected: Model{},
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, tt.state, testRegion)

			if !tt.isValid && err == nil {
				t.Fatalf("expected error, got none")
			}
			if tt.isValid && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.isValid {
				if diff := cmp.Diff(&tt.expected, tt.state); diff != "" {
					t.Fatalf("Data mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestToUploadPayload(t *testing.T) {
	type args struct {
		planModel   *Model
		configModel *Model
	}

	tests := []struct {
		description string
		args        args
		expected    *lbiplists.UploadIPListPayload
		isValid     bool
	}{
		{
			description: "basic_ip_list",
			args: args{
				planModel: new(fixtureModel(func(m *Model) {
					// write-only fields are always null in the plan and state model - they are/should only be present in the config model
					m.FileContent = types.StringNull()
				})),
				configModel: &Model{
					FileContent: types.StringValue("10.0.0.0/16\n192.168.0.0/24"),
				},
			},
			expected: fixtureUploadPayload(),
			isValid:  true,
		},
		{
			description: "no_labels",
			args: args{
				planModel: new(fixtureModel(func(m *Model) {
					m.FileContent = types.StringNull()
					m.Labels = types.MapNull(types.StringType)
				})),
				configModel: &Model{
					FileContent: types.StringValue("10.0.0.0/16\n192.168.0.0/24"),
				},
			},
			expected: fixtureUploadPayload(func(m *lbiplists.UploadIPListPayload) {
				m.Labels = &map[string]string{}
			}),
			isValid: true,
		},
		{
			description: "plan model is nil",
			args: args{
				planModel:   nil,
				configModel: &Model{},
			},
			expected: nil,
			isValid:  false,
		},
		{
			description: "config model is nil",
			args: args{
				planModel:   &Model{},
				configModel: nil,
			},
			expected: nil,
			isValid:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			payload, err := toUploadPayload(context.Background(), tt.args.planModel, tt.args.configModel)

			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.expected, payload)
				if diff != "" {
					t.Fatalf("Data does not match (-want +got):\n%s", diff)
				}
			}
		})
	}
}
