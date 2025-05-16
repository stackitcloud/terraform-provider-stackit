package dns

import (
	"context"
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
		state       Model
		input       *dns.RecordSetResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				ProjectId: types.StringValue("pid"),
				ZoneId:    types.StringValue("zid"),
			},
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
				FQDN:        types.StringNull(),
				Records:     types.ListNull(types.StringType),
				State:       types.StringValue(""),
				TTL:         types.Int64Null(),
				Type:        types.StringValue(""),
			},
			true,
		},
		{
			"simple_values",
			Model{
				ProjectId: types.StringValue("pid"),
				ZoneId:    types.StringValue("zid"),
			},
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
					State: dns.RECORDSETSTATE_CREATING.Ptr(),
					Ttl:   utils.Ptr(int64(1)),
					Type:  dns.RECORDSETTYPE_A.Ptr(),
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
				FQDN:        types.StringValue("name"),
				Records: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("record_1"),
					types.StringValue("record_2"),
				}),
				State: types.StringValue(string(dns.RECORDSETSTATE_CREATING)),
				TTL:   types.Int64Value(1),
				Type:  types.StringValue(string(dns.RECORDSETTYPE_A)),
			},
			true,
		},
		{
			"unordered_records",
			Model{
				ProjectId: types.StringValue("pid"),
				ZoneId:    types.StringValue("zid"),
				Records: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("record_2"),
					types.StringValue("record_1"),
				}),
			},
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
					State: dns.RECORDSETSTATE_CREATING.Ptr(),
					Ttl:   utils.Ptr(int64(1)),
					Type:  dns.RECORDSETTYPE_A.Ptr(),
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
				FQDN:        types.StringValue("name"),
				Records: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("record_2"),
					types.StringValue("record_1"),
				}),
				State: types.StringValue(string(dns.RECORDSETSTATE_CREATING)),
				TTL:   types.Int64Value(1),
				Type:  types.StringValue(string(dns.RECORDSETTYPE_A)),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			Model{
				ProjectId: types.StringValue("pid"),
				ZoneId:    types.StringValue("zid"),
				Name:      types.StringValue("other-name"),
			},
			&dns.RecordSetResponse{
				Rrset: &dns.RecordSet{
					Id:      utils.Ptr("rid"),
					Active:  nil,
					Comment: nil,
					Error:   nil,
					Name:    utils.Ptr("name"),
					Records: nil,
					State:   dns.RECORDSETSTATE_CREATING.Ptr(),
					Ttl:     utils.Ptr(int64(2123456789)),
					Type:    dns.RECORDSETTYPE_A.Ptr(),
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
				Name:        types.StringValue("other-name"),
				FQDN:        types.StringValue("name"),
				Records:     types.ListNull(types.StringType),
				State:       types.StringValue(string(dns.RECORDSETSTATE_CREATING)),
				TTL:         types.Int64Value(2123456789),
				Type:        types.StringValue(string(dns.RECORDSETTYPE_A)),
			},
			true,
		},
		{
			"nil_response",
			Model{
				ProjectId: types.StringValue("pid"),
				ZoneId:    types.StringValue("zid"),
			},
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{
				ProjectId: types.StringValue("pid"),
				ZoneId:    types.StringValue("zid"),
			},
			&dns.RecordSetResponse{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state)
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
				Type: types.StringValue(string(dns.RECORDSETTYPE_A)),
			},
			&dns.CreateRecordSetPayload{
				Comment: utils.Ptr("comment"),
				Name:    utils.Ptr("name"),
				Records: &[]dns.RecordPayload{
					{Content: utils.Ptr("record_1")},
					{Content: utils.Ptr("record_2")},
				},
				Ttl:  utils.Ptr(int64(1)),
				Type: dns.CREATERECORDSETPAYLOADTYPE_A.Ptr(),
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
				Type:    types.StringValue(string(dns.RECORDSETTYPE_A)),
			},
			&dns.CreateRecordSetPayload{
				Comment: nil,
				Name:    utils.Ptr(""),
				Records: &[]dns.RecordPayload{},
				Ttl:     utils.Ptr(int64(2123456789)),
				Type:    dns.CREATERECORDSETPAYLOADTYPE_A.Ptr(),
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
		expected    *dns.PartialUpdateRecordSetPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{},
			&dns.PartialUpdateRecordSetPayload{
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
			&dns.PartialUpdateRecordSetPayload{
				Comment: utils.Ptr("comment"),
				Name:    utils.Ptr("name"),
				Records: &[]dns.RecordPayload{
					{Content: utils.Ptr("record_1")},
					{Content: utils.Ptr("record_2")},
				},
				Ttl: utils.Ptr(int64(1)),
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
			&dns.PartialUpdateRecordSetPayload{
				Comment: nil,
				Name:    utils.Ptr(""),
				Records: &[]dns.RecordPayload{},
				Ttl:     utils.Ptr(int64(2123456789)),
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
