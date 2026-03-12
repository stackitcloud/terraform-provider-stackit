package dns

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	dns "github.com/stackitcloud/stackit-sdk-go/services/dns/v1api"
	"github.com/stackitcloud/stackit-sdk-go/services/dns/v1api/wait"
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
				Name:      types.StringValue("name"),
				DnsName:   types.StringValue("dnsname"),
			},
			&dns.ZoneResponse{
				Zone: dns.Zone{
					Id:                "zid",
					Name:              "name",
					DnsName:           "dnsname",
					Acl:               "1.2.3.4/32",
					DefaultTTL:        3600,
					ExpireTime:        12000,
					NegativeCache:     60,
					PrimaryNameServer: "example.com",
					RefreshTime:       600,
					RetryTime:         300,
					SerialNumber:      1,
				},
			},
			Model{
				Id:                types.StringValue("pid,zid"),
				ProjectId:         types.StringValue("pid"),
				ZoneId:            types.StringValue("zid"),
				Name:              types.StringValue("name"),
				DnsName:           types.StringValue("dnsname"),
				Acl:               types.StringValue("1.2.3.4/32"),
				DefaultTTL:        types.Int32Value(3600),
				ExpireTime:        types.Int32Value(12000),
				RefreshTime:       types.Int32Value(600),
				RetryTime:         types.Int32Value(300),
				SerialNumber:      types.Int32Value(1),
				NegativeCache:     types.Int32Value(60),
				Type:              types.StringValue(""),
				State:             types.StringValue(""),
				PrimaryNameServer: types.StringValue("example.com"),
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
				Zone: dns.Zone{
					Id:                "zid",
					Name:              "name",
					DnsName:           "dnsname",
					Acl:               "acl",
					Active:            utils.Ptr(false),
					CreationStarted:   "bar",
					CreationFinished:  "foo",
					DefaultTTL:        1,
					ExpireTime:        2,
					RefreshTime:       3,
					RetryTime:         4,
					SerialNumber:      5,
					NegativeCache:     6,
					State:             wait.ZONESTATE_CREATING,
					Type:              "primary",
					Primaries:         []string{"primary"},
					PrimaryNameServer: "pns",
					UpdateStarted:     "ufoo",
					UpdateFinished:    "ubar",
					Visibility:        "public",
					Error:             utils.Ptr("error"),
					ContactEmail:      utils.Ptr("a@b.cd"),
					Description:       utils.Ptr("description"),
					IsReverseZone:     utils.Ptr(false),
					RecordCount:       utils.Ptr(int32(3)),
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
				DefaultTTL:        types.Int32Value(1),
				ExpireTime:        types.Int32Value(2),
				RefreshTime:       types.Int32Value(3),
				RetryTime:         types.Int32Value(4),
				SerialNumber:      types.Int32Value(5),
				NegativeCache:     types.Int32Value(6),
				Type:              types.StringValue("primary"),
				State:             types.StringValue(wait.ZONESTATE_CREATING),
				PrimaryNameServer: types.StringValue("pns"),
				Primaries: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("primary"),
				}),
				Visibility:    types.StringValue("public"),
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
				Zone: dns.Zone{
					Id:               "zid",
					Name:             "name",
					DnsName:          "dnsname",
					Acl:              "acl",
					Active:           utils.Ptr(false),
					CreationStarted:  "bar",
					CreationFinished: "foo",
					DefaultTTL:       1,
					ExpireTime:       2,
					RefreshTime:      3,
					RetryTime:        4,
					SerialNumber:     5,
					NegativeCache:    6,
					State:            "creating",
					Type:             "primary",
					Primaries: []string{
						"primary1",
						"primary2",
					},
					PrimaryNameServer: "pns",
					UpdateStarted:     "ufoo",
					UpdateFinished:    "ubar",
					Visibility:        "public",
					Error:             utils.Ptr("error"),
					ContactEmail:      utils.Ptr("a@b.cd"),
					Description:       utils.Ptr("description"),
					IsReverseZone:     utils.Ptr(false),
					RecordCount:       utils.Ptr(int32(3)),
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
				DefaultTTL:        types.Int32Value(1),
				ExpireTime:        types.Int32Value(2),
				RefreshTime:       types.Int32Value(3),
				RetryTime:         types.Int32Value(4),
				SerialNumber:      types.Int32Value(5),
				NegativeCache:     types.Int32Value(6),
				Type:              types.StringValue("primary"),
				State:             types.StringValue("creating"),
				PrimaryNameServer: types.StringValue("pns"),
				Primaries: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("primary2"),
					types.StringValue("primary1"),
				}),
				Visibility:    types.StringValue(string("public")),
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
				Zone: dns.Zone{
					Id:                "zid",
					Name:              "name",
					DnsName:           "dnsname",
					Acl:               "acl",
					Active:            nil,
					CreationStarted:   "bar",
					CreationFinished:  "foo",
					DefaultTTL:        int32(2123456789),
					ExpireTime:        int32(-2),
					RefreshTime:       int32(3),
					RetryTime:         int32(4),
					SerialNumber:      int32(5),
					NegativeCache:     int32(0),
					State:             wait.ZONESTATE_CREATING,
					Type:              "primary",
					Primaries:         nil,
					PrimaryNameServer: "pns",
					UpdateStarted:     "ufoo",
					UpdateFinished:    "ubar",
					Visibility:        "public",
					ContactEmail:      nil,
					Description:       nil,
					IsReverseZone:     nil,
					RecordCount:       utils.Ptr(int32(-2123456789)),
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
				DefaultTTL:        types.Int32Value(2123456789),
				ExpireTime:        types.Int32Value(-2),
				RefreshTime:       types.Int32Value(3),
				RetryTime:         types.Int32Value(4),
				SerialNumber:      types.Int32Value(5),
				NegativeCache:     types.Int32Value(0),
				Type:              types.StringValue("primary"),
				Primaries:         types.ListNull(types.StringType),
				State:             types.StringValue(wait.ZONESTATE_CREATING),
				PrimaryNameServer: types.StringValue("pns"),
				Visibility:        types.StringValue("public"),
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
				ZoneId:    types.StringValue(""),
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
				diff := cmp.Diff(tt.expected, tt.state)
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
				Name:      "Name",
				DnsName:   "DnsName",
				Primaries: []string{},
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
				Type:          types.StringValue("primary"),
				ContactEmail:  types.StringValue("ContactEmail"),
				RetryTime:     types.Int32Value(3),
				RefreshTime:   types.Int32Value(4),
				ExpireTime:    types.Int32Value(5),
				DefaultTTL:    types.Int32Value(4534534),
				NegativeCache: types.Int32Value(-4534534),
				Primaries: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("primary"),
				}),
				IsReverseZone: types.BoolValue(true),
			},
			&dns.CreateZonePayload{
				Name:          "Name",
				DnsName:       "DnsName",
				Acl:           utils.Ptr("Acl"),
				Description:   utils.Ptr("Description"),
				Type:          utils.Ptr("primary"),
				ContactEmail:  utils.Ptr("ContactEmail"),
				Primaries:     []string{"primary"},
				RetryTime:     utils.Ptr(int32(3)),
				RefreshTime:   utils.Ptr(int32(4)),
				ExpireTime:    utils.Ptr(int32(5)),
				DefaultTTL:    utils.Ptr(int32(4534534)),
				NegativeCache: utils.Ptr(int32(-4534534)),
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
				diff := cmp.Diff(tt.expected, output)
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
				Type:              types.StringValue("primary"),
				ContactEmail:      types.StringValue("ContactEmail"),
				PrimaryNameServer: types.StringValue("PrimaryNameServer"),
				Primaries: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("Primary"),
				}),
				RetryTime:     types.Int32Value(3),
				RefreshTime:   types.Int32Value(4),
				ExpireTime:    types.Int32Value(5),
				DefaultTTL:    types.Int32Value(4534534),
				NegativeCache: types.Int32Value(-4534534),
				IsReverseZone: types.BoolValue(true),
			},
			&dns.PartialUpdateZonePayload{
				Name:          utils.Ptr("Name"),
				Acl:           utils.Ptr("Acl"),
				Description:   utils.Ptr("Description"),
				ContactEmail:  utils.Ptr("ContactEmail"),
				RetryTime:     utils.Ptr(int32(3)),
				RefreshTime:   utils.Ptr(int32(4)),
				ExpireTime:    utils.Ptr(int32(5)),
				DefaultTTL:    utils.Ptr(int32(4534534)),
				NegativeCache: utils.Ptr(int32(-4534534)),
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
