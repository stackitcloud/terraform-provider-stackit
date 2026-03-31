package dns

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	dns "github.com/stackitcloud/stackit-sdk-go/services/dns/v1api"
	"github.com/stackitcloud/stackit-sdk-go/services/dns/v1api/wait"
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
				Name:      types.StringValue("rname"),
			},
			&dns.RecordSetResponse{
				Rrset: dns.RecordSet{
					Id:   "rid",
					Name: "rname",
					Ttl:  120,
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
				Name:        types.StringValue("rname"),
				FQDN:        types.StringValue("rname"),
				Records:     types.ListNull(types.StringType),
				State:       types.StringValue(""),
				TTL:         types.Int32Value(120),
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
				Rrset: dns.RecordSet{
					Id:      "rid",
					Active:  new(true),
					Comment: new("comment"),
					Error:   new("error"),
					Name:    "name",
					Records: []dns.Record{
						{Content: "record_1"},
						{Content: "record_2"},
					},
					State: wait.RECORDSETSTATE_CREATING,
					Ttl:   1,
					Type:  "A",
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
				State: types.StringValue(wait.RECORDSETSTATE_CREATING),
				TTL:   types.Int32Value(1),
				Type:  types.StringValue("A"),
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
				Rrset: dns.RecordSet{
					Id:      "rid",
					Active:  new(true),
					Comment: new("comment"),
					Error:   new("error"),
					Name:    "name",
					Records: []dns.Record{
						{Content: "record_1"},
						{Content: "record_2"},
					},
					State: wait.RECORDSETSTATE_CREATING,
					Ttl:   1,
					Type:  "A",
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
				State: types.StringValue(wait.RECORDSETSTATE_CREATING),
				TTL:   types.Int32Value(1),
				Type:  types.StringValue("A"),
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
				Rrset: dns.RecordSet{
					Id:      "rid",
					Active:  nil,
					Comment: nil,
					Error:   nil,
					Name:    "name",
					Records: nil,
					State:   wait.RECORDSETSTATE_CREATING,
					Ttl:     2123456789,
					Type:    "A",
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
				State:       types.StringValue(wait.RECORDSETSTATE_CREATING),
				TTL:         types.Int32Value(2123456789),
				Type:        types.StringValue("A"),
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
				Records: []dns.RecordPayload{},
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
				TTL:  types.Int32Value(1),
				Type: types.StringValue("A"),
			},
			&dns.CreateRecordSetPayload{
				Comment: new("comment"),
				Name:    "name",
				Records: []dns.RecordPayload{
					{Content: "record_1"},
					{Content: "record_2"},
				},
				Ttl:  new(int32(1)),
				Type: "A",
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Comment: types.StringNull(),
				Name:    types.StringValue(""),
				Records: types.ListValueMust(types.StringType, nil),
				TTL:     types.Int32Value(2123456789),
				Type:    types.StringValue("A"),
			},
			&dns.CreateRecordSetPayload{
				Comment: nil,
				Name:    "",
				Records: []dns.RecordPayload{},
				Ttl:     new(int32(2123456789)),
				Type:    "A",
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
				Records: []dns.RecordPayload{},
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
				TTL: types.Int32Value(1),
			},
			&dns.PartialUpdateRecordSetPayload{
				Comment: new("comment"),
				Name:    new("name"),
				Records: []dns.RecordPayload{
					{Content: "record_1"},
					{Content: "record_2"},
				},
				Ttl: new(int32(1)),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				Comment: types.StringNull(),
				Name:    types.StringValue(""),
				Records: types.ListValueMust(types.StringType, nil),
				TTL:     types.Int32Value(2123456789),
			},
			&dns.PartialUpdateRecordSetPayload{
				Comment: nil,
				Name:    new(""),
				Records: []dns.RecordPayload{},
				Ttl:     new(int32(2123456789)),
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
