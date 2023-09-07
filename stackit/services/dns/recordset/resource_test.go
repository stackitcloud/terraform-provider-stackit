package dns

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/dns"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *dns.RecordSetResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&dns.RecordSetResponse{
				Rrset: &dns.RecordSet{
					Id: utils.Ptr("rid"),
				},
			},
			Model{
				Id:          types.StringValue("pid,zid,rid"),
				RecordSetId: types.StringValue("rid"),
				ZoneId:      types.StringValue("zid"),
				ProjectId:   types.StringValue("pid"),
				Active:      types.BoolNull(),
				Comment:     types.StringNull(),
				Error:       types.StringNull(),
				Name:        types.StringNull(),
				Records:     types.ListNull(types.StringType),
				State:       types.StringNull(),
				TTL:         types.Int64Null(),
				Type:        types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&dns.RecordSetResponse{
				Rrset: &dns.RecordSet{
					Id:      utils.Ptr("rid"),
					Active:  utils.Ptr(true),
					Comment: utils.Ptr("comment"),
					Error:   utils.Ptr("error"),
					Name:    utils.Ptr("name"),
					Records: &[]dns.Record{
						{Content: utils.Ptr("record_1")},
						{Content: utils.Ptr("record_2")},
					},
					State: utils.Ptr("state"),
					Ttl:   utils.Ptr(int32(1)),
					Type:  utils.Ptr("type"),
				},
			},
			Model{
				Id:          types.StringValue("pid,zid,rid"),
				RecordSetId: types.StringValue("rid"),
				ZoneId:      types.StringValue("zid"),
				ProjectId:   types.StringValue("pid"),
				Active:      types.BoolValue(true),
				Comment:     types.StringValue("comment"),
				Error:       types.StringValue("error"),
				Name:        types.StringValue("name"),
				Records: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("record_1"),
					types.StringValue("record_2"),
				}),
				State: types.StringValue("state"),
				TTL:   types.Int64Value(1),
				Type:  types.StringValue("type"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&dns.RecordSetResponse{
				Rrset: &dns.RecordSet{
					Id:      utils.Ptr("rid"),
					Active:  nil,
					Comment: nil,
					Error:   nil,
					Name:    utils.Ptr("name"),
					Records: nil,
					State:   utils.Ptr("state"),
					Ttl:     utils.Ptr(int32(2123456789)),
					Type:    utils.Ptr("type"),
				},
			},
			Model{
				Id:          types.StringValue("pid,zid,rid"),
				RecordSetId: types.StringValue("rid"),
				ZoneId:      types.StringValue("zid"),
				ProjectId:   types.StringValue("pid"),
				Active:      types.BoolNull(),
				Comment:     types.StringNull(),
				Error:       types.StringNull(),
				Name:        types.StringValue("name"),
				Records:     types.ListNull(types.StringType),
				State:       types.StringValue("state"),
				TTL:         types.Int64Value(2123456789),
				Type:        types.StringValue("type"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&dns.RecordSetResponse{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId: tt.expected.ProjectId,
				ZoneId:    tt.expected.ZoneId,
			}
			err := mapFields(tt.input, state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected)
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
		expected    *dns.CreateRecordSetPayload
		isValid     bool
	}{
		{
			"default values",
			&Model{},
			&dns.CreateRecordSetPayload{
				Records: &[]dns.RecordPayload{},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				Comment: types.StringValue("comment"),
				Name:    types.StringValue("name"),
				Records: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("record_1"),
					types.StringValue("record_2"),
				}),
				TTL:  types.Int64Value(1),
				Type: types.StringValue("type"),
			},
			&dns.CreateRecordSetPayload{
				Comment: utils.Ptr("comment"),
				Name:    utils.Ptr("name"),
				Records: &[]dns.RecordPayload{
					{Content: utils.Ptr("record_1")},
					{Content: utils.Ptr("record_2")},
				},
				Ttl:  utils.Ptr(int32(1)),
				Type: utils.Ptr("type"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Comment: types.StringNull(),
				Name:    types.StringValue(""),
				Records: types.ListValueMust(types.StringType, nil),
				TTL:     types.Int64Value(2123456789),
				Type:    types.StringValue(""),
			},
			&dns.CreateRecordSetPayload{
				Comment: nil,
				Name:    utils.Ptr(""),
				Records: &[]dns.RecordPayload{},
				Ttl:     utils.Ptr(int32(2123456789)),
				Type:    utils.Ptr(""),
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)
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
		expected    *dns.UpdateRecordSetPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&dns.UpdateRecordSetPayload{
				Records: &[]dns.RecordPayload{},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				Comment: types.StringValue("comment"),
				Name:    types.StringValue("name"),
				Records: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("record_1"),
					types.StringValue("record_2"),
				}),
				TTL: types.Int64Value(1),
			},
			&dns.UpdateRecordSetPayload{
				Comment: utils.Ptr("comment"),
				Name:    utils.Ptr("name"),
				Records: &[]dns.RecordPayload{
					{Content: utils.Ptr("record_1")},
					{Content: utils.Ptr("record_2")},
				},
				Ttl: utils.Ptr(int32(1)),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Comment: types.StringNull(),
				Name:    types.StringValue(""),
				Records: types.ListValueMust(types.StringType, nil),
				TTL:     types.Int64Value(2123456789),
			},
			&dns.UpdateRecordSetPayload{
				Comment: nil,
				Name:    utils.Ptr(""),
				Records: &[]dns.RecordPayload{},
				Ttl:     utils.Ptr(int32(2123456789)),
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
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
