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
		input       *dns.ZoneResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_ok",
			Model{
				ProjectId: types.StringValue("pid"),
				ZoneId:    types.StringValue("zid"),
			},
			&dns.ZoneResponse{
				Zone: &dns.Zone{
					Id: utils.Ptr("zid"),
				},
			},
			Model{
				Id:                types.StringValue("pid,zid"),
				ProjectId:         types.StringValue("pid"),
				ZoneId:            types.StringValue("zid"),
				Name:              types.StringNull(),
				DnsName:           types.StringNull(),
				Acl:               types.StringNull(),
				DefaultTTL:        types.Int64Null(),
				ExpireTime:        types.Int64Null(),
				RefreshTime:       types.Int64Null(),
				RetryTime:         types.Int64Null(),
				SerialNumber:      types.Int64Null(),
				NegativeCache:     types.Int64Null(),
				Type:              types.StringValue(""),
				State:             types.StringValue(""),
				PrimaryNameServer: types.StringNull(),
				Primaries:         types.ListNull(types.StringType),
				Visibility:        types.StringValue(""),
			},
			true,
		},
		{
			"values_ok",
			Model{
				ProjectId: types.StringValue("pid"),
				ZoneId:    types.StringValue("zid"),
			},
			&dns.ZoneResponse{
				Zone: &dns.Zone{
					Id:                utils.Ptr("zid"),
					Name:              utils.Ptr("name"),
					DnsName:           utils.Ptr("dnsname"),
					Acl:               utils.Ptr("acl"),
					Active:            utils.Ptr(false),
					CreationStarted:   utils.Ptr("bar"),
					CreationFinished:  utils.Ptr("foo"),
					DefaultTTL:        utils.Ptr(int64(1)),
					ExpireTime:        utils.Ptr(int64(2)),
					RefreshTime:       utils.Ptr(int64(3)),
					RetryTime:         utils.Ptr(int64(4)),
					SerialNumber:      utils.Ptr(int64(5)),
					NegativeCache:     utils.Ptr(int64(6)),
					State:             dns.ZONESTATE_CREATING.Ptr(),
					Type:              dns.ZONETYPE_PRIMARY.Ptr(),
					Primaries:         &[]string{"primary"},
					PrimaryNameServer: utils.Ptr("pns"),
					UpdateStarted:     utils.Ptr("ufoo"),
					UpdateFinished:    utils.Ptr("ubar"),
					Visibility:        dns.ZONEVISIBILITY_PUBLIC.Ptr(),
					Error:             utils.Ptr("error"),
					ContactEmail:      utils.Ptr("a@b.cd"),
					Description:       utils.Ptr("description"),
					IsReverseZone:     utils.Ptr(false),
					RecordCount:       utils.Ptr(int64(3)),
				},
			},
			Model{
				Id:                types.StringValue("pid,zid"),
				ProjectId:         types.StringValue("pid"),
				ZoneId:            types.StringValue("zid"),
				Name:              types.StringValue("name"),
				DnsName:           types.StringValue("dnsname"),
				Acl:               types.StringValue("acl"),
				Active:            types.BoolValue(false),
				DefaultTTL:        types.Int64Value(1),
				ExpireTime:        types.Int64Value(2),
				RefreshTime:       types.Int64Value(3),
				RetryTime:         types.Int64Value(4),
				SerialNumber:      types.Int64Value(5),
				NegativeCache:     types.Int64Value(6),
				Type:              types.StringValue(string(dns.ZONETYPE_PRIMARY)),
				State:             types.StringValue(string(dns.ZONESTATE_CREATING)),
				PrimaryNameServer: types.StringValue("pns"),
				Primaries: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("primary"),
				}),
				Visibility:    types.StringValue(string(dns.ZONEVISIBILITY_PUBLIC)),
				ContactEmail:  types.StringValue("a@b.cd"),
				Description:   types.StringValue("description"),
				IsReverseZone: types.BoolValue(false),
				RecordCount:   types.Int64Value(3),
			},
			true,
		},
		{
			"primaries_unordered",
			Model{
				ProjectId: types.StringValue("pid"),
				ZoneId:    types.StringValue("zid"),
				Primaries: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("primary2"),
					types.StringValue("primary1"),
				}),
			},
			&dns.ZoneResponse{
				Zone: &dns.Zone{
					Id:               utils.Ptr("zid"),
					Name:             utils.Ptr("name"),
					DnsName:          utils.Ptr("dnsname"),
					Acl:              utils.Ptr("acl"),
					Active:           utils.Ptr(false),
					CreationStarted:  utils.Ptr("bar"),
					CreationFinished: utils.Ptr("foo"),
					DefaultTTL:       utils.Ptr(int64(1)),
					ExpireTime:       utils.Ptr(int64(2)),
					RefreshTime:      utils.Ptr(int64(3)),
					RetryTime:        utils.Ptr(int64(4)),
					SerialNumber:     utils.Ptr(int64(5)),
					NegativeCache:    utils.Ptr(int64(6)),
					State:            dns.ZONESTATE_CREATING.Ptr(),
					Type:             dns.ZONETYPE_PRIMARY.Ptr(),
					Primaries: &[]string{
						"primary1",
						"primary2",
					},
					PrimaryNameServer: utils.Ptr("pns"),
					UpdateStarted:     utils.Ptr("ufoo"),
					UpdateFinished:    utils.Ptr("ubar"),
					Visibility:        dns.ZONEVISIBILITY_PUBLIC.Ptr(),
					Error:             utils.Ptr("error"),
					ContactEmail:      utils.Ptr("a@b.cd"),
					Description:       utils.Ptr("description"),
					IsReverseZone:     utils.Ptr(false),
					RecordCount:       utils.Ptr(int64(3)),
				},
			},
			Model{
				Id:                types.StringValue("pid,zid"),
				ProjectId:         types.StringValue("pid"),
				ZoneId:            types.StringValue("zid"),
				Name:              types.StringValue("name"),
				DnsName:           types.StringValue("dnsname"),
				Acl:               types.StringValue("acl"),
				Active:            types.BoolValue(false),
				DefaultTTL:        types.Int64Value(1),
				ExpireTime:        types.Int64Value(2),
				RefreshTime:       types.Int64Value(3),
				RetryTime:         types.Int64Value(4),
				SerialNumber:      types.Int64Value(5),
				NegativeCache:     types.Int64Value(6),
				Type:              types.StringValue(string(dns.ZONETYPE_PRIMARY)),
				State:             types.StringValue(string(dns.ZONESTATE_CREATING)),
				PrimaryNameServer: types.StringValue("pns"),
				Primaries: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("primary2"),
					types.StringValue("primary1"),
				}),
				Visibility:    types.StringValue(string(dns.ZONEVISIBILITY_PUBLIC)),
				ContactEmail:  types.StringValue("a@b.cd"),
				Description:   types.StringValue("description"),
				IsReverseZone: types.BoolValue(false),
				RecordCount:   types.Int64Value(3),
			},
			true,
		},
		{
			"nullable_fields_and_int_conversions_ok",
			Model{
				Id:        types.StringValue("pid,zid"),
				ProjectId: types.StringValue("pid"),
				ZoneId:    types.StringValue("zid"),
			},
			&dns.ZoneResponse{
				Zone: &dns.Zone{
					Id:                utils.Ptr("zid"),
					Name:              utils.Ptr("name"),
					DnsName:           utils.Ptr("dnsname"),
					Acl:               utils.Ptr("acl"),
					Active:            nil,
					CreationStarted:   utils.Ptr("bar"),
					CreationFinished:  utils.Ptr("foo"),
					DefaultTTL:        utils.Ptr(int64(2123456789)),
					ExpireTime:        utils.Ptr(int64(-2)),
					RefreshTime:       utils.Ptr(int64(3)),
					RetryTime:         utils.Ptr(int64(4)),
					SerialNumber:      utils.Ptr(int64(5)),
					NegativeCache:     utils.Ptr(int64(0)),
					State:             dns.ZONESTATE_CREATING.Ptr(),
					Type:              dns.ZONETYPE_PRIMARY.Ptr(),
					Primaries:         nil,
					PrimaryNameServer: utils.Ptr("pns"),
					UpdateStarted:     utils.Ptr("ufoo"),
					UpdateFinished:    utils.Ptr("ubar"),
					Visibility:        dns.ZONEVISIBILITY_PUBLIC.Ptr(),
					ContactEmail:      nil,
					Description:       nil,
					IsReverseZone:     nil,
					RecordCount:       utils.Ptr(int64(-2123456789)),
				},
			},
			Model{
				Id:                types.StringValue("pid,zid"),
				ProjectId:         types.StringValue("pid"),
				ZoneId:            types.StringValue("zid"),
				Name:              types.StringValue("name"),
				DnsName:           types.StringValue("dnsname"),
				Acl:               types.StringValue("acl"),
				Active:            types.BoolNull(),
				DefaultTTL:        types.Int64Value(2123456789),
				ExpireTime:        types.Int64Value(-2),
				RefreshTime:       types.Int64Value(3),
				RetryTime:         types.Int64Value(4),
				SerialNumber:      types.Int64Value(5),
				NegativeCache:     types.Int64Value(0),
				Type:              types.StringValue(string(dns.ZONETYPE_PRIMARY)),
				Primaries:         types.ListNull(types.StringType),
				State:             types.StringValue(string(dns.ZONESTATE_CREATING)),
				PrimaryNameServer: types.StringValue("pns"),
				Visibility:        types.StringValue(string(dns.ZONEVISIBILITY_PUBLIC)),
				ContactEmail:      types.StringNull(),
				Description:       types.StringNull(),
				IsReverseZone:     types.BoolNull(),
				RecordCount:       types.Int64Value(-2123456789),
			},
			true,
		},
		{
			"response_nil_fail",
			Model{},
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
			&dns.ZoneResponse{},
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
		expected    *dns.CreateZonePayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name:    types.StringValue("Name"),
				DnsName: types.StringValue("DnsName"),
			},
			&dns.CreateZonePayload{
				Name:      utils.Ptr("Name"),
				DnsName:   utils.Ptr("DnsName"),
				Primaries: &[]string{},
			},
			true,
		},
		{
			"mapping_with_conversions_ok",
			&Model{
				Name:          types.StringValue("Name"),
				DnsName:       types.StringValue("DnsName"),
				Acl:           types.StringValue("Acl"),
				Description:   types.StringValue("Description"),
				Type:          types.StringValue(string(dns.CREATEZONEPAYLOADTYPE_PRIMARY)),
				ContactEmail:  types.StringValue("ContactEmail"),
				RetryTime:     types.Int64Value(3),
				RefreshTime:   types.Int64Value(4),
				ExpireTime:    types.Int64Value(5),
				DefaultTTL:    types.Int64Value(4534534),
				NegativeCache: types.Int64Value(-4534534),
				Primaries: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("primary"),
				}),
				IsReverseZone: types.BoolValue(true),
			},
			&dns.CreateZonePayload{
				Name:          utils.Ptr("Name"),
				DnsName:       utils.Ptr("DnsName"),
				Acl:           utils.Ptr("Acl"),
				Description:   utils.Ptr("Description"),
				Type:          dns.CREATEZONEPAYLOADTYPE_PRIMARY.Ptr(),
				ContactEmail:  utils.Ptr("ContactEmail"),
				Primaries:     &[]string{"primary"},
				RetryTime:     utils.Ptr(int64(3)),
				RefreshTime:   utils.Ptr(int64(4)),
				ExpireTime:    utils.Ptr(int64(5)),
				DefaultTTL:    utils.Ptr(int64(4534534)),
				NegativeCache: utils.Ptr(int64(-4534534)),
				IsReverseZone: utils.Ptr(true),
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

func TestToPayloadUpdate(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *dns.PartialUpdateZonePayload
		isValid     bool
	}{
		{
			"single_field_change_ok",
			&Model{
				Name: types.StringValue("Name"),
			},
			&dns.PartialUpdateZonePayload{
				Name: utils.Ptr("Name"),
			},
			true,
		},
		{
			"mapping_with_conversions_ok",
			&Model{
				Name:              types.StringValue("Name"),
				DnsName:           types.StringValue("DnsName"),
				Acl:               types.StringValue("Acl"),
				Active:            types.BoolValue(true),
				Description:       types.StringValue("Description"),
				Type:              types.StringValue(string(dns.ZONETYPE_PRIMARY)),
				ContactEmail:      types.StringValue("ContactEmail"),
				PrimaryNameServer: types.StringValue("PrimaryNameServer"),
				Primaries: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("Primary"),
				}),
				RetryTime:     types.Int64Value(3),
				RefreshTime:   types.Int64Value(4),
				ExpireTime:    types.Int64Value(5),
				DefaultTTL:    types.Int64Value(4534534),
				NegativeCache: types.Int64Value(-4534534),
				IsReverseZone: types.BoolValue(true),
			},
			&dns.PartialUpdateZonePayload{
				Name:          utils.Ptr("Name"),
				Acl:           utils.Ptr("Acl"),
				Description:   utils.Ptr("Description"),
				ContactEmail:  utils.Ptr("ContactEmail"),
				RetryTime:     utils.Ptr(int64(3)),
				RefreshTime:   utils.Ptr(int64(4)),
				ExpireTime:    utils.Ptr(int64(5)),
				DefaultTTL:    utils.Ptr(int64(4534534)),
				NegativeCache: utils.Ptr(int64(-4534534)),
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

func TestDnsNameNoTrailingDot(t *testing.T) {
	tests := []struct {
		description string
		input       string
		match       bool
	}{
		{
			"normal domain without trailing dot",
			"example.com",
			true,
		},
		{
			"single layer without trailing dot",
			"example",
			true,
		},
		{
			"domain with trailing dot",
			"example.com.",
			false,
		},
		{
			"only trailing dot",
			".",
			false,
		},
		{
			"single layer with trailing dot",
			"example.",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got := dnsNameNoTrailingDotRegex.MatchString(tt.input)
			if got != tt.match {
				t.Fatalf("dnsNameNoTrailingDotRegex.MatchString(%q) = %v, want %v", tt.input, got, tt.match)
			}
		})
	}
}
